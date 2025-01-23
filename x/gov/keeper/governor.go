package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

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

	return v1.MustUnmarshalGovernor(k.cdc, bz), true
}

// SetGovernor sets the governor in the store
func (k Keeper) SetGovernor(ctx sdk.Context, governor v1.Governor) {
	store := ctx.KVStore(k.storeKey)
	bz := v1.MustMarshalGovernor(k.cdc, &governor)
	store.Set(types.GovernorKey(governor.GetAddress()), bz)
}

// GetAllGovernors returns all governors
func (k Keeper) GetAllGovernors(ctx sdk.Context) (governors []*v1.Governor) {
	store := ctx.KVStore(k.storeKey)

	iterator := sdk.KVStorePrefixIterator(store, types.GovernorKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		governor := v1.MustUnmarshalGovernor(k.cdc, iterator.Value())
		governors = append(governors, &governor)
	}

	return governors
}

// GetAllActiveGovernors returns all active governors
func (k Keeper) GetAllActiveGovernors(ctx sdk.Context) (governors []*v1.Governor) {
	store := ctx.KVStore(k.storeKey)

	iterator := sdk.KVStorePrefixIterator(store, types.GovernorKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		governor := v1.MustUnmarshalGovernor(k.cdc, iterator.Value())
		if governor.IsActive() {
			governors = append(governors, &governor)
		}
	}

	return governors
}

// IterateGovernors iterates over all governors and performs a callback function
func (k Keeper) IterateGovernors(ctx sdk.Context, cb func(index int64, governor v1.GovernorI) (stop bool)) {
	store := ctx.KVStore(k.storeKey)

	iterator := sdk.KVStorePrefixIterator(store, types.GovernorKeyPrefix)
	defer iterator.Close()

	for i := int64(0); iterator.Valid(); iterator.Next() {
		governor := v1.MustUnmarshalGovernor(k.cdc, iterator.Value())
		if cb(i, governor) {
			break
		}
		i++
	}
}

func (k Keeper) getGovernorBondedTokens(ctx sdk.Context, govAddr types.GovernorAddress) (bondedTokens math.Int) {
	bondedTokens = sdk.ZeroInt()
	addr := sdk.AccAddress(govAddr)
	k.sk.IterateDelegations(ctx, addr, func(_ int64, delegation stakingtypes.DelegationI) (stop bool) {
		validatorAddr := delegation.GetValidatorAddr()
		validator, _ := k.sk.GetValidator(ctx, validatorAddr)
		shares := delegation.GetShares()
		bt := shares.MulInt(validator.GetBondedTokens()).Quo(validator.GetDelegatorShares()).TruncateInt()
		bondedTokens = bondedTokens.Add(bt)

		return false
	})

	return bondedTokens
}

func (k Keeper) ValidateGovernorMinSelfDelegation(ctx sdk.Context, governor v1.Governor) bool {
	// ensure that the governor is active and that has a valid governance self-delegation
	if !governor.IsActive() {
		return false
	}
	minGovernorSelfDelegation, _ := math.NewIntFromString(k.GetParams(ctx).MinGovernorSelfDelegation)
	bondedTokens := k.getGovernorBondedTokens(ctx, governor.GetAddress())
	delAddr := sdk.AccAddress(governor.GetAddress())

	if del, found := k.GetGovernanceDelegation(ctx, delAddr); !found || governor.GovernorAddress != del.GovernorAddress {
		panic("active governor without governance self-delegation")
	}

	if bondedTokens.LT(minGovernorSelfDelegation) {
		return false
	}

	return true
}
