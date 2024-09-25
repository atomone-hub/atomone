package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// SetGovernanceDelegation sets a governance delegation in the store
func (k Keeper) SetGovernanceDelegation(ctx sdk.Context, delegation v1.GovernanceDelegation) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshal(&delegation)
	delAddr := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)
	store.Set(types.GovernanceDelegationKey(delAddr), b)

	// Set the reverse mapping from governor to delegation
	// mainly for querying all delegations for a governor
	// TODO: see if we can avoid duplicate storage
	govAddr := types.MustGovernorAddressFromBech32(delegation.GovernorAddress)
	store.Set(types.GovernanceDelegationsByGovernorKey(govAddr, delAddr), b)
}

// GetGovernanceDelegation gets a governance delegation from the store
func (k Keeper) GetGovernanceDelegation(ctx sdk.Context, delegatorAddr sdk.AccAddress) (v1.GovernanceDelegation, bool) {
	store := ctx.KVStore(k.storeKey)
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
	store := ctx.KVStore(k.storeKey)
	delegation, found := k.GetGovernanceDelegation(ctx, delegatorAddr)
	if !found {
		return
	}
	delAddr := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)
	store.Delete(types.GovernanceDelegationKey(delAddr))

	govAddr := types.MustGovernorAddressFromBech32(delegation.GovernorAddress)
	store.Delete(types.GovernanceDelegationsByGovernorKey(govAddr, delAddr))
}

// SetGovernorValShares sets a governor validator shares in the store
func (k Keeper) SetGovernorValShares(ctx sdk.Context, share v1.GovernorValShares) {
	store := ctx.KVStore(k.storeKey)
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
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.ValidatorSharesByGovernorKey(governorAddr, validatorAddr))
	if b == nil {
		return v1.GovernorValShares{}, false
	}
	var share v1.GovernorValShares
	k.cdc.MustUnmarshal(b, &share)
	return share, true
}

// IterateGovernorValShares iterates over all governor validator shares
func (k Keeper) IterateGovernorValShares(ctx sdk.Context, governorAddr types.GovernorAddress, cb func(index int64, share v1.GovernorValShares) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.ValidatorSharesByGovernorKey(governorAddr, []byte{}))
	defer iterator.Close()

	for i := int64(0); iterator.Valid(); iterator.Next() {
		var share v1.GovernorValShares
		k.cdc.MustUnmarshal(iterator.Value(), &share)
		if cb(i, share) {
			break
		}
		i++
	}
}

// IterateGovernorDelegations iterates over all governor delegations
func (k Keeper) IterateGovernorDelegations(ctx sdk.Context, governorAddr types.GovernorAddress, cb func(index int64, delegation v1.GovernanceDelegation) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.GovernanceDelegationsByGovernorKey(governorAddr, []byte{}))
	defer iterator.Close()

	for i := int64(0); iterator.Valid(); iterator.Next() {
		var delegation v1.GovernanceDelegation
		k.cdc.MustUnmarshal(iterator.Value(), &delegation)
		if cb(i, delegation) {
			break
		}
		i++
	}
}

// RemoveGovernorValShares removes a governor validator shares from the store
func (k Keeper) RemoveGovernorValShares(ctx sdk.Context, governorAddr types.GovernorAddress, validatorAddr sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ValidatorSharesByGovernorKey(governorAddr, validatorAddr))
}

// GetAllGovernanceDelegationsByGovernor gets all governance delegations for a specific governor
func (k Keeper) GetAllGovernanceDelegationsByGovernor(ctx sdk.Context, governorAddr types.GovernorAddress) (delegations []*v1.GovernanceDelegation) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.GovernanceDelegationsByGovernorKey(governorAddr, []byte{}))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var delegation v1.GovernanceDelegation
		k.cdc.MustUnmarshal(iterator.Value(), &delegation)
		delegations = append(delegations, &delegation)
	}
	return delegations
}

// GetAllGovernorValShares gets all governor validators shares
func (k Keeper) GetAllGovernorValShares(ctx sdk.Context, governorAddr types.GovernorAddress) []v1.GovernorValShares {
	store := ctx.KVStore(k.storeKey)
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

// IncreaseGovernorShares increases the governor validator shares in the store
func (k Keeper) IncreaseGovernorShares(ctx sdk.Context, governorAddr types.GovernorAddress, validatorAddr sdk.ValAddress, shares sdk.Dec) {
	valShares, found := k.GetGovernorValShares(ctx, governorAddr, validatorAddr)
	if !found {
		valShares = v1.NewGovernorValShares(governorAddr, validatorAddr, shares)
	} else {
		valShares.Shares = valShares.Shares.Add(shares)
	}
	k.SetGovernorValShares(ctx, valShares)
}

// DecreaseGovernorShares decreases the governor validator shares in the store
func (k Keeper) DecreaseGovernorShares(ctx sdk.Context, governorAddr types.GovernorAddress, validatorAddr sdk.ValAddress, shares sdk.Dec) {
	share, found := k.GetGovernorValShares(ctx, governorAddr, validatorAddr)
	if !found {
		panic("cannot decrease shares for a non-existent governor delegation")
	}
	share.Shares = share.Shares.Sub(shares)
	if share.Shares.IsNegative() {
		panic("negative shares")
	}
	if share.Shares.IsZero() {
		k.RemoveGovernorValShares(ctx, governorAddr, validatorAddr)
	} else {
		k.SetGovernorValShares(ctx, share)
	}
}

// UndelegateFromGovernor decreases all governor validator shares in the store
// and then removes the governor delegation for the given delegator
func (k Keeper) UndelegateFromGovernor(ctx sdk.Context, delegatorAddr sdk.AccAddress) {
	delegation, found := k.GetGovernanceDelegation(ctx, delegatorAddr)
	if !found {
		return
	}
	govAddr := types.MustGovernorAddressFromBech32(delegation.GovernorAddress)
	// iterate all delegations of delegator and decrease shares
	k.sk.IterateDelegations(ctx, delegatorAddr, func(_ int64, delegation stakingtypes.DelegationI) (stop bool) {
		k.DecreaseGovernorShares(ctx, govAddr, delegation.GetValidatorAddr(), delegation.GetShares())
		return false
	})
	// remove the governor delegation
	k.RemoveGovernanceDelegation(ctx, delegatorAddr)
}

// DelegateGovernor creates a governor delegation for the given delegator
// and increases all governor validator shares in the store
func (k Keeper) DelegateToGovernor(ctx sdk.Context, delegatorAddr sdk.AccAddress, governorAddr types.GovernorAddress) {
	delegation := v1.NewGovernanceDelegation(delegatorAddr, governorAddr)
	k.SetGovernanceDelegation(ctx, delegation)
	// iterate all delegations of delegator and increase shares
	k.sk.IterateDelegations(ctx, delegatorAddr, func(_ int64, delegation stakingtypes.DelegationI) (stop bool) {
		k.IncreaseGovernorShares(ctx, governorAddr, delegation.GetValidatorAddr(), delegation.GetShares())
		return false
	})
}

// RedelegateGovernor re-delegates all governor validator shares from one governor to another
func (k Keeper) RedelegateToGovernor(ctx sdk.Context, delegatorAddr sdk.AccAddress, dstGovernorAddr types.GovernorAddress) {
	// undelegate from the source governor
	k.UndelegateFromGovernor(ctx, delegatorAddr)
	// delegate to the destination governor
	k.DelegateToGovernor(ctx, delegatorAddr, dstGovernorAddr)
}
