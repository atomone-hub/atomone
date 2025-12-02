package keeper

import (
	"context"

	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	sdkv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

type msgServer struct {
	sdkv1.MsgServer
}

// NewMsgServerImpl returns an implementation of the gov MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(k *Keeper) v1.MsgServer {
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

	return &v1.MsgVoteWeightedResponse{}, nil
}

// Deposit implements the MsgServer.Deposit method.
func (k msgServer) Deposit(ctx context.Context, msg *v1.MsgDeposit) (*v1.MsgDepositResponse, error) {

	return &v1.MsgDepositResponse{}, nil
}

// UpdateParams implements the MsgServer.UpdateParams method.
func (k msgServer) UpdateParams(ctx context.Context, msg *v1.MsgUpdateParams) (*v1.MsgUpdateParamsResponse, error) {

	return &v1.MsgUpdateParamsResponse{}, nil
}

// ProposeLaw implements the MsgServer.ProposeLaw method.
func (k msgServer) ProposeLaw(ctx context.Context, msg *v1.MsgProposeLaw) (*v1.MsgProposeLawResponse, error) {

	// only a no-op for now
	return &v1.MsgProposeLawResponse{}, nil
}

// ProposeConstitutionAmendment implements the MsgServer.ProposeConstitutionAmendment method.
func (k msgServer) ProposeConstitutionAmendment(ctx context.Context, msg *v1.MsgProposeConstitutionAmendment) (*v1.MsgProposeConstitutionAmendmentResponse, error) {

	return &v1.MsgProposeConstitutionAmendmentResponse{}, nil
}

type legacyMsgServer struct {
	server v1.MsgServer
}

// NewLegacyMsgServerImpl returns an implementation of the v1beta1 legacy MsgServer interface. It wraps around
// the current MsgServer
func NewLegacyMsgServerImpl(v1Server v1.MsgServer) v1beta1.MsgServer {
	return &legacyMsgServer{server: v1Server}
}

var _ v1beta1.MsgServer = legacyMsgServer{}

func (k legacyMsgServer) SubmitProposal(goCtx context.Context, msg *v1beta1.MsgSubmitProposal) (*v1beta1.MsgSubmitProposalResponse, error) {

	return &v1beta1.MsgSubmitProposalResponse{ProposalId: resp.ProposalId}, nil
}

func (k legacyMsgServer) Vote(goCtx context.Context, msg *v1beta1.MsgVote) (*v1beta1.MsgVoteResponse, error) {

	return &v1beta1.MsgVoteResponse{}, nil
}

func (k legacyMsgServer) VoteWeighted(goCtx context.Context, msg *v1beta1.MsgVoteWeighted) (*v1beta1.MsgVoteWeightedResponse, error) {

	return &v1beta1.MsgVoteWeightedResponse{}, nil
}

func (k legacyMsgServer) Deposit(goCtx context.Context, msg *v1beta1.MsgDeposit) (*v1beta1.MsgDepositResponse, error) {

	return &v1beta1.MsgDepositResponse{}, nil
}
