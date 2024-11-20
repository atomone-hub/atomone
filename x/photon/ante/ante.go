package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	appparams "github.com/atomone-hub/atomone/app/params"
	"github.com/atomone-hub/atomone/x/photon/keeper"
	"github.com/atomone-hub/atomone/x/photon/types"
)

// ValidateFeeDecorator implements sdk.AnteDecorator and ensures that uphoton
// is the only fee token, except for some specific messages for which the type
// URLs are stored in the module parameters.
type ValidateFeeDecorator struct {
	k *keeper.Keeper
}

func NewValidateFeeDecorator(k *keeper.Keeper) ValidateFeeDecorator {
	return ValidateFeeDecorator{k: k}
}

// AnteHandle returns an error if tx fees doesn't follow the expectations:
//   - tx should have a single fee denom
//   - fee denom should be uphoton unless all the tx messages are present in
//     txFeeExceptions, in that case both uphoton and uatone are accepted.
func (vfd ValidateFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// If this is genesis height or simulate, don't validate the fee.
	// This is required because AtomOne's gentxs have no fees.
	// NOTE(tb): This could also be intercepted by checking if we are in checkTx
	// mode, but that allows malicious validators to deliver txs without photon
	// as the fee token.
	if ctx.BlockHeight() == 0 || simulate {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx") //nolint:staticcheck
	}
	feeCoins := feeTx.GetFee()
	if feeCoins.IsZero() {
		// no fees are allowed
		return next(ctx, tx, simulate)
	}
	if len(feeCoins) > 1 {
		return ctx, types.ErrTooManyFeeCoins
	}
	feeDenom := feeCoins[0].Denom
	if feeDenom == types.Denom {
		// feeDenom photon is allowed
		return next(ctx, tx, simulate)
	}
	if feeDenom == appparams.BondDenom && isTxFeeExcepted(tx, vfd.k.GetParams(ctx).TxFeeExceptions) {
		// feeDenom atone and tx fee excepted is allowed
		return next(ctx, tx, simulate)
	}
	// feeDenom not allowed
	return ctx, sdkerrors.Wrapf(types.ErrInvalidFeeToken, "fee denom %s not allowed", feeDenom) //nolint:staticcheck
}

// isTxFeeExcepted returns true if all tx messages type URL are presents in
// txFeeExceptions.
func isTxFeeExcepted(tx sdk.Tx, txFeeExceptions []string) bool {
	var exceptedMsgCount int
	for _, msg := range tx.GetMsgs() {
		msgTypeURL := sdk.MsgTypeURL(msg)
		for _, exception := range txFeeExceptions {
			if exception == msgTypeURL {
				exceptedMsgCount++
				break
			}
		}
	}
	return exceptedMsgCount == len(tx.GetMsgs())
}
