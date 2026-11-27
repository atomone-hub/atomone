package recoverylock

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/app/recoverylock/core"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

// ErrLocked identifies a refusal by the recovery policy.
var ErrLocked = errorsmod.Register("recoverylock", 1, "recovery lock")

// Lock shares one immutable policy across the ante, bank and dispatch guards.
type Lock struct {
	config     *core.Config
	chainID    string
	cdc        codec.Codec
	withdraw   WithdrawAddrKeeper
	lastHeight func() int64
}

func New(config *core.Config, chainID string, cdc codec.Codec, withdraw WithdrawAddrKeeper, lastHeight func() int64) *Lock {
	return &Lock{config: config, chainID: chainID, cdc: cdc, withdraw: withdraw, lastHeight: lastHeight}
}

func (l *Lock) active(ctx sdk.Context) bool {
	if ctx.ChainID() != l.chainID {
		return false
	}
	height := ctx.BlockHeight()
	if ctx.IsCheckTx() {
		if height == 0 && l.lastHeight != nil {
			height = l.lastHeight()
		}
		height++
		if ctx.BlockTime().IsZero() {
			return l.config.HeightActive(height)
		}
	}
	return l.config.Active(height, ctx.BlockTime().Unix())
}

func (l *Lock) checkSigners(msg sdk.Msg) (listed, allListed bool, err error) {
	// Each top-level Any gets its own SDK budget, including the root.
	// Enforce the same limits on cached messages constructed in memory.
	remaining := codectypes.MaxUnpackAnySubCalls
	return l.checkSignerTree(msg, codectypes.MaxUnpackAnyRecursionDepth, &remaining)
}

func (l *Lock) checkSignerTree(msg sdk.Msg, depth int, remaining *int) (listed, allListed bool, err error) {
	if depth == 0 {
		return false, false, ErrLocked.Wrap("authz nesting exceeds the recovery limit")
	}
	if *remaining == 0 {
		return false, false, ErrLocked.Wrap("authz message count exceeds the recovery limit")
	}
	*remaining -= 1

	if exec, ok := msg.(*authz.MsgExec); ok {
		if exec == nil {
			return false, false, sdkerrors.ErrInvalidRequest.Wrap("nil authz execution")
		}
		// MsgExec declares grantee as its sole signer. Use the same account
		// codec as SDK signing without serializing the entire subtree.
		grantee, err := l.cdc.InterfaceRegistry().SigningContext().AddressCodec().StringToBytes(exec.Grantee)
		if err != nil {
			return false, false, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
		}
		allListed = l.config.IsListed(grantee)
		listed = allListed
		for _, packed := range exec.Msgs {
			if packed == nil {
				return false, false, sdkerrors.ErrInvalidRequest.Wrap("nil authz message")
			}
			inner, ok := packed.GetCachedValue().(sdk.Msg)
			if !ok {
				return false, false, sdkerrors.ErrInvalidRequest.Wrap("unpacked authz message is missing")
			}
			nestedListed, _, err := l.checkSignerTree(inner, depth-1, remaining)
			if err != nil {
				return false, false, err
			}
			listed = listed || nestedListed
		}
		return listed, allListed, nil
	}

	signers, _, err := l.cdc.GetMsgV1Signers(msg)
	if err != nil {
		return false, false, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	allListed = len(signers) > 0
	for _, signer := range signers {
		isListed := l.config.IsListed(signer)
		listed = listed || isListed
		allListed = allListed && isListed
	}
	return listed, allListed, nil
}

// checkRecoveryMessage applies the direct rules after all signers are listed.
func (l *Lock) checkRecoveryMessage(ctx sdk.Context, msg sdk.Msg) error {
	addressCodec := l.cdc.InterfaceRegistry().SigningContext().AddressCodec()
	var delegator string
	switch m := msg.(type) {
	case *banktypes.MsgSend:
		to, err := addressCodec.StringToBytes(m.ToAddress)
		if err == nil && l.config.IsRecovery(to) {
			return nil
		}
		return ErrLocked.Wrap("transaction must contain only direct recovery messages from listed accounts")
	case *stakingtypes.MsgUndelegate:
		delegator = m.DelegatorAddress
	case *stakingtypes.MsgCancelUnbondingDelegation:
		delegator = m.DelegatorAddress
	case *distrtypes.MsgWithdrawDelegatorReward:
		delegator = m.DelegatorAddress
	case *distrtypes.MsgWithdrawValidatorCommission:
		operator, err := l.cdc.InterfaceRegistry().SigningContext().ValidatorAddressCodec().StringToBytes(m.ValidatorAddress)
		if err != nil {
			return err
		}
		return l.safeWithdrawAddr(ctx, sdk.AccAddress(operator))
	case *distrtypes.MsgSetWithdrawAddress:
		wallet, err := addressCodec.StringToBytes(m.DelegatorAddress)
		if err != nil {
			return err
		}
		withdraw, err := addressCodec.StringToBytes(m.WithdrawAddress)
		if err != nil {
			return err
		}
		if !l.isSafeWithdrawAddr(wallet, withdraw) {
			return ErrLocked.Wrap("withdraw address may only be the wallet or the recovery address")
		}
		return nil
	default:
		return ErrLocked.Wrap("transaction must contain only direct recovery messages from listed accounts")
	}

	wallet, err := addressCodec.StringToBytes(delegator)
	if err != nil {
		return err
	}
	return l.safeWithdrawAddr(ctx, sdk.AccAddress(wallet))
}

func (l *Lock) isSafeWithdrawAddr(wallet, withdraw sdk.AccAddress) bool {
	return bytes.Equal(wallet, withdraw) || l.config.IsRecovery(withdraw)
}

func (l *Lock) safeWithdrawAddr(ctx sdk.Context, wallet sdk.AccAddress) error {
	withdraw, err := l.withdraw.GetDelegatorWithdrawAddr(ctx, wallet)
	if err != nil {
		return err
	}
	if !l.isSafeWithdrawAddr(wallet, withdraw) {
		return ErrLocked.Wrapf("withdraw address %s is neither the wallet nor the recovery address; set it first", withdraw)
	}
	return nil
}

// AnteHandle verifies the recovery policy before either stock fee deduction path.
func (l *Lock) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !l.active(ctx) {
		return next(ctx, tx, simulate)
	}
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return next(ctx, tx, simulate)
	}
	involved := false
	recoveryOnly := true
	var recoveryErr error
	for _, msg := range msgs {
		listed, allListed, err := l.checkSigners(msg)
		if err != nil {
			return ctx, err
		}
		involved = involved || listed
		recoveryOnly = recoveryOnly && allListed
		if recoveryOnly && recoveryErr == nil {
			recoveryErr = l.checkRecoveryMessage(ctx, msg)
		}
	}
	feeTx, hasFee := tx.(sdk.FeeTx)
	if hasFee && (l.config.IsListed(feeTx.FeePayer()) || l.config.IsListed(feeTx.FeeGranter())) {
		return ctx, ErrLocked.Wrap("listed accounts cannot pay or grant fees")
	}
	if !involved {
		return next(ctx, tx, simulate)
	}
	if !recoveryOnly {
		return ctx, ErrLocked.Wrap("transaction must contain only direct recovery messages from listed accounts")
	}
	if recoveryErr != nil {
		return ctx, recoveryErr
	}
	// The SDK verifies both signatures. A fallback payer is a listed first
	// signer and was refused above, so the configured payer must be explicit.
	if !hasFee || !l.config.IsFeePayer(feeTx.FeePayer()) || len(feeTx.FeeGranter()) != 0 {
		return ctx, ErrLocked.Wrap("the configured fee payer must pay directly")
	}
	fee := feeTx.GetFee()
	if len(fee) != 1 || fee[0].Denom != photontypes.Denom || !fee[0].IsPositive() {
		return ctx, ErrLocked.Wrap("recovery requires a positive uphoton fee")
	}
	return next(ctx, tx, simulate)
}
