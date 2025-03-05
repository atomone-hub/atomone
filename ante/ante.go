package ante

import (
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	atomoneerrors "github.com/atomone-hub/atomone/types/errors"
	feemarketante "github.com/atomone-hub/atomone/x/feemarket/ante"
	feemarketkeeper "github.com/atomone-hub/atomone/x/feemarket/keeper"
	photonante "github.com/atomone-hub/atomone/x/photon/ante"
	photonkeeper "github.com/atomone-hub/atomone/x/photon/keeper"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions
	AccountKeeper   feemarketante.AccountKeeper
	BankKeeper      feemarketante.BankKeeper
	Codec           codec.BinaryCodec
	IBCkeeper       *ibckeeper.Keeper
	StakingKeeper   *stakingkeeper.Keeper
	PhotonKeeper    *photonkeeper.Keeper
	TxFeeChecker    ante.TxFeeChecker
	FeemarketKeeper *feemarketkeeper.Keeper
}

func NewAnteHandler(opts HandlerOptions) (sdk.AnteHandler, error) {
	if opts.AccountKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if opts.BankKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if opts.SignModeHandler == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "sign mode handler is required for AnteHandler")
	}
	if opts.IBCkeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "IBC keeper is required for AnteHandler")
	}
	if opts.StakingKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrNotFound, "staking param store is required for AnteHandler")
	}
	if opts.PhotonKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrNotFound, "photon keeper is required for AnteHandler")
	}

	sigGasConsumer := opts.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}
	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		NewGovVoteDecorator(opts.Codec, opts.StakingKeeper),
		photonante.NewValidateFeeDecorator(opts.PhotonKeeper),
		feemarketante.NewFeeMarketCheckDecorator( // fee market check replaces fee deduct decorator
			opts.AccountKeeper,
			opts.BankKeeper,
			opts.FeegrantKeeper,
			opts.FeemarketKeeper,
			ante.NewDeductFeeDecorator(
				opts.AccountKeeper,
				opts.BankKeeper,
				opts.FeegrantKeeper,
				opts.TxFeeChecker,
			),
		), // fees are deducted in the fee market deduct post handler
		ante.NewSetPubKeyDecorator(opts.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(opts.AccountKeeper),
		ante.NewSigGasConsumeDecorator(opts.AccountKeeper, sigGasConsumer),
		ante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		ante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCkeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
