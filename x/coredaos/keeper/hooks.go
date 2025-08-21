package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/coredaos/types"
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

// BeforeDelegationSharesModified is called before a delegation's shares are modified
func (h Hooks) BeforeDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	params := h.k.GetParams(ctx)
	return validateDelegation(params, delAddr)
}

// BeforeDelegationCreated is called before a delegation is created
func (h Hooks) BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	params := h.k.GetParams(ctx)
	return validateDelegation(params, delAddr)
}

func (h Hooks) AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeValidatorSlashed(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error {
	return nil
}

func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
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

func (h Hooks) AfterUnbondingInitiated(ctx context.Context, unbondingID uint64) error {
	return nil
}

// validateDelegation checks if the delegation is being performed by a core DAO
// and returns an error if it is, since by Constitution, core DAOs cannot stake.
func validateDelegation(params types.Params, delAddr sdk.AccAddress) error {
	if params.GetSteeringDaoAddress() != "" {
		steeringDaoAddr := sdk.MustAccAddressFromBech32(params.GetSteeringDaoAddress())
		if delAddr.Equals(steeringDaoAddr) {
			return sdkerrors.Wrap(types.ErrCannotStake, "Steering DAO cannot stake")
		}
	}
	if params.GetOversightDaoAddress() != "" {
		oversightDaoAddr := sdk.MustAccAddressFromBech32(params.GetOversightDaoAddress())
		if delAddr.Equals(oversightDaoAddr) {
			return sdkerrors.Wrap(types.ErrCannotStake, "Oversight DAO cannot stake")
		}
	}
	return nil
}
