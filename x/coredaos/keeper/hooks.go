package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	atomoneerrors "github.com/atomone-hub/atomone/types/errors"
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

func (h Hooks) AfterConsensusPubKeyUpdate(ctx context.Context, oldPk, newPk cryptotypes.PubKey, fee sdk.Coin) error {
	return nil
}

// validateDelegation checks if the delegation is being performed by a core DAO
// and returns an error if it is, since by Constitution, core DAOs cannot stake.
func validateDelegation(params types.Params, delAddr sdk.AccAddress) error {
	if params.GetSteeringDaoAddress() != "" {
		steeringDaoAddr := sdk.MustAccAddressFromBech32(params.GetSteeringDaoAddress())
		if delAddr.Equals(steeringDaoAddr) {
			return errorsmod.Wrap(types.ErrCannotStake, "Steering DAO cannot stake")
		}
	}
	if params.GetOversightDaoAddress() != "" {
		oversightDaoAddr := sdk.MustAccAddressFromBech32(params.GetOversightDaoAddress())
		if delAddr.Equals(oversightDaoAddr) {
			return errorsmod.Wrap(types.ErrCannotStake, "Oversight DAO cannot stake")
		}
	}
	return nil
}

var _ govtypes.GovHooks = Hooks{}

// GovHooks returns the gov hooks for the coredaos keeper.
func (k Keeper) GovHooks() Hooks {
	return Hooks{k}
}

// AfterProposalSubmission enforces two coredaos invariants on a submitted proposal:
//
//  1. coredaos MsgUpdateParams may never be delegated via authz. Its authority is the
//     governance account, so delegating it to any other account would hand governance-only
//     power to that account, which is unconsitutional as it provides control over core DAOs
//     to a delegate. Inspecting only top-level MsgGrant messages catches every such
//     delegation, because a gov->grantee grant for this message can arise nowhere else:
//     - It can only be created inside a governance proposal. A MsgGrant is signed by its
//     granter, and here the granter must be the gov account (authz keys the grant by the
//     executed message's signer, which is MsgUpdateParams.Authority == gov). The gov
//     module account has no key and cannot sign a transaction, so no standalone tx can
//     create the grant; proposal execution is the only path, and it is what this hook sees.
//     - Within a proposal it must be a top-level message. A MsgGrant nested inside an
//     authz.MsgExec is either self-executing (grantee == gov), which gov's SubmitProposal
//     already rejects, or requires a pre-existing grant to be dispatched, i.e. is circular.
//
//  2. A proposal that changes the oversight DAO address may not be bundled with other
//     messages. Only top-level messages are inspected: a MsgUpdateParams hidden below the top
//     level (e.g. inside a non-self-executing authz.MsgExec) could only ever take effect via
//     an authz grant, which invariant (1) has already made impossible to create.
func (h Hooks) AfterProposalSubmission(ctx context.Context, proposalID uint64) error {
	proposal, err := h.k.govKeeper.Proposals.Get(ctx, proposalID)
	if err != nil {
		return nil // proposal not found; nothing to enforce
	}

	updateParamsTypeURL := sdk.MsgTypeURL(&types.MsgUpdateParams{})

	// (1) Reject any authz grant that would delegate coredaos MsgUpdateParams. This is
	// unconditional: it does not depend on whether an oversight DAO is currently set, because
	// MsgUpdateParams governs every coredaos parameter.
	for _, anyMsg := range proposal.Messages {
		var msg sdk.Msg
		if err := h.k.cdc.UnpackAny(anyMsg, &msg); err != nil {
			continue
		}
		grant, ok := msg.(*authz.MsgGrant)
		if !ok {
			continue
		}
		var authorization authz.Authorization
		if err := h.k.cdc.UnpackAny(grant.Grant.Authorization, &authorization); err != nil {
			continue
		}
		if authorization.MsgTypeURL() == updateParamsTypeURL {
			return errorsmod.Wrap(atomoneerrors.ErrUnauthorized,
				"coredaos MsgUpdateParams authority cannot be delegated via authz")
		}
	}

	// (2) Reject bundling an oversight-DAO address change with other messages.
	params := h.k.GetParams(ctx)
	if params.OversightDaoAddress == "" {
		return nil
	}
	if len(proposal.Messages) <= 1 {
		return nil // bundling requires more than one message
	}
	for _, anyMsg := range proposal.Messages {
		var msg sdk.Msg
		if err := h.k.cdc.UnpackAny(anyMsg, &msg); err != nil {
			continue
		}
		updateParams, ok := msg.(*types.MsgUpdateParams)
		if !ok {
			continue
		}
		changed, err := oversightDaoAddressChanged(updateParams.Params.OversightDaoAddress, params.OversightDaoAddress)
		if err != nil {
			return errorsmod.Wrap(atomoneerrors.ErrUnauthorized, "failed to compare Oversight DAO addresses: "+err.Error())
		}
		if changed {
			return errorsmod.Wrap(atomoneerrors.ErrUnauthorized,
				"proposal that changes the Oversight DAO address cannot be bundled with other messages")
		}
	}
	return nil
}

func (h Hooks) AfterProposalDeposit(ctx context.Context, proposalID uint64, depositorAddr sdk.AccAddress) error {
	return nil
}

func (h Hooks) AfterProposalVote(ctx context.Context, proposalID uint64, voterAddr sdk.AccAddress) error {
	return nil
}

func (h Hooks) AfterProposalFailedMinDeposit(ctx context.Context, proposalID uint64) error {
	return nil
}

func (h Hooks) AfterProposalVotingPeriodEnded(ctx context.Context, proposalID uint64) error {
	return nil
}

// oversightDaoAddressChanged returns true if newAddr and currentAddr decode to different
// accounts (case-insensitive). Empty addresses are handled explicitly.
func oversightDaoAddressChanged(newAddr, currentAddr string) (bool, error) {
	if newAddr == "" && currentAddr == "" {
		return false, nil
	}
	if newAddr == "" || currentAddr == "" {
		return true, nil
	}
	newAccAddr, err := sdk.AccAddressFromBech32(newAddr)
	if err != nil {
		return false, err
	}
	currentAccAddr, err := sdk.AccAddressFromBech32(currentAddr)
	if err != nil {
		return false, err
	}
	return !newAccAddr.Equals(currentAccAddr), nil
}
