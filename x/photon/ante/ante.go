package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/atomone-hub/atomone/x/photon/keeper"
	"github.com/atomone-hub/atomone/x/photon/types"
)

var _ sdk.AnteDecorator = ValidateFeeDecorator{}

type ValidateFeeDecorator struct {
	k *keeper.Keeper
}

func NewValidateFeeDecorator(k *keeper.Keeper) ValidateFeeDecorator {
	return ValidateFeeDecorator{k: k}
}

// AnteHandle implements the sdk.AnteDecorator interface.
// It returns an error if the tx fee denom is not photon, with some exceptions:
//   - tx is a gentx
//   - tx mode is simulate
//   - tx messages' type URLs match the `TxFeeExceptions` field of the
//     [types.Params].
//   - tx has no fees or 0 fees.
func (vfd ValidateFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if ctx.BlockHeight() == 0 || simulate {
		// Skip if this is genesis height or simulate mode, because genesis and
		// simulated transactions might have no fees.
		return next(ctx, tx, simulate)
	}
	if isTxFeeExcepted(tx, vfd.k.GetParams(ctx).TxFeeExceptions) {
		// Skip if tx is excepted (any fee coins are allowed).
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx") //nolint:staticcheck
	}
	feeCoins := feeTx.GetFee()
	if feeCoins.IsZero() {
		// Skip if no fees
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
