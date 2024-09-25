package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov/types"
)

// Hooks wrapper struct for gov keeper
type Hooks struct {
	k Keeper
}

var _ stakingtypes.StakingHooks = Hooks{}

// Return the slashing hooks
func (k Keeper) StakingHooks() Hooks {
	return Hooks{k}
}

// BeforeDelegationSharesModified is called when a delegation's shares are modified
// We trigger a governor shares decrease here subtracting all delegation shares.
// The right amount of shares will be possibly added back in AfterDelegationModified
func (h Hooks) BeforeDelegationSharesModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// does the delegator have a governance delegation?
	govDelegation, found := h.k.GetGovernanceDelegation(ctx, delAddr)
	if !found {
		return nil
	}
	govAddr := types.MustGovernorAddressFromBech32(govDelegation.GovernorAddress)

	// Fetch the delegation
	delegation, _ := h.k.sk.GetDelegation(ctx, delAddr, valAddr)

	// update the Governor's Validator shares
	h.k.DecreaseGovernorShares(ctx, govAddr, valAddr, delegation.Shares)

	return nil
}

// AfterDelegationModified is called when a delegation is created or modified
// We trigger a governor shares increase here adding all delegation shares.
// It is balanced by the full-amount decrease in BeforeDelegationSharesModified
func (h Hooks) AfterDelegationModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// does the delegator have a governance delegation?
	govDelegation, found := h.k.GetGovernanceDelegation(ctx, delAddr)
	if !found {
		return nil
	}

	// Fetch the delegation
	delegation, found := h.k.sk.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		return nil
	}

	governor, _ := h.k.GetGovernor(ctx, types.MustGovernorAddressFromBech32(govDelegation.GovernorAddress))

	// Calculate the new shares and update the Governor's shares
	shares := delegation.Shares

	h.k.IncreaseGovernorShares(ctx, governor.GetAddress(), valAddr, shares)

	return nil
}

// BeforeValidatorSlashed is called when a validator is slashed
func (h Hooks) BeforeValidatorSlashed(ctx sdk.Context, valAddr sdk.ValAddress, fraction sdk.Dec) error {
	return nil
}

// BeforeDelegationRemoved is called when a delegation is removed
func (h Hooks) BeforeDelegationRemoved(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeValidatorModified(ctx sdk.Context, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorRemoved(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBonded(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationCreated(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterUnbondingInitiated(ctx sdk.Context, unbondingID uint64) error {
	return nil
}
