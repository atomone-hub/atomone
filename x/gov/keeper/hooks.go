package keeper

import (
	context "context"

	"cosmossdk.io/math"

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

// Return the staking hooks
func (k Keeper) StakingHooks() Hooks {
	return Hooks{k}
}

// BeforeDelegationSharesModified is called when a delegation's shares are modified
// We trigger a governor shares decrease here subtracting all delegation shares.
// The right amount of shares will be possibly added back in AfterDelegationModified
func (h Hooks) BeforeDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// does the delegator have a governance delegation?
	govDelegation, found := h.k.GetGovernanceDelegation(sdkCtx, delAddr)
	if !found {
		return nil
	}
	govAddr := types.MustGovernorAddressFromBech32(govDelegation.GovernorAddress)

	// Fetch the delegation
	delegation, _ := h.k.sk.GetDelegation(sdkCtx, delAddr, valAddr)

	// update the Governor's Validator shares
	h.k.DecreaseGovernorShares(sdkCtx, govAddr, valAddr, delegation.Shares)

	return nil
}

// AfterDelegationModified is called when a delegation is created or modified
// We trigger a governor shares increase here adding all delegation shares.
// It is balanced by the full-amount decrease in BeforeDelegationSharesModified
func (h Hooks) AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// does the delegator have a governance delegation?
	govDelegation, found := h.k.GetGovernanceDelegation(sdkCtx, delAddr)
	if !found {
		return nil
	}

	// Fetch the delegation
	delegation, err := h.k.sk.GetDelegation(sdkCtx, delAddr, valAddr)
	if err != nil {
		return err
	}

	govAddr := types.MustGovernorAddressFromBech32(govDelegation.GovernorAddress)

	// Calculate the new shares and update the Governor's shares
	shares := delegation.Shares

	h.k.IncreaseGovernorShares(sdkCtx, govAddr, valAddr, shares)

	// if the delegator is also an active governor, ensure min self-delegation requirement is met,
	// otherwise set governor to inactive
	delGovAddr := types.GovernorAddress(delAddr.Bytes())
	if governor, found := h.k.GetGovernor(sdkCtx, delGovAddr); found && governor.IsActive() {
		if governor.GetAddress().String() != govDelegation.GovernorAddress {
			panic("active governor delegating to another governor")
		}
		// if the governor no longer meets the min self-delegation, set to inactive
		if !h.k.ValidateGovernorMinSelfDelegation(sdkCtx, governor) {
			governor.Status = v1.Inactive
			now := sdkCtx.BlockTime()
			governor.LastStatusChangeTime = &now
			h.k.SetGovernor(sdkCtx, governor)
		}
	}

	return nil
}

// BeforeDelegationRemoved is called when a delegation is removed
// We verify if the delegator is also an active governor and if so check
// that the min self-delegation requirement is still met, otherwise set governor
// status to inactive
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// if the delegator is also an active governor, ensure min self-delegation requirement is met,
	// otherwise set governor to inactive
	delGovAddr := types.GovernorAddress(delAddr.Bytes())
	if governor, found := h.k.GetGovernor(sdkCtx, delGovAddr); found && governor.IsActive() {
		govDelegation, found := h.k.GetGovernanceDelegation(sdkCtx, delAddr)
		if !found {
			panic("active governor without governance self-delegation")
		}
		if governor.GetAddress().String() != govDelegation.GovernorAddress {
			panic("active governor delegating to another governor")
		}
		// if the governor no longer meets the min self-delegation, set to inactive
		if !h.k.ValidateGovernorMinSelfDelegation(sdkCtx, governor) {
			governor.Status = v1.Inactive
			now := sdkCtx.BlockTime()
			governor.LastStatusChangeTime = &now
			h.k.SetGovernor(sdkCtx, governor)
		}
	}

	return nil
}

func (h Hooks) BeforeValidatorSlashed(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error {
	return nil
}

func (h Hooks) AfterValidatorCreated(ctx context.Context, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeValidatorModified(ctx context.Context, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorRemoved(ctx context.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBonded(ctx context.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterUnbondingInitiated(ctx context.Context, unbondingID uint64) error {
	return nil
}
