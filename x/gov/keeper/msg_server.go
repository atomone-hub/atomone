package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"

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
func (k msgServer) SubmitProposal(ctx context.Context, msg *v1.MsgSubmitProposal) (*v1.MsgSubmitProposalResponse, error) {

	return &v1.MsgSubmitProposalResponse{
		ProposalId: proposal.Id,
	}, nil
}

// ExecLegacyContent implements the MsgServer.ExecLegacyContent method.
func (k msgServer) ExecLegacyContent(ctx context.Context, msg *v1.MsgExecLegacyContent) (*v1.MsgExecLegacyContentResponse, error) {

	return &v1.MsgExecLegacyContentResponse{}, nil
}

// Vote implements the MsgServer.Vote method.
func (k msgServer) Vote(ctx context.Context, msg *v1.MsgVote) (*v1.MsgVoteResponse, error) {

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
