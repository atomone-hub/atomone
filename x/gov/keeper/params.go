package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// SetParams sets the gov module's parameters.
func (k Keeper) SetParams(ctx sdk.Context, params v1.Params) error {
	// before params change, trigger an update of the last min deposit
	minDeposit := k.GetMinDeposit(ctx)
	k.SetLastMinDeposit(ctx, minDeposit)
	// params.MinDeposit is deprecated and therefore should not be set.
	// Override any set value with the current min deposit, although
	// since the value of this param is ignored it will have no effect.
	params.MinDeposit = minDeposit

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)

	return nil
}

// GetParams gets the gov module's parameters.
func (k Keeper) GetParams(clientCtx sdk.Context) (params v1.Params) {
	store := clientCtx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}
