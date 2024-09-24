package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors1 "github.com/cosmos/cosmos-sdk/types/errors"

	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the gov MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) v1.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ v1.MsgServer = msgServer{}

// SubmitProposal implements the MsgServer.SubmitProposal method.
func (k msgServer) SubmitProposal(goCtx context.Context, msg *v1.MsgSubmitProposal) (*v1.MsgSubmitProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	initialDeposit := msg.GetInitialDeposit()

	if err := k.validateInitialDeposit(ctx, initialDeposit); err != nil {
		return nil, err
	}

	proposalMsgs, err := msg.GetMsgs()
	if err != nil {
		return nil, err
	}

	proposer, err := sdk.AccAddressFromBech32(msg.GetProposer())
	if err != nil {
		return nil, err
	}

	proposal, err := k.Keeper.SubmitProposal(ctx, proposalMsgs, msg.Metadata, msg.Title, msg.Summary, proposer)
	if err != nil {
		return nil, err
	}

	bytes, err := proposal.Marshal()
	if err != nil {
		return nil, err
	}

	// ref: https://github.com/cosmos/cosmos-sdk/issues/9683
	ctx.GasMeter().ConsumeGas(
		3*ctx.KVGasConfig().WriteCostPerByte*uint64(len(bytes)),
		"submit proposal",
	)

	votingStarted, err := k.Keeper.AddDeposit(ctx, proposal.Id, proposer, msg.GetInitialDeposit())
	if err != nil {
		return nil, err
	}

	if votingStarted {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(govtypes.EventTypeSubmitProposal,
				sdk.NewAttribute(govtypes.AttributeKeyVotingPeriodStart, fmt.Sprintf("%d", proposal.Id)),
			),
		)
	}

	return &v1.MsgSubmitProposalResponse{
		ProposalId: proposal.Id,
	}, nil
}

// ExecLegacyContent implements the MsgServer.ExecLegacyContent method.
func (k msgServer) ExecLegacyContent(goCtx context.Context, msg *v1.MsgExecLegacyContent) (*v1.MsgExecLegacyContentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	govAcct := k.GetGovernanceAccount(ctx).GetAddress().String()
	if govAcct != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "expected %s got %s", govAcct, msg.Authority)
	}

	content, err := v1.LegacyContentFromMessage(msg)
	if err != nil {
		return nil, errors.Wrapf(govtypes.ErrInvalidProposalContent, "%+v", err)
	}

	// Ensure that the content has a respective handler
	if !k.Keeper.legacyRouter.HasRoute(content.ProposalRoute()) {
		return nil, errors.Wrap(govtypes.ErrNoProposalHandlerExists, content.ProposalRoute())
	}

	handler := k.Keeper.legacyRouter.GetRoute(content.ProposalRoute())
	if err := handler(ctx, content); err != nil {
		return nil, errors.Wrapf(govtypes.ErrInvalidProposalContent, "failed to run legacy handler %s, %+v", content.ProposalRoute(), err)
	}

	return &v1.MsgExecLegacyContentResponse{}, nil
}

// Vote implements the MsgServer.Vote method.
func (k msgServer) Vote(goCtx context.Context, msg *v1.MsgVote) (*v1.MsgVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	accAddr, err := sdk.AccAddressFromBech32(msg.Voter)
	if err != nil {
		return nil, err
	}
	err = k.Keeper.AddVote(ctx, msg.ProposalId, accAddr, v1.NewNonSplitVoteOption(msg.Option), msg.Metadata)
	if err != nil {
		return nil, err
	}

	return &v1.MsgVoteResponse{}, nil
}

// VoteWeighted implements the MsgServer.VoteWeighted method.
func (k msgServer) VoteWeighted(goCtx context.Context, msg *v1.MsgVoteWeighted) (*v1.MsgVoteWeightedResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	accAddr, accErr := sdk.AccAddressFromBech32(msg.Voter)
	if accErr != nil {
		return nil, accErr
	}
	err := k.Keeper.AddVote(ctx, msg.ProposalId, accAddr, msg.Options, msg.Metadata)
	if err != nil {
		return nil, err
	}

	return &v1.MsgVoteWeightedResponse{}, nil
}

// Deposit implements the MsgServer.Deposit method.
func (k msgServer) Deposit(goCtx context.Context, msg *v1.MsgDeposit) (*v1.MsgDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	accAddr, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, err
	}

	if err := validateDeposit(msg.Amount); err != nil {
		return nil, err
	}

	votingStarted, err := k.Keeper.AddDeposit(ctx, msg.ProposalId, accAddr, msg.Amount)
	if err != nil {
		return nil, err
	}

	if votingStarted {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				govtypes.EventTypeProposalDeposit,
				sdk.NewAttribute(govtypes.AttributeKeyVotingPeriodStart, fmt.Sprintf("%d", msg.ProposalId)),
			),
		)
	}

	return &v1.MsgDepositResponse{}, nil
}

// validateDeposit validates the deposit amount, do not use for initial deposit.
func validateDeposit(amount sdk.Coins) error {
	if !amount.IsValid() || !amount.IsAllPositive() {
		return sdkerrors1.ErrInvalidCoins.Wrap(amount.String())
	}

	return nil
}

// UpdateParams implements the MsgServer.UpdateParams method.
func (k msgServer) UpdateParams(goCtx context.Context, msg *v1.MsgUpdateParams) (*v1.MsgUpdateParamsResponse, error) {
	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &v1.MsgUpdateParamsResponse{}, nil
}

// ProposeLaw implements the MsgServer.ProposeLaw method.
func (k msgServer) ProposeLaw(goCtx context.Context, msg *v1.MsgProposeLaw) (*v1.MsgProposeLawResponse, error) {
	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}
	// only a no-op for now
	return &v1.MsgProposeLawResponse{}, nil
}

// ProposeConstitutionAmendment implements the MsgServer.ProposeConstitutionAmendment method.
func (k msgServer) ProposeConstitutionAmendment(goCtx context.Context, msg *v1.MsgProposeConstitutionAmendment) (*v1.MsgProposeConstitutionAmendmentResponse, error) {
	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}
	// only a no-op for now
	return &v1.MsgProposeConstitutionAmendmentResponse{}, nil
}

func (k msgServer) CreateGovernor(goCtx context.Context, msg *v1.MsgCreateGovernor) (*v1.MsgCreateGovernorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Ensure the governor does not already exist
	addr := sdk.MustAccAddressFromBech32(msg.Address)
	govAddr := govtypes.GovernorAddress(addr.Bytes())
	if _, found := k.GetGovernor(ctx, govAddr); found {
		return nil, govtypes.ErrGovernorExists
	}

	// Ensure the governor has a valid description
	if _, err := msg.Description.EnsureLength(); err != nil {
		return nil, err
	}

	minSelfDelegation, _ := math.NewIntFromString(k.GetParams(ctx).MinGovernorSelfDelegation)
	bondedTokens := k.getGovernorBondedTokens(ctx, govAddr)
	if bondedTokens.LT(minSelfDelegation) {
		return nil, govtypes.ErrInsufficientGovernorDelegation.Wrapf("minimum self-delegation required: %s, total bonded tokens: %s", minSelfDelegation, bondedTokens)
	}

	governor, err := v1.NewGovernor(govAddr.String(), msg.Description, ctx.BlockTime())
	if err != nil {
		return nil, err
	}

	k.SetGovernor(ctx, governor)

	// a base account automatically creates a governance delegation to itself
	k.DelegateToGovernor(ctx, addr, govAddr)

	return &v1.MsgCreateGovernorResponse{}, nil
}

func (k msgServer) EditGovernor(goCtx context.Context, msg *v1.MsgEditGovernor) (*v1.MsgEditGovernorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Ensure the governor exists
	addr := sdk.MustAccAddressFromBech32(msg.Address)
	govAddr := govtypes.GovernorAddress(addr.Bytes())
	governor, found := k.GetGovernor(ctx, govAddr)
	if !found {
		return nil, govtypes.ErrUnknownGovernor
	}

	// Ensure the governor has a valid description
	if _, err := msg.Description.EnsureLength(); err != nil {
		return nil, err
	}

	// Update the governor
	governor.Description = msg.Description
	k.SetGovernor(ctx, governor)

	return &v1.MsgEditGovernorResponse{}, nil
}

func (k msgServer) UpdateGovernorStatus(goCtx context.Context, msg *v1.MsgUpdateGovernorStatus) (*v1.MsgUpdateGovernorStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Ensure the governor exists
	addr := sdk.MustAccAddressFromBech32(msg.Address)
	govAddr := govtypes.GovernorAddress(addr.Bytes())
	governor, found := k.GetGovernor(ctx, govAddr)
	if !found {
		return nil, govtypes.ErrUnknownGovernor
	}

	if !msg.Status.IsValid() {
		return nil, govtypes.ErrInvalidGovernorStatus
	}

	// Ensure the governor is not already in the desired status
	if governor.Status == msg.Status {
		return nil, govtypes.ErrGovernorStatusEqual
	}

	// Ensure the governor has been in the current status for the required period
	governorStatusChangePeriod := *k.GetParams(ctx).GovernorStatusChangePeriod
	changeTime := ctx.BlockTime()
	if governor.LastStatusChangeTime.Add(governorStatusChangePeriod).Before(changeTime) {
		return nil, govtypes.ErrGovernorStatusChangePeriod.Wrapf("last status change time: %s, need to wait until: %s", governor.LastStatusChangeTime, governor.LastStatusChangeTime.Add(governorStatusChangePeriod))
	}

	// Update the governor status
	governor.Status = msg.Status
	governor.LastStatusChangeTime = &changeTime
	k.SetGovernor(ctx, governor)
	// if status changes to active, create governance self-delegation
	// in case it didn't exist
	if governor.IsActive() {
		k.RedelegateToGovernor(ctx, addr, govAddr)
	}
	return &v1.MsgUpdateGovernorStatusResponse{}, nil
}

func (k msgServer) DelegateGovernor(goCtx context.Context, msg *v1.MsgDelegateGovernor) (*v1.MsgDelegateGovernorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	govAddr := govtypes.MustGovernorAddressFromBech32(msg.GovernorAddress)

	// Ensure the delegator is not already an active governor, as they cannot delegate
	if g, found := k.GetGovernor(ctx, govtypes.GovernorAddress(delAddr.Bytes())); found && g.IsActive() {
		return nil, govtypes.ErrDelegatorIsGovernor
	}

	// Ensure the delegation is not already present
	gd, found := k.GetGovernanceDelegation(ctx, delAddr)
	if found && govAddr.Equals(govtypes.MustGovernorAddressFromBech32(gd.GovernorAddress)) {
		return nil, govtypes.ErrGovernanceDelegationExists
	}
	// redelegate if a delegation to another governor already exists
	if found {
		k.RedelegateToGovernor(ctx, delAddr, govAddr)
	} else {
		// Create the delegation
		k.DelegateToGovernor(ctx, delAddr, govAddr)
	}

	return &v1.MsgDelegateGovernorResponse{}, nil
}

func (k msgServer) UndelegateGovernor(goCtx context.Context, msg *v1.MsgUndelegateGovernor) (*v1.MsgUndelegateGovernorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)

	// Ensure the delegation exists
	_, found := k.GetGovernanceDelegation(ctx, delAddr)
	if !found {
		return nil, govtypes.ErrUnknownGovernanceDelegation
	}

	// Remove the delegation
	k.UndelegateFromGovernor(ctx, delAddr)

	return &v1.MsgUndelegateGovernorResponse{}, nil
}

type legacyMsgServer struct {
	govAcct string
	server  v1.MsgServer
}

// NewLegacyMsgServerImpl returns an implementation of the v1beta1 legacy MsgServer interface. It wraps around
// the current MsgServer
func NewLegacyMsgServerImpl(govAcct string, v1Server v1.MsgServer) v1beta1.MsgServer {
	return &legacyMsgServer{govAcct: govAcct, server: v1Server}
}

var _ v1beta1.MsgServer = legacyMsgServer{}

func (k legacyMsgServer) SubmitProposal(goCtx context.Context, msg *v1beta1.MsgSubmitProposal) (*v1beta1.MsgSubmitProposalResponse, error) {
	contentMsg, err := v1.NewLegacyContent(msg.GetContent(), k.govAcct)
	if err != nil {
		return nil, fmt.Errorf("error converting legacy content into proposal message: %w", err)
	}

	proposal, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{contentMsg},
		msg.InitialDeposit,
		msg.Proposer,
		"",
		msg.GetContent().GetTitle(),
		msg.GetContent().GetDescription(),
	)
	if err != nil {
		return nil, err
	}

	resp, err := k.server.SubmitProposal(goCtx, proposal)
	if err != nil {
		return nil, err
	}

	return &v1beta1.MsgSubmitProposalResponse{ProposalId: resp.ProposalId}, nil
}

func (k legacyMsgServer) Vote(goCtx context.Context, msg *v1beta1.MsgVote) (*v1beta1.MsgVoteResponse, error) {
	_, err := k.server.Vote(goCtx, &v1.MsgVote{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Option:     v1.VoteOption(msg.Option),
	})
	if err != nil {
		return nil, err
	}
	return &v1beta1.MsgVoteResponse{}, nil
}

func (k legacyMsgServer) VoteWeighted(goCtx context.Context, msg *v1beta1.MsgVoteWeighted) (*v1beta1.MsgVoteWeightedResponse, error) {
	opts := make([]*v1.WeightedVoteOption, len(msg.Options))
	for idx, opt := range msg.Options {
		opts[idx] = &v1.WeightedVoteOption{
			Option: v1.VoteOption(opt.Option),
			Weight: opt.Weight.String(),
		}
	}

	_, err := k.server.VoteWeighted(goCtx, &v1.MsgVoteWeighted{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Options:    opts,
	})
	if err != nil {
		return nil, err
	}
	return &v1beta1.MsgVoteWeightedResponse{}, nil
}

func (k legacyMsgServer) Deposit(goCtx context.Context, msg *v1beta1.MsgDeposit) (*v1beta1.MsgDepositResponse, error) {
	_, err := k.server.Deposit(goCtx, &v1.MsgDeposit{
		ProposalId: msg.ProposalId,
		Depositor:  msg.Depositor,
		Amount:     msg.Amount,
	})
	if err != nil {
		return nil, err
	}
	return &v1beta1.MsgDepositResponse{}, nil
}
