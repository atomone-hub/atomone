package keeper

import (
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// Request type mappings (atomone -> SDK)

func mapProposalRequest(req *v1.QueryProposalRequest) *sdkgovv1.QueryProposalRequest {
	if req == nil {
		return nil
	}
	return &sdkgovv1.QueryProposalRequest{
		ProposalId: req.ProposalId,
	}
}

func mapProposalsRequest(req *v1.QueryProposalsRequest) *sdkgovv1.QueryProposalsRequest {
	if req == nil {
		return nil
	}
	return &sdkgovv1.QueryProposalsRequest{
		ProposalStatus: sdkgovv1.ProposalStatus(req.ProposalStatus),
		Voter:          req.Voter,
		Depositor:      req.Depositor,
		Pagination:     req.Pagination,
	}
}

func mapVoteRequest(req *v1.QueryVoteRequest) *sdkgovv1.QueryVoteRequest {
	if req == nil {
		return nil
	}
	return &sdkgovv1.QueryVoteRequest{
		ProposalId: req.ProposalId,
		Voter:      req.Voter,
	}
}

func mapVotesRequest(req *v1.QueryVotesRequest) *sdkgovv1.QueryVotesRequest {
	if req == nil {
		return nil
	}
	return &sdkgovv1.QueryVotesRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	}
}

func mapParamsRequest(req *v1.QueryParamsRequest) *sdkgovv1.QueryParamsRequest {
	if req == nil {
		return nil
	}
	return &sdkgovv1.QueryParamsRequest{
		ParamsType: req.ParamsType,
	}
}

func mapDepositRequest(req *v1.QueryDepositRequest) *sdkgovv1.QueryDepositRequest {
	if req == nil {
		return nil
	}
	return &sdkgovv1.QueryDepositRequest{
		ProposalId: req.ProposalId,
		Depositor:  req.Depositor,
	}
}

func mapDepositsRequest(req *v1.QueryDepositsRequest) *sdkgovv1.QueryDepositsRequest {
	if req == nil {
		return nil
	}
	return &sdkgovv1.QueryDepositsRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	}
}

func mapTallyResultRequest(req *v1.QueryTallyResultRequest) *sdkgovv1.QueryTallyResultRequest {
	if req == nil {
		return nil
	}
	return &sdkgovv1.QueryTallyResultRequest{
		ProposalId: req.ProposalId,
	}
}

// Response type mappings (SDK -> atomone)

func mapProposalResponse(resp *sdkgovv1.QueryProposalResponse) *v1.QueryProposalResponse {
	if resp == nil {
		return nil
	}
	return &v1.QueryProposalResponse{
		Proposal: mapProposal(resp.Proposal),
	}
}

func mapProposalsResponse(resp *sdkgovv1.QueryProposalsResponse) *v1.QueryProposalsResponse {
	if resp == nil {
		return nil
	}
	proposals := make([]*v1.Proposal, len(resp.Proposals))
	for i, p := range resp.Proposals {
		proposals[i] = mapProposal(p)
	}
	return &v1.QueryProposalsResponse{
		Proposals:  proposals,
		Pagination: resp.Pagination,
	}
}

func mapQueryVoteResponse(resp *sdkgovv1.QueryVoteResponse) *v1.QueryVoteResponse {
	if resp == nil {
		return nil
	}
	return &v1.QueryVoteResponse{
		Vote: mapVote(resp.Vote),
	}
}

func mapVotesResponse(resp *sdkgovv1.QueryVotesResponse) *v1.QueryVotesResponse {
	if resp == nil {
		return nil
	}
	votes := make([]*v1.Vote, len(resp.Votes))
	for i, vote := range resp.Votes {
		votes[i] = mapVote(vote)
	}
	return &v1.QueryVotesResponse{
		Votes:      votes,
		Pagination: resp.Pagination,
	}
}

func mapQueryDepositResponse(resp *sdkgovv1.QueryDepositResponse) *v1.QueryDepositResponse {
	if resp == nil {
		return nil
	}
	return &v1.QueryDepositResponse{
		Deposit: mapDeposit(resp.Deposit),
	}
}

func mapDepositsResponse(resp *sdkgovv1.QueryDepositsResponse) *v1.QueryDepositsResponse {
	if resp == nil {
		return nil
	}
	deposits := make([]*v1.Deposit, len(resp.Deposits))
	for i, deposit := range resp.Deposits {
		deposits[i] = mapDeposit(deposit)
	}
	return &v1.QueryDepositsResponse{
		Deposits:   deposits,
		Pagination: resp.Pagination,
	}
}

func mapTallyResultResponse(resp *sdkgovv1.QueryTallyResultResponse) *v1.QueryTallyResultResponse {
	if resp == nil {
		return nil
	}
	return &v1.QueryTallyResultResponse{
		Tally: mapTallyResult(resp.Tally),
	}
}

func mapParamsResponse(resp *sdkgovv1.QueryParamsResponse) *v1.QueryParamsResponse {
	if resp == nil {
		return nil
	}
	return &v1.QueryParamsResponse{
		VotingParams:  mapVotingParams(resp.VotingParams),
		DepositParams: mapDepositParams(resp.DepositParams),
		TallyParams:   mapTallyParams(resp.TallyParams),
		Params:        mapParams(resp.Params),
	}
}

func mapVotingParams(params *sdkgovv1.VotingParams) *v1.VotingParams {
	if params == nil {
		return nil
	}
	return &v1.VotingParams{
		VotingPeriod: params.VotingPeriod,
	}
}

func mapDepositParams(params *sdkgovv1.DepositParams) *v1.DepositParams {
	if params == nil {
		return nil
	}
	return &v1.DepositParams{
		MinDeposit:       params.MinDeposit,
		MaxDepositPeriod: params.MaxDepositPeriod,
	}
}

func mapTallyParams(params *sdkgovv1.TallyParams) *v1.TallyParams {
	if params == nil {
		return nil
	}
	return &v1.TallyParams{
		Quorum:                         params.Quorum,
		Threshold:                      params.Threshold,
		ConstitutionAmendmentQuorum:    params.ConstitutionAmendmentQuorum,
		ConstitutionAmendmentThreshold: params.ConstitutionAmendmentThreshold,
		LawQuorum:                      params.LawQuorum,
		LawThreshold:                   params.LawThreshold,
	}
}

func mapParams(params *sdkgovv1.Params) *v1.Params {
	if params == nil {
		return nil
	}
	return &v1.Params{
		MinDeposit:                       params.MinDeposit,
		MaxDepositPeriod:                 params.MaxDepositPeriod,
		VotingPeriod:                     params.VotingPeriod,
		Quorum:                           params.Quorum,
		Threshold:                        params.Threshold,
		MinInitialDepositRatio:           params.MinInitialDepositRatio,
		BurnVoteQuorum:                   params.BurnVoteQuorum,
		BurnProposalDepositPrevote:       params.BurnProposalDepositPrevote,
		MinDepositRatio:                  params.MinDepositRatio,
		ConstitutionAmendmentQuorum:      params.ConstitutionAmendmentQuorum,
		ConstitutionAmendmentThreshold:   params.ConstitutionAmendmentThreshold,
		LawQuorum:                        params.LawQuorum,
		LawThreshold:                     params.LawThreshold,
		QuorumTimeout:                    params.QuorumTimeout,
		MaxVotingPeriodExtension:         params.MaxVotingPeriodExtension,
		QuorumCheckCount:                 params.QuorumCheckCount,
		MinDepositThrottler:              mapMinDepositThrottler(params.MinDepositThrottler),
		MinInitialDepositThrottler:       mapMinInitialDepositThrottler(params.MinInitialDepositThrottler),
		BurnDepositNoThreshold:           params.BurnDepositNoThreshold,
		QuorumRange:                      mapQuorumRange(params.QuorumRange),
		ConstitutionAmendmentQuorumRange: mapQuorumRange(params.ConstitutionAmendmentQuorumRange),
		LawQuorumRange:                   mapQuorumRange(params.LawQuorumRange),
	}
}

func mapMinDepositThrottler(throttler *sdkgovv1.MinDepositThrottler) *v1.MinDepositThrottler {
	if throttler == nil {
		return nil
	}
	return &v1.MinDepositThrottler{
		MinDeposit:          throttler.MinDeposit,
		UpdateTimeThreshold: throttler.UpdateTimeThreshold,
		Smoothing:           throttler.Smoothing,
		LowerThresholdCap:   throttler.LowerThresholdCap,
		UpperThresholdCap:   throttler.UpperThresholdCap,
	}
}

func mapMinInitialDepositThrottler(throttler *sdkgovv1.MinInitialDepositThrottler) *v1.MinInitialDepositThrottler {
	if throttler == nil {
		return nil
	}
	return &v1.MinInitialDepositThrottler{
		MinInitialRatio:   throttler.MinInitialRatio,
		DecreaseFactor:    throttler.DecreaseFactor,
		IncreaseFactor:    throttler.IncreaseFactor,
		TargetThreshold:   throttler.TargetThreshold,
		MinInitialDeposit: throttler.MinInitialDeposit,
	}
}

func mapQuorumRange(qRange *sdkgovv1.QuorumRange) *v1.QuorumRange {
	if qRange == nil {
		return nil
	}
	return &v1.QuorumRange{
		Min: qRange.Min,
		Max: qRange.Max,
	}
}

// Core type mappings

func mapProposal(p *sdkgovv1.Proposal) *v1.Proposal {
	if p == nil {
		return nil
	}
	return &v1.Proposal{
		Id:               p.Id,
		Messages:         p.Messages,
		Status:           v1.ProposalStatus(p.Status),
		FinalTallyResult: mapTallyResult(p.FinalTallyResult),
		SubmitTime:       p.SubmitTime,
		DepositEndTime:   p.DepositEndTime,
		TotalDeposit:     p.TotalDeposit,
		VotingStartTime:  p.VotingStartTime,
		VotingEndTime:    p.VotingEndTime,
		Metadata:         p.Metadata,
		Title:            p.Title,
		Summary:          p.Summary,
		Proposer:         p.Proposer,
	}
}

func mapVote(v *sdkgovv1.Vote) *v1.Vote {
	if v == nil {
		return nil
	}
	options := make([]*v1.WeightedVoteOption, len(v.Options))
	for i, opt := range v.Options {
		options[i] = &v1.WeightedVoteOption{
			Option: v1.VoteOption(opt.Option),
			Weight: opt.Weight,
		}
	}
	return &v1.Vote{
		ProposalId: v.ProposalId,
		Voter:      v.Voter,
		Options:    options,
		Metadata:   v.Metadata,
	}
}

func mapDeposit(d *sdkgovv1.Deposit) *v1.Deposit {
	if d == nil {
		return nil
	}
	return &v1.Deposit{
		ProposalId: d.ProposalId,
		Depositor:  d.Depositor,
		Amount:     d.Amount,
	}
}

func mapTallyResult(t *sdkgovv1.TallyResult) *v1.TallyResult {
	if t == nil {
		return nil
	}
	return &v1.TallyResult{
		YesCount:     t.YesCount,
		AbstainCount: t.AbstainCount,
		NoCount:      t.NoCount,
	}
}

// Message type mappings (atomone -> SDK)

func mapSubmitProposalMsg(msg *v1.MsgSubmitProposal) *sdkgovv1.MsgSubmitProposal {
	if msg == nil {
		return nil
	}
	return &sdkgovv1.MsgSubmitProposal{
		Messages:       msg.Messages,
		InitialDeposit: msg.InitialDeposit,
		Proposer:       msg.Proposer,
		Metadata:       msg.Metadata,
		Title:          msg.Title,
		Summary:        msg.Summary,
	}
}

func mapVoteMsg(msg *v1.MsgVote) *sdkgovv1.MsgVote {
	if msg == nil {
		return nil
	}
	return &sdkgovv1.MsgVote{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Option:     sdkgovv1.VoteOption(msg.Option),
		Metadata:   msg.Metadata,
	}
}

func mapVoteWeightedMsg(msg *v1.MsgVoteWeighted) *sdkgovv1.MsgVoteWeighted {
	if msg == nil {
		return nil
	}
	options := make([]*sdkgovv1.WeightedVoteOption, len(msg.Options))
	for i, opt := range msg.Options {
		options[i] = &sdkgovv1.WeightedVoteOption{
			Option: sdkgovv1.VoteOption(opt.Option),
			Weight: opt.Weight,
		}
	}
	return &sdkgovv1.MsgVoteWeighted{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Options:    options,
		Metadata:   msg.Metadata,
	}
}

func mapDepositMsg(msg *v1.MsgDeposit) *sdkgovv1.MsgDeposit {
	if msg == nil {
		return nil
	}
	return &sdkgovv1.MsgDeposit{
		ProposalId: msg.ProposalId,
		Depositor:  msg.Depositor,
		Amount:     msg.Amount,
	}
}

func mapUpdateParamsMsg(msg *v1.MsgUpdateParams) *sdkgovv1.MsgUpdateParams {
	if msg == nil {
		return nil
	}
	return &sdkgovv1.MsgUpdateParams{
		Authority: msg.Authority,
		Params: sdkgovv1.Params{
			MinDeposit:                 msg.Params.MinDeposit,
			MaxDepositPeriod:           msg.Params.MaxDepositPeriod,
			VotingPeriod:               msg.Params.VotingPeriod,
			Quorum:                     msg.Params.Quorum,
			Threshold:                  msg.Params.Threshold,
			MinInitialDepositRatio:     msg.Params.MinInitialDepositRatio,
			BurnVoteQuorum:             msg.Params.BurnVoteQuorum,
			BurnProposalDepositPrevote: msg.Params.BurnProposalDepositPrevote,
			MinDepositRatio:            msg.Params.MinDepositRatio,
		},
	}
}

// Response message mappings (SDK -> atomone)

func mapSubmitProposalResponse(resp *sdkgovv1.MsgSubmitProposalResponse) *v1.MsgSubmitProposalResponse {
	if resp == nil {
		return nil
	}
	return &v1.MsgSubmitProposalResponse{
		ProposalId: resp.ProposalId,
	}
}

func mapVoteResponse(resp *sdkgovv1.MsgVoteResponse) *v1.MsgVoteResponse {
	if resp == nil {
		return nil
	}
	return &v1.MsgVoteResponse{}
}

func mapVoteWeightedResponse(resp *sdkgovv1.MsgVoteWeightedResponse) *v1.MsgVoteWeightedResponse {
	if resp == nil {
		return nil
	}
	return &v1.MsgVoteWeightedResponse{}
}

func mapDepositResponse(resp *sdkgovv1.MsgDepositResponse) *v1.MsgDepositResponse {
	if resp == nil {
		return nil
	}
	return &v1.MsgDepositResponse{}
}

func mapUpdateParamsResponse(resp *sdkgovv1.MsgUpdateParamsResponse) *v1.MsgUpdateParamsResponse {
	if resp == nil {
		return nil
	}
	return &v1.MsgUpdateParamsResponse{}
}
