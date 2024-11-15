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
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}
	feeCoins := feeTx.GetFee()
	if len(feeCoins) > 1 {
		return ctx, types.ErrTooManyFeeCoins
	}
	// Base accepted fee denom is uphoton
	acceptedFeeCoins := sdk.NewCoins(sdk.NewInt64Coin(types.Denom, 1))
	if isTxFeeExcepted(tx, vfd.k.GetParams(ctx).TxFeeExceptions) {
		// tx is fee excepted, add uatone as an other accepted fee denom
		// TODO replace "uatone" with constant or parameter
		acceptedFeeCoins = acceptedFeeCoins.Add(sdk.NewInt64Coin("uatone", 1))
	}
	// feeCoins must be at least higher than one of the acceptedFeeCoins.
	// NOTE(tb): this check rejects tx with 0 fees, maybe this should not be there (does this work when simulate=true?)
	if !feeCoins.IsAnyGTE(acceptedFeeCoins) {
		return ctx, sdkerrors.Wrapf(types.ErrInvalidFeeToken, "expected %s got %s", acceptedFeeCoins, feeCoins)
	}
	return next(ctx, tx, simulate)
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
