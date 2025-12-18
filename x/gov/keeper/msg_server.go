package keeper

import (
	"context"

	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	sdkv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	sdkv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

type msgServer struct {
	sdkv1.MsgServer
}

// NewMsgServerImpl returns an implementation of the gov MsgServer interface
// for the provided Keeper.
// We return an private type, as we do not want to do type casting in module.
// Making it public adds no benefits.
func NewMsgServerImpl(k *Keeper) *msgServer {
	return &msgServer{MsgServer: govkeeper.NewMsgServerImpl(k.Keeper)}
}

var _ v1.MsgServer = msgServer{}

// SubmitProposal implements the MsgServer.SubmitProposal method.
func (k msgServer) SubmitProposal(ctx context.Context, msg *v1.MsgSubmitProposal) (*v1.MsgSubmitProposalResponse, error) {
	result, err := k.MsgServer.SubmitProposal(ctx, &sdkv1.MsgSubmitProposal{
		Messages:       msg.GetMessages(),
		InitialDeposit: msg.GetInitialDeposit(),
		Proposer:       msg.GetProposer(),
		Metadata:       msg.GetMetadata(),
		Title:          msg.GetTitle(),
		Summary:        msg.GetSummary(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgSubmitProposalResponse{
		ProposalId: result.GetProposalId(),
	}, nil
}

// ExecLegacyContent implements the MsgServer.ExecLegacyContent method.
func (k msgServer) ExecLegacyContent(ctx context.Context, msg *v1.MsgExecLegacyContent) (*v1.MsgExecLegacyContentResponse, error) {
	_, err := k.MsgServer.ExecLegacyContent(ctx, &sdkv1.MsgExecLegacyContent{
		Content:   msg.GetContent(),
		Authority: msg.GetAuthority(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgExecLegacyContentResponse{}, nil
}

// Vote implements the MsgServer.Vote method.
func (k msgServer) Vote(ctx context.Context, msg *v1.MsgVote) (*v1.MsgVoteResponse, error) {
	_, err := k.MsgServer.Vote(ctx, &sdkv1.MsgVote{
		ProposalId: msg.GetProposalId(),
		Voter:      msg.GetVoter(),
		Option:     sdkv1.VoteOption(msg.Option),
		Metadata:   msg.GetMetadata(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgVoteResponse{}, nil
}

// VoteWeighted implements the MsgServer.VoteWeighted method.
func (k msgServer) VoteWeighted(ctx context.Context, msg *v1.MsgVoteWeighted) (*v1.MsgVoteWeightedResponse, error) {
	_, err := k.MsgServer.VoteWeighted(ctx, &sdkv1.MsgVoteWeighted{
		ProposalId: msg.GetProposalId(),
		Voter:      msg.GetVoter(),
		Options:    v1.ConvertAtomOneWeightedVoteOptionsToSDK(msg.GetOptions()),
		Metadata:   msg.GetMetadata(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgVoteWeightedResponse{}, nil
}

// Deposit implements the MsgServer.Deposit method.
func (k msgServer) Deposit(ctx context.Context, msg *v1.MsgDeposit) (*v1.MsgDepositResponse, error) {
	_, err := k.MsgServer.Deposit(ctx, &sdkv1.MsgDeposit{
		ProposalId: msg.GetProposalId(),
		Depositor:  msg.GetDepositor(),
		Amount:     msg.GetAmount(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgDepositResponse{}, nil
}

// UpdateParams implements the MsgServer.UpdateParams method.
func (k msgServer) UpdateParams(ctx context.Context, msg *v1.MsgUpdateParams) (*v1.MsgUpdateParamsResponse, error) {
	_, err := k.MsgServer.UpdateParams(ctx, &sdkv1.MsgUpdateParams{
		Authority: msg.GetAuthority(),
		Params:    *v1.ConvertAtomOneParamsToSDK(&msg.Params),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgUpdateParamsResponse{}, nil
}

// ProposeLaw implements the MsgServer.ProposeLaw method.
func (k msgServer) ProposeLaw(ctx context.Context, msg *v1.MsgProposeLaw) (*v1.MsgProposeLawResponse, error) {
	_, err := k.MsgServer.ProposeLaw(ctx, &sdkv1.MsgProposeLaw{
		Authority: msg.GetAuthority(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgProposeLawResponse{}, nil
}

// ProposeConstitutionAmendment implements the MsgServer.ProposeConstitutionAmendment method.
func (k msgServer) ProposeConstitutionAmendment(ctx context.Context, msg *v1.MsgProposeConstitutionAmendment) (*v1.MsgProposeConstitutionAmendmentResponse, error) {
	_, err := k.MsgServer.ProposeConstitutionAmendment(ctx, &sdkv1.MsgProposeConstitutionAmendment{
		Authority: msg.GetAuthority(),
		Amendment: msg.GetAmendment(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgProposeConstitutionAmendmentResponse{}, nil
}

func (k msgServer) CreateGovernor(goCtx context.Context, msg *v1.MsgCreateGovernor) (*v1.MsgCreateGovernorResponse, error) {
	_, err := k.MsgServer.CreateGovernor(goCtx, &sdkv1.MsgCreateGovernor{
		Address:     msg.GetAddress(),
		Description: msg.GetDescription(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgCreateGovernorResponse{}, nil
}

func (k msgServer) EditGovernor(goCtx context.Context, msg *v1.MsgEditGovernor) (*v1.MsgEditGovernorResponse, error) {
	_, err := k.MsgServer.EditGovernor(goCtx, &sdkv1.MsgEditGovernor{
		Address:     msg.GetAddress(),
		Description: msg.GetDescription(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgEditGovernorResponse{}, nil
}

func (k msgServer) UpdateGovernorStatus(goCtx context.Context, msg *v1.MsgUpdateGovernorStatus) (*v1.MsgUpdateGovernorStatusResponse, error) {
	_, err := k.MsgServer.UpdateGovernorStatus(goCtx, &sdkv1.MsgUpdateGovernorStatus{
		Address: msg.GetAddress(),
		Status:  sdkv1.GovernorStatus(msg.GetStatus()),
	})

	return &v1.MsgUpdateGovernorStatusResponse{}, nil
}

func (k msgServer) DelegateGovernor(goCtx context.Context, msg *v1.MsgDelegateGovernor) (*v1.MsgDelegateGovernorResponse, error) {
	_, err := k.MsgServer.DelegateGovernor(goCtx, &sdkv1.MsgDelegateGovernor{
		DelegatorAddress: msg.GetDelegatorAddress(),
		GovernorAddress:  msg.GetGovernorAddress(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgDelegateGovernorResponse{}, nil
}

func (k msgServer) UndelegateGovernor(goCtx context.Context, msg *v1.MsgUndelegateGovernor) (*v1.MsgUndelegateGovernorResponse, error) {
	_, err := k.MsgServer.UndelegateGovernor(goCtx, &sdkv1.MsgUndelegateGovernor{
		DelegatorAddress: msg.GetDelegatorAddress(),
	})
	if err != nil {
		return nil, err
	}

	return &v1.MsgUndelegateGovernorResponse{}, nil
}

type legacyMsgServer struct {
	sdkv1beta1.MsgServer
}

// NewLegacyMsgServerImpl returns an implementation of the v1beta1 legacy MsgServer interface. It wraps around
// the SDK's v1beta1 MsgServer
func NewLegacyMsgServerImpl(govAcct string, msgServer *msgServer) v1beta1.MsgServer {
	return &legacyMsgServer{
		MsgServer: govkeeper.NewLegacyMsgServerImpl(govAcct, msgServer.MsgServer),
	}
}

var _ v1beta1.MsgServer = legacyMsgServer{}

func (k legacyMsgServer) SubmitProposal(goCtx context.Context, msg *v1beta1.MsgSubmitProposal) (*v1beta1.MsgSubmitProposalResponse, error) {
	sdkMsg := &sdkv1beta1.MsgSubmitProposal{
		Content:        msg.Content,
		InitialDeposit: msg.InitialDeposit,
		Proposer:       msg.Proposer,
	}

	resp, err := k.MsgServer.SubmitProposal(goCtx, sdkMsg)
	if err != nil {
		return nil, err
	}

	return &v1beta1.MsgSubmitProposalResponse{ProposalId: resp.ProposalId}, nil
}

func (k legacyMsgServer) Vote(goCtx context.Context, msg *v1beta1.MsgVote) (*v1beta1.MsgVoteResponse, error) {
	sdkMsg := &sdkv1beta1.MsgVote{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Option:     sdkv1beta1.VoteOption(msg.Option),
	}

	_, err := k.MsgServer.Vote(goCtx, sdkMsg)
	if err != nil {
		return nil, err
	}

	return &v1beta1.MsgVoteResponse{}, nil
}

func (k legacyMsgServer) VoteWeighted(goCtx context.Context, msg *v1beta1.MsgVoteWeighted) (*v1beta1.MsgVoteWeightedResponse, error) {
	options := make([]sdkv1beta1.WeightedVoteOption, len(msg.Options))
	for i, opt := range msg.Options {
		options[i] = sdkv1beta1.WeightedVoteOption{
			Option: sdkv1beta1.VoteOption(opt.Option),
			Weight: opt.Weight,
		}
	}

	sdkMsg := &sdkv1beta1.MsgVoteWeighted{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Options:    options,
	}

	_, err := k.MsgServer.VoteWeighted(goCtx, sdkMsg)
	if err != nil {
		return nil, err
	}

	return &v1beta1.MsgVoteWeightedResponse{}, nil
}

func (k legacyMsgServer) Deposit(goCtx context.Context, msg *v1beta1.MsgDeposit) (*v1beta1.MsgDepositResponse, error) {
	sdkMsg := &sdkv1beta1.MsgDeposit{
		ProposalId: msg.ProposalId,
		Depositor:  msg.Depositor,
		Amount:     msg.Amount,
	}

	_, err := k.MsgServer.Deposit(goCtx, sdkMsg)
	if err != nil {
		return nil, err
	}

	return &v1beta1.MsgDepositResponse{}, nil
}
