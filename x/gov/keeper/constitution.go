package keeper

import (
	"github.com/atomone-hub/atomone/x/gov/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (keeper Keeper) GetConstitution(ctx sdk.Context) (constitution string) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(types.KeyConstitution)

	return string(bz)
}

func (keeper Keeper) SetConstitution(ctx sdk.Context, constitution string) {
	store := ctx.KVStore(keeper.storeKey)
	store.Set([]byte(types.KeyConstitution), []byte(constitution))
}
