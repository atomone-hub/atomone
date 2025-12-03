package keeper

import (
	"context"

	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	sdkv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	sdkv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

var _ v1.QueryServer = grpcServer{}

type grpcServer struct {
	sdkv1.QueryServer
}

func NewQueryServer(k *Keeper) v1.QueryServer {
	return &grpcServer{
		QueryServer: govkeeper.NewQueryServer(k.Keeper),
	}
}

func (q grpcServer) Constitution(ctx context.Context, _ *v1.QueryConstitutionRequest) (*v1.QueryConstitutionResponse, error) {
	result, err := q.QueryServer.Constitution(ctx, &sdkv1.QueryConstitutionRequest{})
	if err != nil {
		return nil, err
	}

	return &v1.QueryConstitutionResponse{Constitution: result.GetConstitution()}, nil
}

// Proposal returns proposal details based on ProposalID
func (q grpcServer) Proposal(ctx context.Context, req *v1.QueryProposalRequest) (*v1.QueryProposalResponse, error) {
	result, err := q.QueryServer.Proposal(ctx, &sdkv1.QueryProposalRequest{
		ProposalId: req.ProposalId,
	})
	if err != nil {
		return nil, err
	}

	return &v1.QueryProposalResponse{
		Proposal: v1.ConvertSDKProposalToAtomOne(result.GetProposal()),
	}, nil
}

// Proposals implements the Query/Proposals gRPC method
func (q grpcServer) Proposals(ctx context.Context, req *v1.QueryProposalsRequest) (*v1.QueryProposalsResponse, error) {
	result, err := q.QueryServer.Proposals(ctx, &sdkv1.QueryProposalsRequest{
		ProposalStatus: sdkv1.ProposalStatus(req.ProposalStatus),
		Voter:          req.Voter,
		Depositor:      req.Depositor,
		Pagination:     req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	return &v1.QueryProposalsResponse{
		Proposals:  v1.ConvertSDKProposalsToAtomOne(result.GetProposals()),
		Pagination: result.GetPagination(),
	}, nil
}

// Vote returns Voted information based on proposalID, voterAddr
func (q grpcServer) Vote(ctx context.Context, req *v1.QueryVoteRequest) (*v1.QueryVoteResponse, error) {
	result, err := q.QueryServer.Vote(ctx, &sdkv1.QueryVoteRequest{
		ProposalId: req.ProposalId,
		Voter:      req.Voter,
	})
	if err != nil {
		return nil, err
	}

	return &v1.QueryVoteResponse{Vote: v1.ConvertSDKVoteToAtomOne(result.GetVote())}, nil
}

// Votes returns single proposal's votes
func (q grpcServer) Votes(ctx context.Context, req *v1.QueryVotesRequest) (*v1.QueryVotesResponse, error) {
	result, err := q.QueryServer.Votes(ctx, &sdkv1.QueryVotesRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	return &v1.QueryVotesResponse{
		Votes:      v1.ConvertSDKVotesToAtomOne(result.GetVotes()),
		Pagination: result.GetPagination(),
	}, nil
}

// Params queries all params
func (q grpcServer) Params(ctx context.Context, req *v1.QueryParamsRequest) (*v1.QueryParamsResponse, error) {
	result, err := q.QueryServer.Params(ctx, &sdkv1.QueryParamsRequest{
		ParamsType: req.ParamsType,
	})
	if err != nil {
		return nil, err
	}

	return &v1.QueryParamsResponse{
		VotingParams:  v1.ConvertSDKVotingParamsToAtomOne(result.VotingParams),   //nolint:staticcheck // backward compat
		DepositParams: v1.ConvertSDKDepositParamsToAtomOne(result.DepositParams), //nolint:staticcheck // backward compat
		TallyParams:   v1.ConvertSDKTallyParamsToAtomOne(result.TallyParams),     //nolint:staticcheck // backward compat
		Params:        v1.ConvertSDKParamsToAtomOne(result.Params),
	}, nil
}

// Deposit queries single deposit information based on proposalID, depositAddr.
func (q grpcServer) Deposit(ctx context.Context, req *v1.QueryDepositRequest) (*v1.QueryDepositResponse, error) {
	result, err := q.QueryServer.Deposit(ctx, &sdkv1.QueryDepositRequest{
		ProposalId: req.ProposalId,
		Depositor:  req.Depositor,
	})
	if err != nil {
		return nil, err
	}

	return &v1.QueryDepositResponse{Deposit: v1.ConvertSDKDepositToAtomOne(result.GetDeposit())}, nil
}

// Deposits returns single proposal's all deposits
func (q grpcServer) Deposits(ctx context.Context, req *v1.QueryDepositsRequest) (*v1.QueryDepositsResponse, error) {
	result, err := q.QueryServer.Deposits(ctx, &sdkv1.QueryDepositsRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	return &v1.QueryDepositsResponse{
		Deposits:   v1.ConvertSDKDepositsToAtomOne(result.GetDeposits()),
		Pagination: result.GetPagination(),
	}, nil
}

// TallyResult queries the tally of a proposal vote
func (q grpcServer) TallyResult(ctx context.Context, req *v1.QueryTallyResultRequest) (*v1.QueryTallyResultResponse, error) {
	result, err := q.QueryServer.TallyResult(ctx, &sdkv1.QueryTallyResultRequest{
		ProposalId: req.ProposalId,
	})
	if err != nil {
		return nil, err
	}

	return &v1.QueryTallyResultResponse{Tally: v1.ConvertSDKTallyResultToAtomOne(result.GetTally())}, nil
}

// MinDeposit returns the minimum deposit currently required for a proposal to enter voting period
func (q grpcServer) MinDeposit(ctx context.Context, req *v1.QueryMinDepositRequest) (*v1.QueryMinDepositResponse, error) {
	result, err := q.QueryServer.MinDeposit(ctx, &sdkv1.QueryMinDepositRequest{})
	if err != nil {
		return nil, err
	}

	return &v1.QueryMinDepositResponse{MinDeposit: result.GetMinDeposit()}, nil
}

// MinInitialDeposit returns the minimum deposit required for a proposal to be submitted
func (q grpcServer) MinInitialDeposit(ctx context.Context, req *v1.QueryMinInitialDepositRequest) (*v1.QueryMinInitialDepositResponse, error) {
	result, err := q.QueryServer.MinInitialDeposit(ctx, &sdkv1.QueryMinInitialDepositRequest{})
	if err != nil {
		return nil, err
	}

	return &v1.QueryMinInitialDepositResponse{MinInitialDeposit: result.GetMinInitialDeposit()}, nil
}

// Quorums returns the current quorums
func (q grpcServer) Quorums(ctx context.Context, _ *v1.QueryQuorumsRequest) (*v1.QueryQuorumsResponse, error) {
	result, err := q.QueryServer.Quorums(ctx, &sdkv1.QueryQuorumsRequest{})
	if err != nil {
		return nil, err
	}

	return &v1.QueryQuorumsResponse{
		Quorum:                      result.GetQuorum(),
		ConstitutionAmendmentQuorum: result.GetConstitutionAmendmentQuorum(),
		LawQuorum:                   result.GetLawQuorum(),
	}, nil
}

// ParticipationEMAs queries the state of the proposal participation exponential moving averages.
func (q grpcServer) ParticipationEMAs(ctx context.Context, _ *v1.QueryParticipationEMAsRequest) (*v1.QueryParticipationEMAsResponse, error) {
	result, err := q.QueryServer.ParticipationEMAs(ctx, &sdkv1.QueryParticipationEMAsRequest{})
	if err != nil {
		return nil, err
	}

	return &v1.QueryParticipationEMAsResponse{
		ParticipationEma:                      result.GetParticipationEma(),
		ConstitutionAmendmentParticipationEma: result.GetConstitutionAmendmentParticipationEma(),
		LawParticipationEma:                   result.GetLawParticipationEma(),
	}, nil
}

var _ v1beta1.QueryServer = legacyQueryServer{}

type legacyQueryServer struct {
	QueryServer sdkv1beta1.QueryServer
}

// NewLegacyQueryServer returns an implementation of the v1beta1 legacy QueryServer interface.
func NewLegacyQueryServer(k *Keeper) v1beta1.QueryServer {
	return &legacyQueryServer{QueryServer: govkeeper.NewLegacyQueryServer(k.Keeper)}
}

func (q legacyQueryServer) Proposal(c context.Context, req *v1beta1.QueryProposalRequest) (*v1beta1.QueryProposalResponse, error) {
	result, err := q.QueryServer.Proposal(c, &sdkv1beta1.QueryProposalRequest{
		ProposalId: req.ProposalId,
	})
	if err != nil {
		return nil, err
	}

	proposal := result.GetProposal()
	converted := v1beta1.ConvertSDKProposalToAtomOne(&proposal)
	if converted == nil {
		return &v1beta1.QueryProposalResponse{}, nil
	}
	return &v1beta1.QueryProposalResponse{Proposal: *converted}, nil
}

func (q legacyQueryServer) Proposals(c context.Context, req *v1beta1.QueryProposalsRequest) (*v1beta1.QueryProposalsResponse, error) {
	result, err := q.QueryServer.Proposals(c, &sdkv1beta1.QueryProposalsRequest{
		ProposalStatus: sdkv1beta1.ProposalStatus(req.ProposalStatus),
		Voter:          req.Voter,
		Depositor:      req.Depositor,
		Pagination:     req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryProposalsResponse{
		Proposals:  v1beta1.ConvertSDKProposalsToAtomOne(result.GetProposals()),
		Pagination: result.GetPagination(),
	}, nil
}

func (q legacyQueryServer) Vote(c context.Context, req *v1beta1.QueryVoteRequest) (*v1beta1.QueryVoteResponse, error) {
	result, err := q.QueryServer.Vote(c, &sdkv1beta1.QueryVoteRequest{
		ProposalId: req.ProposalId,
		Voter:      req.Voter,
	})
	if err != nil {
		return nil, err
	}

	vote := result.GetVote()
	converted := v1beta1.ConvertSDKVoteToAtomOne(&vote)
	if converted == nil {
		return &v1beta1.QueryVoteResponse{}, nil
	}
	return &v1beta1.QueryVoteResponse{Vote: *converted}, nil
}

func (q legacyQueryServer) Votes(c context.Context, req *v1beta1.QueryVotesRequest) (*v1beta1.QueryVotesResponse, error) {
	result, err := q.QueryServer.Votes(c, &sdkv1beta1.QueryVotesRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryVotesResponse{
		Votes:      v1beta1.ConvertSDKVotesToAtomOne(result.GetVotes()),
		Pagination: result.GetPagination(),
	}, nil
}

func (q legacyQueryServer) Params(c context.Context, req *v1beta1.QueryParamsRequest) (*v1beta1.QueryParamsResponse, error) {
	result, err := q.QueryServer.Params(c, &sdkv1beta1.QueryParamsRequest{
		ParamsType: req.ParamsType,
	})
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryParamsResponse{
		VotingParams:  *v1beta1.ConvertSDKVotingParamsToAtomOne(&result.VotingParams),
		DepositParams: *v1beta1.ConvertSDKDepositParamsToAtomOne(&result.DepositParams),
		TallyParams:   *v1beta1.ConvertSDKTallyParamsToAtomOne(&result.TallyParams),
	}, nil
}

func (q legacyQueryServer) Deposit(c context.Context, req *v1beta1.QueryDepositRequest) (*v1beta1.QueryDepositResponse, error) {
	result, err := q.QueryServer.Deposit(c, &sdkv1beta1.QueryDepositRequest{
		ProposalId: req.ProposalId,
		Depositor:  req.Depositor,
	})
	if err != nil {
		return nil, err
	}

	deposit := result.GetDeposit()
	converted := v1beta1.ConvertSDKDepositToAtomOne(&deposit)
	if converted == nil {
		return &v1beta1.QueryDepositResponse{}, nil
	}
	return &v1beta1.QueryDepositResponse{Deposit: *converted}, nil
}

func (q legacyQueryServer) Deposits(c context.Context, req *v1beta1.QueryDepositsRequest) (*v1beta1.QueryDepositsResponse, error) {
	result, err := q.QueryServer.Deposits(c, &sdkv1beta1.QueryDepositsRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryDepositsResponse{
		Deposits:   v1beta1.ConvertSDKDepositsToAtomOne(result.GetDeposits()),
		Pagination: result.GetPagination(),
	}, nil
}

func (q legacyQueryServer) TallyResult(c context.Context, req *v1beta1.QueryTallyResultRequest) (*v1beta1.QueryTallyResultResponse, error) {
	result, err := q.QueryServer.TallyResult(c, &sdkv1beta1.QueryTallyResultRequest{
		ProposalId: req.ProposalId,
	})
	if err != nil {
		return nil, err
	}

	tally := result.GetTally()
	return &v1beta1.QueryTallyResultResponse{Tally: v1beta1.ConvertSDKTallyResultToAtomOne(&tally)}, nil
}
