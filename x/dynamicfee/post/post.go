package post

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DynamicfeeStateUpdateDecorator updates the state of the dynamic fee pricing
// based on the gas consumed in the gasmeter. Call next PostHandler if fees
// successfully deducted.
// CONTRACT: Tx must implement FeeTx interface
type DynamicfeeStateUpdateDecorator struct {
	dynamicfeeKeeper      DynamicfeeKeeper
	consensusParamsKeeper ConsensusParamsKeeper
}

func NewDynamicfeeStateUpdateDecorator(fmk DynamicfeeKeeper, cpk ConsensusParamsKeeper) DynamicfeeStateUpdateDecorator {
	return DynamicfeeStateUpdateDecorator{
		dynamicfeeKeeper:      fmk,
		consensusParamsKeeper: cpk,
	}
}

// PostHandle deducts the fee from the fee payer based on the min base fee and the gas consumed in the gasmeter.
// If there is a difference between the provided fee and the min-base fee, the difference is paid as a tip.
// Fees are sent to the x/dynamicfee fee-collector address.
func (dfd DynamicfeeStateUpdateDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool, next sdk.PostHandler) (sdk.Context, error) {
	// GenTx consume no fee
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate, success)
	}

	// update dynamic fee pricing params
	params, err := dfd.dynamicfeeKeeper.GetParams(ctx)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to get dynamicfee params")
	}

	// return if disabled
	if !params.Enabled {
		return next(ctx, tx, simulate, success)
	}

	enabledHeight, err := dfd.dynamicfeeKeeper.GetEnabledHeight(ctx)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to get dynamicfee enabled height")
	}

	// if the current height is that which enabled the dynamicfee or lower, skip deduction
	if ctx.BlockHeight() <= enabledHeight {
		return next(ctx, tx, simulate, success)
	}

	// update dynamic fee pricing state
	state, err := dfd.dynamicfeeKeeper.GetState(ctx)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to get dynamicfee state")
	}

	gas := ctx.GasMeter().GasConsumed() // use context gas consumed

	ctx.Logger().Info("dynamicfee post handle",
		"gas consumed", gas,
	)

	consensusParams, err := dfd.consensusParamsKeeper.Get(ctx)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to get consensus params")
	}
	maxBlockGas := uint64(consensusParams.Block.MaxGas)

	err = state.Update(gas, maxBlockGas)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to update dynamicfee state")
	}

	err = dfd.dynamicfeeKeeper.SetState(ctx, state)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to set dynamicfee state")
	}

	return next(ctx, tx, simulate, success)
}
