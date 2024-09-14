package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
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
	// iterate through all GovernorValShares and reduce the governor VP by the appropriate amount
	h.k.IterateGovernorValSharesByValidator(ctx, valAddr, func(index int64, shares v1.GovernorValShares) bool {
		govAddr := types.MustGovernorAddressFromBech32(shares.GovernorAddress)
		governor, _ := h.k.GetGovernor(ctx, govAddr)
		validator, _ := h.k.sk.GetValidator(ctx, valAddr)
		tokensBurned := shares.Shares.MulInt(validator.GetBondedTokens()).Quo(validator.GetDelegatorShares()).Mul(fraction)
		governorVP := governor.GetVotingPower().Sub(tokensBurned)
		if governorVP.IsNegative() {
			panic("negative governor voting power")
		}
		governor.SetVotingPower(governorVP)
		h.k.SetGovernor(ctx, governor)
		return false
	})

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
