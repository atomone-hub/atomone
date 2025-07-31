package keeper

import (
	sdkerrors "cosmossdk.io/errors"

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
func (h Hooks) BeforeDelegationSharesModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	params := h.k.GetParams(ctx)
	return validateDelegation(params, delAddr)
}

// BeforeDelegationCreated is called before a delegation is created
func (h Hooks) BeforeDelegationCreated(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	params := h.k.GetParams(ctx)
	return validateDelegation(params, delAddr)
}

func (h Hooks) AfterDelegationModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeValidatorSlashed(ctx sdk.Context, valAddr sdk.ValAddress, fraction sdk.Dec) error {
	return nil
}

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

func (h Hooks) AfterUnbondingInitiated(ctx sdk.Context, unbondingID uint64) error {
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
