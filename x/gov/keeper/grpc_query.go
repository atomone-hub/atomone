package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	sdkgovv3 "github.com/cosmos/cosmos-sdk/x/gov/migrations/v3"
	v3 "github.com/cosmos/cosmos-sdk/x/gov/migrations/v3"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

type queryServer struct {
	sdkgovv1.QueryServer
}

var _ v1.QueryServer = queryServer{}

func (q queryServer) Constitution(c context.Context, req *v1.QueryConstitutionRequest) (*v1.QueryConstitutionResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	constitution := q.QueryServer.GetConstitution(ctx)

	return &v1.QueryConstitutionResponse{Constitution: constitution}, nil
}

// Proposal returns proposal details based on ProposalID
func (q queryServer) Proposal(c context.Context, req *v1.QueryProposalRequest) (*v1.QueryProposalResponse, error) {
	resp, err := q.QueryServer.Proposal(c, mapProposalRequest(req))
	if err != nil {
		return nil, err
	}

	return mapProposalResponse(resp), nil
}

// Proposals implements the Query/Proposals gRPC method
func (q queryServer) Proposals(c context.Context, req *v1.QueryProposalsRequest) (*v1.QueryProposalsResponse, error) {
	resp, err := q.QueryServer.Proposals(c, mapProposalsRequest(req))
	if err != nil {
		return nil, err
	}

	return mapProposalsResponse(resp), nil
}

// Vote returns Voted information based on proposalID, voterAddr
func (q queryServer) Vote(c context.Context, req *v1.QueryVoteRequest) (*v1.QueryVoteResponse, error) {
	resp, err := q.QueryServer.Vote(c, mapVoteRequest(req))
	if err != nil {
		return nil, err
	}

	return mapQueryVoteResponse(resp), nil
}

// Votes returns single proposal's votes
func (q queryServer) Votes(c context.Context, req *v1.QueryVotesRequest) (*v1.QueryVotesResponse, error) {
	resp, err := q.QueryServer.Votes(c, mapVotesRequest(req))
	if err != nil {
		return nil, err
	}

	return mapVotesResponse(resp), nil
}

// Params queries all params
func (q queryServer) Params(c context.Context, req *v1.QueryParamsRequest) (*v1.QueryParamsResponse, error) {
	resp, err := q.QueryServer.Params(c, mapParamsRequest(req))
	if err != nil {
		return nil, err
	}

	return mapParamsResponse(resp), nil
}

// Deposit queries single deposit information based on proposalID, depositAddr.
func (q queryServer) Deposit(c context.Context, req *v1.QueryDepositRequest) (*v1.QueryDepositResponse, error) {
	resp, err := q.QueryServer.Deposit(c, mapDepositRequest(req))
	if err != nil {
		return nil, err
	}

	return mapQueryDepositResponse(resp), nil
}

// Deposits returns single proposal's all deposits
func (q queryServer) Deposits(c context.Context, req *v1.QueryDepositsRequest) (*v1.QueryDepositsResponse, error) {
	resp, err := q.QueryServer.Deposits(c, mapDepositsRequest(req))
	if err != nil {
		return nil, err
	}

	return mapDepositsResponse(resp), nil
}

// TallyResult queries the tally of a proposal vote
func (q queryServer) TallyResult(c context.Context, req *v1.QueryTallyResultRequest) (*v1.QueryTallyResultResponse, error) {
	resp, err := q.QueryServer.TallyResult(c, mapTallyResultRequest(req))
	if err != nil {
		return nil, err
	}

	return mapTallyResultResponse(resp), nil
}

// MinDeposit returns the minimum deposit currently required for a proposal to enter voting period
func (q queryServer) MinDeposit(c context.Context, req *v1.QueryMinDepositRequest) (*v1.QueryMinDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	minDeposit := q.GetMinDeposit(ctx)

	return &v1.QueryMinDepositResponse{MinDeposit: minDeposit}, nil
}

// MinInitialDeposit returns the minimum deposit required for a proposal to be submitted
func (q queryServer) MinInitialDeposit(c context.Context, req *v1.QueryMinInitialDepositRequest) (*v1.QueryMinInitialDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	minInitialDeposit := q.GetMinInitialDeposit(ctx)

	return &v1.QueryMinInitialDepositResponse{MinInitialDeposit: minInitialDeposit}, nil
}

// Quorums returns the current quorums
func (q queryServer) Quorums(c context.Context, _ *v1.QueryQuorumsRequest) (*v1.QueryQuorumsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &v1.QueryQuorumsResponse{
		Quorum:                      q.GetQuorum(ctx).String(),
		ConstitutionAmendmentQuorum: q.GetConstitutionAmendmentQuorum(ctx).String(),
		LawQuorum:                   q.GetLawQuorum(ctx).String(),
	}, nil
}

// ParticipationEMAs queries the state of the proposal participation exponential moving averages.
func (q queryServer) ParticipationEMAs(c context.Context, _ *v1.QueryParticipationEMAsRequest) (*v1.QueryParticipationEMAsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &v1.QueryParticipationEMAsResponse{
		ParticipationEma:                      q.GetParticipationEMA(ctx).String(),
		ConstitutionAmendmentParticipationEma: q.GetConstitutionAmendmentParticipationEMA(ctx).String(),
		LawParticipationEma:                   q.GetLawParticipationEMA(ctx).String(),
	}, nil
}

var _ v1beta1.QueryServer = legacyQueryServer{}

type legacyQueryServer struct {
	keeper *keeper.Keeper
}

// NewLegacyQueryServer returns an implementation of the v1beta1 legacy QueryServer interface.
func NewLegacyQueryServer(k *keeper.Keeper) v1beta1.QueryServer {
	return &legacyQueryServer{keeper: k}
}

func (q legacyQueryServer) Proposal(c context.Context, req *v1beta1.QueryProposalRequest) (*v1beta1.QueryProposalResponse, error) {
	resp, err := q.keeper.Proposal(c, &v1.QueryProposalRequest{
		ProposalId: req.ProposalId,
	})
	if err != nil {
		return nil, err
	}

	proposal, err := sdkgovv3.ConvertToLegacyProposal(*resp.Proposal)
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryProposalResponse{Proposal: proposal}, nil
}

func (q legacyQueryServer) Proposals(c context.Context, req *v1beta1.QueryProposalsRequest) (*v1beta1.QueryProposalsResponse, error) {
	resp, err := q.keeper.Proposals(c, &v1.QueryProposalsRequest{
		ProposalStatus: v1.ProposalStatus(req.ProposalStatus),
		Voter:          req.Voter,
		Depositor:      req.Depositor,
		Pagination:     req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	legacyProposals := make([]v1beta1.Proposal, len(resp.Proposals))
	for idx, proposal := range resp.Proposals {
		legacyProposal, err := sdkgovv3.ConvertToLegacyProposal(*proposal)
		if err != nil {
			return nil, err
		}

		legacyProposals[idx] = legacyProposal
	}

	return &v1beta1.QueryProposalsResponse{
		Proposals:  legacyProposals,
		Pagination: resp.Pagination,
	}, nil
}

func (q legacyQueryServer) Vote(c context.Context, req *v1beta1.QueryVoteRequest) (*v1beta1.QueryVoteResponse, error) {
	resp, err := q.keeper.Vote(c, &v1.QueryVoteRequest{
		ProposalId: req.ProposalId,
		Voter:      req.Voter,
	})
	if err != nil {
		return nil, err
	}

	vote, err := sdkgovv3.ConvertToLegacyVote(*resp.Vote)
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryVoteResponse{Vote: vote}, nil
}

func (q legacyQueryServer) Votes(c context.Context, req *v1beta1.QueryVotesRequest) (*v1beta1.QueryVotesResponse, error) {
	resp, err := q.keeper.Votes(c, &v1.QueryVotesRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	votes := make([]v1beta1.Vote, len(resp.Votes))
	for i, v := range resp.Votes {
		votes[i], err = sdkgovv3.ConvertToLegacyVote(*v)
		if err != nil {
			return nil, err
		}
	}

	return &v1beta1.QueryVotesResponse{
		Votes:      votes,
		Pagination: resp.Pagination,
	}, nil
}

//nolint:staticcheck
func (q legacyQueryServer) Params(c context.Context, req *v1beta1.QueryParamsRequest) (*v1beta1.QueryParamsResponse, error) {
	resp, err := q.keeper.Params(c, &v1.QueryParamsRequest{
		ParamsType: req.ParamsType,
	})
	if err != nil {
		return nil, err
	}

	response := &v1beta1.QueryParamsResponse{}

	if resp.DepositParams != nil {
		minDeposit := sdk.NewCoins(resp.DepositParams.MinDeposit...)
		response.DepositParams = v1beta1.NewDepositParams(minDeposit, *resp.DepositParams.MaxDepositPeriod)
	}

	if resp.VotingParams != nil {
		response.VotingParams = v1beta1.NewVotingParams(*resp.VotingParams.VotingPeriod)
	}

	if resp.TallyParams != nil {
		quorumRes, err := q.keeper.Quorums(c, &v1.QueryQuorumsRequest{})
		if err != nil {
			return nil, err
		}
		threshold, err := math.LegacyNewDecFromStr(resp.TallyParams.Threshold)
		if err != nil {
			return nil, err
		}
		quorum := math.LegacyMustNewDecFromStr(quorumRes.Quorum)
		response.TallyParams = v1beta1.NewTallyParams(quorum, threshold, math.LegacyZeroDec())
	}

	return response, nil
}

func (q legacyQueryServer) Deposit(c context.Context, req *v1beta1.QueryDepositRequest) (*v1beta1.QueryDepositResponse, error) {
	resp, err := q.keeper.Deposit(c, &v1.QueryDepositRequest{
		ProposalId: req.ProposalId,
		Depositor:  req.Depositor,
	})
	if err != nil {
		return nil, err
	}

	deposit := v3.ConvertToLegacyDeposit(resp.Deposit)
	return &v1beta1.QueryDepositResponse{Deposit: deposit}, nil
}

func (q legacyQueryServer) Deposits(c context.Context, req *v1beta1.QueryDepositsRequest) (*v1beta1.QueryDepositsResponse, error) {
	resp, err := q.keeper.Deposits(c, &v1.QueryDepositsRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}
	deposits := make([]v1beta1.Deposit, len(resp.Deposits))
	for idx, deposit := range resp.Deposits {
		deposits[idx] = v3.ConvertToLegacyDeposit(deposit)
	}

	return &v1beta1.QueryDepositsResponse{Deposits: deposits, Pagination: resp.Pagination}, nil
}

func (q legacyQueryServer) TallyResult(c context.Context, req *v1beta1.QueryTallyResultRequest) (*v1beta1.QueryTallyResultResponse, error) {
	resp, err := q.keeper.TallyResult(c, &v1.QueryTallyResultRequest{
		ProposalId: req.ProposalId,
	})
	if err != nil {
		return nil, err
	}

	tally, err := v3.ConvertToLegacyTallyResult(resp.Tally)
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryTallyResultResponse{Tally: tally}, nil
}
