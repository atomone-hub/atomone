package keeper

import (
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetGovernanceDelegation sets a governance delegation in the store
func (k Keeper) SetGovernanceDelegation(ctx sdk.Context, delegation v1.GovernanceDelegation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GovernanceDelegationKeyPrefix)
	b := k.cdc.MustMarshal(&delegation)
	delAddr := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)
	store.Set(types.GovernanceDelegationKey(delAddr), b)

	// Set the reverse mapping from governor to delegation
	// mainly for querying all delegations for a governor
	// TODO: see if we can avoid duplicate storage
	govAddr := types.MustGovernorAddressFromBech32(delegation.GovernorAddress)
	store = prefix.NewStore(ctx.KVStore(k.storeKey), types.GovernanceDelegationKeyPrefix)
	store.Set(types.GovernanceDelegationsByGovernorKey(govAddr, delAddr), b)
}

// GetGovernanceDelegation gets a governance delegation from the store
func (k Keeper) GetGovernanceDelegation(ctx sdk.Context, delegatorAddr sdk.AccAddress) (v1.GovernanceDelegation, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GovernanceDelegationKeyPrefix)
	b := store.Get(types.GovernanceDelegationKey(delegatorAddr))
	if b == nil {
		return v1.GovernanceDelegation{}, false
	}
	var delegation v1.GovernanceDelegation
	k.cdc.MustUnmarshal(b, &delegation)
	return delegation, true
}

// RemoveGovernanceDelegation removes a governance delegation from the store
func (k Keeper) RemoveGovernanceDelegation(ctx sdk.Context, delegatorAddr sdk.AccAddress) {
	// need to remove from both the delegator and governor mapping
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GovernanceDelegationKeyPrefix)
	delegation, found := k.GetGovernanceDelegation(ctx, delegatorAddr)
	if !found {
		return
	}
	delAddr := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)
	store.Delete(types.GovernanceDelegationKey(delAddr))

	govAddr := types.MustGovernorAddressFromBech32(delegation.GovernorAddress)
	store = prefix.NewStore(ctx.KVStore(k.storeKey), types.GovernanceDelegationKeyPrefix)
	store.Delete(types.GovernanceDelegationsByGovernorKey(govAddr, delAddr))
}

// SetGovernorValShares sets a governor validator shares in the store
func (k Keeper) SetGovernorValShares(ctx sdk.Context, share v1.GovernorValShares) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValidatorSharesByGovernorKeyPrefix)
	b := k.cdc.MustMarshal(&share)
	govAddr := types.MustGovernorAddressFromBech32(share.GovernorAddress)
	valAddr, err := sdk.ValAddressFromBech32(share.ValidatorAddress)
	if err != nil {
		panic(err)
	}
	store.Set(types.ValidatorSharesByGovernorKey(govAddr, valAddr), b)
}

// GetGovernorValShares gets a governor validator shares from the store
func (k Keeper) GetGovernorValShares(ctx sdk.Context, governorAddr types.GovernorAddress, validatorAddr sdk.ValAddress) (v1.GovernorValShares, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValidatorSharesByGovernorKeyPrefix)
	b := store.Get(types.ValidatorSharesByGovernorKey(governorAddr, validatorAddr))
	if b == nil {
		return v1.GovernorValShares{}, false
	}
	var share v1.GovernorValShares
	k.cdc.MustUnmarshal(b, &share)
	return share, true
}

// RemoveGovernorValShares removes a governor validator shares from the store
func (k Keeper) RemoveGovernorValShares(ctx sdk.Context, governorAddr types.GovernorAddress, validatorAddr sdk.ValAddress) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValidatorSharesByGovernorKeyPrefix)
	store.Delete(types.ValidatorSharesByGovernorKey(governorAddr, validatorAddr))
}

// GetAllGovernanceDelegationsByGovernor gets all governance delegations for a specific governor
func (k Keeper) GetAllGovernanceDelegationsByGovernor(ctx sdk.Context, governorAddr types.GovernorAddress) []v1.GovernanceDelegation {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.GovernanceDelegationsByGovernorKeyPrefix)
	iterator := sdk.KVStorePrefixIterator(store, types.GovernanceDelegationsByGovernorKey(governorAddr, []byte{}))
	defer iterator.Close()

	var delegations []v1.GovernanceDelegation
	for ; iterator.Valid(); iterator.Next() {
		var delegation v1.GovernanceDelegation
		k.cdc.MustUnmarshal(iterator.Value(), &delegation)
		delegations = append(delegations, delegation)
	}
	return delegations
}

// GetAllGovernorValShares gets all governor validators shares
func (k Keeper) GetAllGovernorValShares(ctx sdk.Context, governorAddr types.GovernorAddress) []v1.GovernorValShares {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValidatorSharesByGovernorKeyPrefix)
	iterator := sdk.KVStorePrefixIterator(store, types.ValidatorSharesByGovernorKey(governorAddr, []byte{}))
	defer iterator.Close()

	var shares []v1.GovernorValShares
	for ; iterator.Valid(); iterator.Next() {
		var share v1.GovernorValShares
		k.cdc.MustUnmarshal(iterator.Value(), &share)
		shares = append(shares, share)
	}
	return shares
}
