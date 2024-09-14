package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// GetGovernor returns the governor with the provided address
func (k Keeper) GetGovernor(ctx sdk.Context, addr types.GovernorAddress) (v1.Governor, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GovernorKey(addr))
	if bz == nil {
		return v1.Governor{}, false
	}

	var governor v1.Governor
	v1.MustMarshalGovernor(k.cdc, &governor)
	return governor, true
}

// SetGovernor sets the governor in the store
func (k Keeper) SetGovernor(ctx sdk.Context, governor v1.Governor) {
	store := ctx.KVStore(k.storeKey)
	bz := v1.MustMarshalGovernor(k.cdc, &governor)
	store.Set(types.GovernorKey(governor.GetAddress()), bz)
}

// GetAllGovernors returns all governors
func (k Keeper) GetAllGovernors(ctx sdk.Context) (governors []v1.Governor) {
	store := ctx.KVStore(k.storeKey)

	iterator := sdk.KVStorePrefixIterator(store, types.GovernorKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		governor := v1.MustUnmarshalGovernor(k.cdc, iterator.Value())
		governors = append(governors, governor)
	}

	return governors
}
