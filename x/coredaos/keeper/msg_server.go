package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/atomone-hub/atomone/x/coredaos/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

var _ types.MsgServer = (*MsgServer)(nil)

type MsgServer struct {
	k *Keeper
}

// NewMsgServer returns the MsgServer implementation.
func NewMsgServer(k *Keeper) types.MsgServer {
	return &MsgServer{k: k}
}

// UpdateParams defines a method that updates the module's parameters. The signer of the message must
// be the module authority.
func (ms MsgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if ms.k.GetAuthority() != msg.Authority {
		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.GetAuthority(), msg.Authority)
	}

	params := msg.Params
	// check if any of the core DAOs has bonded or unbonding tokens, and if so return an error
	if params.SteeringDaoAddress != "" {
		delegatorBonded, err := ms.k.stakingKeeper.GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(params.SteeringDaoAddress))
		if err != nil {
			return nil, err
		}
		delegatorUnbonded, err := ms.k.stakingKeeper.GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(params.SteeringDaoAddress))
		if err != nil {
			return nil, err
		}
		if delegatorBonded.GT(math.ZeroInt()) ||
			delegatorUnbonded.GT(math.ZeroInt()) {
			return nil, errors.Wrapf(types.ErrCannotStake, "cannot update params while Steering DAO have bonded or unbonding tokens")
		}
	}
	if params.OversightDaoAddress != "" {
		delegatorBonded, err := ms.k.stakingKeeper.GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(params.OversightDaoAddress))
		if err != nil {
			return nil, err
		}
		delegatorUnbonded, err := ms.k.stakingKeeper.GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(params.OversightDaoAddress))
		if err != nil {
			return nil, err
		}
		if delegatorBonded.GT(math.ZeroInt()) ||
			delegatorUnbonded.GT(math.ZeroInt()) {
			return nil, errors.Wrapf(types.ErrCannotStake, "cannot update params while Oversight DAO have bonded or unbonding tokens")
		}
	}
	if err := ms.k.Params.Set(ctx, params); err != nil {
		return nil, errors.Wrapf(err, "error setting params")
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// AnnotateProposal adds an annotation to the proposal with the given ID.
// The annotation is a string that can be used to provide additional context or information about the proposal.
// The proposal must be in the voting period for the annotation to be added.
// It is only available to the Steering DAO.
func (ms MsgServer) AnnotateProposal(goCtx context.Context, msg *types.MsgAnnotateProposal) (*types.MsgAnnotateProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := ms.k.GetParams(ctx)

	logger := ms.k.Logger(ctx)

	if params.SteeringDaoAddress == "" {
		logger.Info("Steering DAO address is not set, function is disabled")

		return nil, errors.Wrapf(types.ErrFunctionDisabled, "Steering DAO address is not set")
	}

	if msg.Annotator != params.SteeringDaoAddress {
		logger.Error(
			"invalid authority for annotating proposal",
			"expected", params.SteeringDaoAddress,
			"got", msg.Annotator,
		)

		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", params.SteeringDaoAddress, msg.Annotator)
	}

	proposal, found := ms.k.govKeeper.GetProposal(ctx, msg.ProposalId)
	if !found {
		logger.Error(
			"proposal not found",
			"proposal_id", msg.ProposalId,
			"authority", msg.Annotator,
		)

		return nil, errors.Wrapf(govtypes.ErrUnknownProposal, "proposal with ID %d not found", msg.ProposalId)
	}
	if proposal.Status != govtypesv1.StatusVotingPeriod {
		logger.Error(
			"proposal is not in voting period",
			"proposal", proposal.Id,
			"status", proposal.Status,
			"authority", msg.Annotator,
		)

		return nil, errors.Wrapf(sdkgovtypes.ErrInactiveProposal, "proposal with ID %d is not in voting period", msg.ProposalId)
	}

	// Check if the proposal already has an annotation, if so, allow overwriting only if the `overwrite` flag is set to true.
	if proposal.Annotation != "" && !msg.Overwrite {
		logger.Error(
			"proposal already has an annotation and overwrite is set to false",
			"proposal", proposal.Id,
			"authority", msg.Annotator,
		)

		return nil, errors.Wrapf(types.ErrAnnotationAlreadyPresent, "proposal with ID %d already has an annotation", msg.ProposalId)
	}

	proposal.Annotation = msg.Annotation
	ms.k.govKeeper.SetProposal(ctx, proposal)

	logger.Info(
		"proposal annotated",
		"proposal", proposal.Id,
		"authority", msg.Annotator,
	)

	// Emit event for proposal annotation
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAnnotateProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
			sdk.NewAttribute(types.AttributeKeySigner, msg.Annotator),
		),
	})

	return &types.MsgAnnotateProposalResponse{}, nil
}

// EndorseProposal allows the Steering DAO to endorse a proposal.
// It requires the proposal to be in the voting period, and the endorsing account must be the Steering DAO.
// A proposal can only be endorsed once.
func (ms MsgServer) EndorseProposal(goCtx context.Context, msg *types.MsgEndorseProposal) (*types.MsgEndorseProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := ms.k.GetParams(ctx)

	logger := ms.k.Logger(ctx)

	if params.SteeringDaoAddress == "" {
		logger.Info("Steering DAO address is not set, function is disabled")

		return nil, errors.Wrapf(types.ErrFunctionDisabled, "Steering DAO address is not set")
	}

	if msg.Endorser != params.SteeringDaoAddress {
		logger.Error(
			"invalid authority for endorsing proposal",
			"expected", params.SteeringDaoAddress,
			"got", msg.Endorser,
		)

		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", params.SteeringDaoAddress, msg.Endorser)
	}

	proposal, found := ms.k.govKeeper.GetProposal(ctx, msg.ProposalId)
	if !found {
		logger.Error(
			"proposal not found",
			"proposal_id", msg.ProposalId,
			"authority", msg.Endorser,
		)

		return nil, errors.Wrapf(govtypes.ErrUnknownProposal, "proposal with ID %d not found", msg.ProposalId)
	}
	if proposal.Status != govtypesv1.StatusVotingPeriod {
		logger.Error(
			"proposal is not in voting period",
			"proposal", proposal.Id,
			"status", proposal.Status,
			"authority", msg.Endorser,
		)

		return nil, errors.Wrapf(sdkgovtypes.ErrInactiveProposal, "proposal with ID %d is not in voting period", msg.ProposalId)
	}
	if proposal.Endorsed {
		logger.Error(
			"proposal has already been endorsed",
			"proposal", proposal.Id,
			"authority", msg.Endorser,
		)

		return nil, errors.Wrapf(types.ErrProposalAlreadyEndorsed, "proposal with ID %d has already been endorsed", msg.ProposalId)
	}

	proposal.Endorsed = true
	ms.k.govKeeper.SetProposal(ctx, proposal)

	logger.Info(
		"proposal endorsed",
		"proposal", proposal.Id,
		"authority", msg.Endorser,
	)

	// Emit event for proposal endorsement
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeEndorseProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
			sdk.NewAttribute(types.AttributeKeySigner, msg.Endorser),
		),
	})

	return &types.MsgEndorseProposalResponse{}, nil
}

// ExtendVotingPeriod allows the signer to extend the voting period of a proposal.
// The proposal must be in the voting period, and the signer must be the designated
// Steering DAO or Oversight DAO.
// The voting period cannot be extended further than the maximum defined in the module parameters.
// The extension duration is defined in the module parameters.
func (ms MsgServer) ExtendVotingPeriod(goCtx context.Context, msg *types.MsgExtendVotingPeriod) (*types.MsgExtendVotingPeriodResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := ms.k.GetParams(ctx)

	logger := ms.k.Logger(ctx)

	if params.SteeringDaoAddress == "" && params.OversightDaoAddress == "" {
		logger.Info("Steering DAO address and Oversight DAO address are not set, function is disabled")

		return nil, errors.Wrapf(types.ErrFunctionDisabled, "Steering DAO address and Oversight DAO address are not set")
	}

	if msg.Extender != params.SteeringDaoAddress && msg.Extender != params.OversightDaoAddress {
		// one of the two addresses must be set otherwise it would have been caught earlier
		addressesString := fmt.Sprintf("%s or %s", params.SteeringDaoAddress, params.OversightDaoAddress)
		if params.SteeringDaoAddress == "" {
			addressesString = params.OversightDaoAddress
		} else if params.OversightDaoAddress == "" {
			addressesString = params.SteeringDaoAddress
		}

		logger.Error(
			"invalid authority for extending voting period",
			"expected", addressesString,
			"got", msg.Extender,
		)

		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", addressesString, msg.Extender)
	}

	proposal, found := ms.k.govKeeper.GetProposal(ctx, msg.ProposalId)
	if !found {
		logger.Error(
			"proposal not found",
			"proposal_id", msg.ProposalId,
			"authority", msg.Extender,
		)

		return nil, errors.Wrapf(govtypes.ErrUnknownProposal, "proposal with ID %d not found", msg.ProposalId)
	}
	if proposal.Status != govtypesv1.StatusVotingPeriod {
		logger.Error(
			"proposal is not in voting period",
			"proposal", proposal.Id,
			"status", proposal.Status,
			"authority", msg.Extender,
		)

		return nil, errors.Wrapf(sdkgovtypes.ErrInactiveProposal, "proposal with ID %d is not in voting period", msg.ProposalId)
	}
	if proposal.TimesVotingPeriodExtended >= params.VotingPeriodExtensionsLimit {
		logger.Error(
			"proposal has reached the maximum number of voting period extensions",
			"proposal", proposal.Id,
			"times_extended", proposal.TimesVotingPeriodExtended,
			"authority", msg.Extender,
		)

		return nil, errors.Wrapf(sdkgovtypes.ErrInvalidProposalContent, "proposal with ID %d has reached the maximum number of voting period extensions", msg.ProposalId)
	}

	newEndTime := proposal.VotingEndTime.Add(*params.VotingPeriodExtensionDuration)

	// Update ActiveProposalsQueue with new VotingEndTime
	ms.k.govKeeper.RemoveFromActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)
	proposal.VotingEndTime = &newEndTime
	ms.k.govKeeper.InsertActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)

	proposal.TimesVotingPeriodExtended++
	ms.k.govKeeper.SetProposal(ctx, proposal)

	logger.Info(
		"voting period extended",
		"proposal", proposal.Id,
		"new_end_time", proposal.VotingEndTime,
		"authority", msg.Extender,
		"times_extended", proposal.TimesVotingPeriodExtended,
	)

	// Emit event for voting period extension
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeExtendVotingPeriod,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
			sdk.NewAttribute(types.AttributeKeySigner, msg.Extender),
			sdk.NewAttribute(types.AttributeKeyNewEndTime, proposal.VotingEndTime.String()),
			sdk.NewAttribute(types.AttributeKeyTimesExtended, fmt.Sprintf("%d", proposal.TimesVotingPeriodExtended)),
		),
	})

	return &types.MsgExtendVotingPeriodResponse{}, nil
}

// VetoProposal allows the signer to veto a proposal.
// The proposal must be in the voting period, and the signer must be the designated Oversight DAO.
// If the proposal is vetoed, it will be removed from the active proposals queue and rejected.
func (ms MsgServer) VetoProposal(goCtx context.Context, msg *types.MsgVetoProposal) (*types.MsgVetoProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := ms.k.GetParams(ctx)

	logger := ms.k.Logger(ctx)

	if params.OversightDaoAddress == "" {
		logger.Info("Oversight DAO address is not set, function is disabled")

		return nil, errors.Wrapf(types.ErrFunctionDisabled, "Oversight DAO address is not set")
	}

	if msg.Vetoer != params.OversightDaoAddress {
		logger.Error(
			"invalid authority for vetoing proposal",
			"expected", params.OversightDaoAddress,
			"got", msg.Vetoer,
		)

		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", params.OversightDaoAddress, msg.Vetoer)
	}

	proposal, found := ms.k.govKeeper.GetProposal(ctx, msg.ProposalId)
	if !found {
		logger.Error(
			"proposal not found",
			"proposal_id", msg.ProposalId,
			"authority", msg.Vetoer,
		)

		return nil, errors.Wrapf(govtypes.ErrUnknownProposal, "proposal with ID %d not found", msg.ProposalId)
	}
	if proposal.Status != govtypesv1.StatusVotingPeriod {
		logger.Error(
			"proposal is not in voting period",
			"proposal", proposal.Id,
			"status", proposal.Status,
			"authority", msg.Vetoer,
		)

		return nil, errors.Wrapf(sdkgovtypes.ErrInactiveProposal, "proposal with ID %d is not in voting period", msg.ProposalId)
	}

	// follows the same logic as in x/gov/abci.go for rejected proposals
	if msg.BurnDeposit {
		ms.k.govKeeper.DeleteAndBurnDeposits(ctx, proposal.Id)
	} else {
		ms.k.govKeeper.RefundAndDeleteDeposits(ctx, proposal.Id)
	}
	proposal.Status = govtypesv1.StatusVetoed

	// Since the proposal is veoted, we set the final tally result to an empty tally.
	emptyTally := govtypesv1.EmptyTallyResult()
	proposal.FinalTallyResult = &emptyTally

	ms.k.govKeeper.SetProposal(ctx, proposal)
	ms.k.govKeeper.DeleteVotes(ctx, proposal.Id)
	ms.k.govKeeper.RemoveFromActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)

	ms.k.govKeeper.UpdateMinInitialDeposit(ctx, true)
	ms.k.govKeeper.UpdateMinDeposit(ctx, true)

	logger.Info(
		"proposal vetoed",
		"proposal", proposal.Id,
		"authority", msg.Vetoer,
	)

	// Emit event for proposal veto
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeVetoProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
			sdk.NewAttribute(types.AttributeKeySigner, msg.Vetoer),
			sdk.NewAttribute(sdkgovtypes.AttributeKeyProposalResult, types.AttributeValueProposalVetoed),
		),
	})

	return &types.MsgVetoProposalResponse{}, nil
}
