package post

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// FeemarketStateUpdateDecorator updates the state of the fee market based on the gas consumed in the gasmeter.
// Call next PostHandler if fees successfully deducted.
// CONTRACT: Tx must implement FeeTx interface
type FeemarketStateUpdateDecorator struct {
	feemarketKeeper FeemarketKeeper
}

func NewFeemarketStateUpdateDecorator(fmk FeemarketKeeper) FeemarketStateUpdateDecorator {
	return FeemarketStateUpdateDecorator{
		feemarketKeeper: fmk,
	}
}

// PostHandle deducts the fee from the fee payer based on the min base fee and the gas consumed in the gasmeter.
// If there is a difference between the provided fee and the min-base fee, the difference is paid as a tip.
// Fees are sent to the x/feemarket fee-collector address.
func (dfd FeemarketStateUpdateDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool, next sdk.PostHandler) (sdk.Context, error) {
	// GenTx consume no fee
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate, success)
	}

	// update fee market params
	params, err := dfd.feemarketKeeper.GetParams(ctx)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to get fee market params")
	}

	// return if disabled
	if !params.Enabled {
		return next(ctx, tx, simulate, success)
	}

	enabledHeight, err := dfd.feemarketKeeper.GetEnabledHeight(ctx)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to get fee market enabled height")
	}

	// if the current height is that which enabled the feemarket or lower, skip deduction
	if ctx.BlockHeight() <= enabledHeight {
		return next(ctx, tx, simulate, success)
	}

	// update fee market state
	state, err := dfd.feemarketKeeper.GetState(ctx)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to get fee market state")
	}

	gas := ctx.GasMeter().GasConsumed() // use context gas consumed

	ctx.Logger().Info("feemarket post handle",
		"gas consumed", gas,
	)

	err = state.Update(gas, params)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to update fee market state")
	}

	err = dfd.feemarketKeeper.SetState(ctx, state)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to set fee market state")
	}

	return next(ctx, tx, simulate, success)
}
