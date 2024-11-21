package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

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
	// This is required because genesis and simulated txs might have no fees.
	if ctx.BlockHeight() == 0 || simulate {
		return next(ctx, tx, simulate)
	}
	// If tx is excepted, don't validate the fee
	if isTxFeeExcepted(tx, vfd.k.GetParams(ctx).TxFeeExceptions) {
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
	if feeDenom := feeCoins[0].Denom; feeDenom != types.Denom {
		// feeDenom not allowed
		return ctx, sdkerrors.Wrapf(types.ErrInvalidFeeToken, "fee denom %s not allowed", feeDenom) //nolint:staticcheck
	}
	// feeDenom photon is allowed
	return next(ctx, tx, simulate)
}

// isTxFeeExcepted returns true if all tx messages type URL are presents in
// txFeeExceptions, or if it starts with a wildcard "*".
func isTxFeeExcepted(tx sdk.Tx, txFeeExceptions []string) bool {
	if len(txFeeExceptions) > 0 && txFeeExceptions[0] == "*" {
		// wildcard detected, all tx are excepted.
		return true
	}
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
