package v1

import (
	sdkv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// ConvertSDKProposalToAtomOne converts a Cosmos SDK v1 Proposal to an AtomOne v1 Proposal
func ConvertSDKProposalToAtomOne(sdkProposal *sdkv1.Proposal) *Proposal {
	if sdkProposal == nil {
		return nil
	}

	return &Proposal{
		Id:                        sdkProposal.Id,
		Messages:                  sdkProposal.Messages,
		Status:                    ProposalStatus(sdkProposal.Status),
		FinalTallyResult:          ConvertSDKTallyResultToAtomOne(sdkProposal.FinalTallyResult),
		SubmitTime:                sdkProposal.SubmitTime,
		DepositEndTime:            sdkProposal.DepositEndTime,
		TotalDeposit:              sdkProposal.TotalDeposit,
		VotingStartTime:           sdkProposal.VotingStartTime,
		VotingEndTime:             sdkProposal.VotingEndTime,
		Metadata:                  sdkProposal.Metadata,
		Title:                     sdkProposal.Title,
		Summary:                   sdkProposal.Summary,
		Proposer:                  sdkProposal.Proposer,
		Endorsed:                  false,
		Annotation:                "",
		TimesVotingPeriodExtended: 0,
	}
}

// ConvertSDKProposalsToAtomOne converts a slice of SDK proposals to AtomOne proposals
func ConvertSDKProposalsToAtomOne(sdkProposals []*sdkv1.Proposal) []*Proposal {
	if sdkProposals == nil {
		return nil
	}

	proposals := make([]*Proposal, len(sdkProposals))
	for i, p := range sdkProposals {
		proposals[i] = ConvertSDKProposalToAtomOne(p)
	}
	return proposals
}

// ConvertSDKVoteToAtomOne converts a Cosmos SDK v1 Vote to an AtomOne v1 Vote
func ConvertSDKVoteToAtomOne(sdkVote *sdkv1.Vote) *Vote {
	if sdkVote == nil {
		return nil
	}

	return &Vote{
		ProposalId: sdkVote.ProposalId,
		Voter:      sdkVote.Voter,
		Options:    ConvertSDKWeightedVoteOptionsToAtomOne(sdkVote.Options),
		Metadata:   sdkVote.Metadata,
	}
}

// ConvertSDKVotesToAtomOne converts a slice of SDK votes to AtomOne votes
func ConvertSDKVotesToAtomOne(sdkVotes []*sdkv1.Vote) []*Vote {
	if sdkVotes == nil {
		return nil
	}

	votes := make([]*Vote, len(sdkVotes))
	for i, v := range sdkVotes {
		votes[i] = ConvertSDKVoteToAtomOne(v)
	}
	return votes
}

// ConvertSDKWeightedVoteOptionsToAtomOne converts SDK weighted vote options to AtomOne
func ConvertSDKWeightedVoteOptionsToAtomOne(sdkOptions []*sdkv1.WeightedVoteOption) []*WeightedVoteOption {
	if sdkOptions == nil {
		return nil
	}

	options := make([]*WeightedVoteOption, len(sdkOptions))
	for i, opt := range sdkOptions {
		options[i] = &WeightedVoteOption{
			Option: VoteOption(opt.Option),
			Weight: opt.Weight,
		}
	}
	return options
}

// ConvertSDKDepositToAtomOne converts a Cosmos SDK v1 Deposit to an AtomOne v1 Deposit
func ConvertSDKDepositToAtomOne(sdkDeposit *sdkv1.Deposit) *Deposit {
	if sdkDeposit == nil {
		return nil
	}

	return &Deposit{
		ProposalId: sdkDeposit.ProposalId,
		Depositor:  sdkDeposit.Depositor,
		Amount:     sdkDeposit.Amount,
	}
}

// ConvertSDKDepositsToAtomOne converts a slice of SDK deposits to AtomOne deposits
func ConvertSDKDepositsToAtomOne(sdkDeposits []*sdkv1.Deposit) []*Deposit {
	if sdkDeposits == nil {
		return nil
	}

	deposits := make([]*Deposit, len(sdkDeposits))
	for i, d := range sdkDeposits {
		deposits[i] = ConvertSDKDepositToAtomOne(d)
	}
	return deposits
}

// ConvertSDKTallyResultToAtomOne converts a Cosmos SDK v1 TallyResult to an AtomOne v1 TallyResult
func ConvertSDKTallyResultToAtomOne(sdkTally *sdkv1.TallyResult) *TallyResult {
	if sdkTally == nil {
		return nil
	}

	return &TallyResult{
		YesCount:     sdkTally.YesCount,
		AbstainCount: sdkTally.AbstainCount,
		NoCount:      sdkTally.NoCount,
	}
}

// ConvertSDKParamsToAtomOne converts SDK params to AtomOne params
func ConvertSDKParamsToAtomOne(sdkParams *sdkv1.Params) *Params {
	if sdkParams == nil {
		return nil
	}

	return &Params{
		MinDeposit:                       sdkParams.MinDeposit,
		MaxDepositPeriod:                 sdkParams.MaxDepositPeriod,
		VotingPeriod:                     sdkParams.VotingPeriod,
		Quorum:                           sdkParams.Quorum,
		Threshold:                        sdkParams.Threshold,
		MinInitialDepositRatio:           sdkParams.MinInitialDepositRatio,
		BurnVoteQuorum:                   sdkParams.BurnVoteQuorum,
		BurnProposalDepositPrevote:       sdkParams.BurnProposalDepositPrevote,
		MinDepositRatio:                  sdkParams.MinDepositRatio,
		ConstitutionAmendmentQuorum:      sdkParams.ConstitutionAmendmentQuorum,
		ConstitutionAmendmentThreshold:   sdkParams.ConstitutionAmendmentThreshold,
		LawQuorum:                        sdkParams.LawQuorum,
		LawThreshold:                     sdkParams.LawThreshold,
		QuorumTimeout:                    sdkParams.QuorumTimeout,
		MaxVotingPeriodExtension:         sdkParams.MaxVotingPeriodExtension,
		QuorumCheckCount:                 sdkParams.QuorumCheckCount,
		MinDepositThrottler:              ConvertSDKMinDepositThrottlerToAtomOne(sdkParams.MinDepositThrottler),
		MinInitialDepositThrottler:       ConvertSDKMinInitialDepositThrottlerToAtomOne(sdkParams.MinInitialDepositThrottler),
		BurnDepositNoThreshold:           sdkParams.BurnDepositNoThreshold,
		QuorumRange:                      ConvertSDKQuorumRangeToAtomOne(sdkParams.QuorumRange),
		ConstitutionAmendmentQuorumRange: ConvertSDKQuorumRangeToAtomOne(sdkParams.ConstitutionAmendmentQuorumRange),
		LawQuorumRange:                   ConvertSDKQuorumRangeToAtomOne(sdkParams.LawQuorumRange),
	}
}

// ConvertSDKMinDepositThrottlerToAtomOne converts SDK MinDepositThrottler to AtomOne
func ConvertSDKMinDepositThrottlerToAtomOne(sdkThrottler *sdkv1.MinDepositThrottler) *MinDepositThrottler {
	if sdkThrottler == nil {
		return nil
	}

	return &MinDepositThrottler{
		FloorValue:            sdkThrottler.FloorValue,
		UpdatePeriod:          sdkThrottler.UpdatePeriod,
		TargetActiveProposals: sdkThrottler.TargetActiveProposals,
		IncreaseRatio:         sdkThrottler.IncreaseRatio,
		DecreaseRatio:         sdkThrottler.DecreaseRatio,
	}
}

// ConvertSDKMinInitialDepositThrottlerToAtomOne converts SDK MinInitialDepositThrottler to AtomOne
func ConvertSDKMinInitialDepositThrottlerToAtomOne(sdkThrottler *sdkv1.MinInitialDepositThrottler) *MinInitialDepositThrottler {
	if sdkThrottler == nil {
		return nil
	}

	return &MinInitialDepositThrottler{
		FloorValue:      sdkThrottler.FloorValue,
		UpdatePeriod:    sdkThrottler.UpdatePeriod,
		TargetProposals: sdkThrottler.TargetProposals,
		IncreaseRatio:   sdkThrottler.IncreaseRatio,
		DecreaseRatio:   sdkThrottler.DecreaseRatio,
	}
}

// ConvertSDKQuorumRangeToAtomOne converts SDK QuorumRange to AtomOne
func ConvertSDKQuorumRangeToAtomOne(sdkRange *sdkv1.QuorumRange) *QuorumRange {
	if sdkRange == nil {
		return nil
	}

	return &QuorumRange{
		Max: sdkRange.Max,
		Min: sdkRange.Min,
	}
}

// ConvertSDKVotingParamsToAtomOne converts SDK voting params to AtomOne voting params
func ConvertSDKVotingParamsToAtomOne(sdkParams *sdkv1.VotingParams) *VotingParams {
	if sdkParams == nil {
		return nil
	}

	return &VotingParams{
		VotingPeriod: sdkParams.VotingPeriod,
	}
}

// ConvertSDKDepositParamsToAtomOne converts SDK deposit params to AtomOne deposit params
func ConvertSDKDepositParamsToAtomOne(sdkParams *sdkv1.DepositParams) *DepositParams {
	if sdkParams == nil {
		return nil
	}

	return &DepositParams{
		MinDeposit:       sdkParams.MinDeposit,
		MaxDepositPeriod: sdkParams.MaxDepositPeriod,
	}
}

// ConvertSDKTallyParamsToAtomOne converts SDK tally params to AtomOne tally params
func ConvertSDKTallyParamsToAtomOne(sdkParams *sdkv1.TallyParams) *TallyParams {
	if sdkParams == nil {
		return nil
	}

	return &TallyParams{
		Quorum:                         sdkParams.Quorum,
		Threshold:                      sdkParams.Threshold,
		ConstitutionAmendmentQuorum:    sdkParams.ConstitutionAmendmentQuorum,
		ConstitutionAmendmentThreshold: sdkParams.ConstitutionAmendmentThreshold,
		LawQuorum:                      sdkParams.LawQuorum,
		LawThreshold:                   sdkParams.LawThreshold,
	}
}

// ConvertAtomOneProposalToSDK converts an AtomOne v1 Proposal to a Cosmos SDK v1 Proposal
func ConvertAtomOneProposalToSDK(atomoneProposal *Proposal) *sdkv1.Proposal {
	if atomoneProposal == nil {
		return nil
	}

	return &sdkv1.Proposal{
		Id:               atomoneProposal.Id,
		Messages:         atomoneProposal.Messages,
		Status:           sdkv1.ProposalStatus(atomoneProposal.Status),
		FinalTallyResult: ConvertAtomOneTallyResultToSDK(atomoneProposal.FinalTallyResult),
		SubmitTime:       atomoneProposal.SubmitTime,
		DepositEndTime:   atomoneProposal.DepositEndTime,
		TotalDeposit:     atomoneProposal.TotalDeposit,
		VotingStartTime:  atomoneProposal.VotingStartTime,
		VotingEndTime:    atomoneProposal.VotingEndTime,
		Metadata:         atomoneProposal.Metadata,
		Title:            atomoneProposal.Title,
		Summary:          atomoneProposal.Summary,
		Proposer:         atomoneProposal.Proposer,
	}
}

// ConvertAtomOneTallyResultToSDK converts an AtomOne v1 TallyResult to a Cosmos SDK v1 TallyResult
func ConvertAtomOneTallyResultToSDK(atomoneTally *TallyResult) *sdkv1.TallyResult {
	if atomoneTally == nil {
		return nil
	}

	return &sdkv1.TallyResult{
		YesCount:     atomoneTally.YesCount,
		AbstainCount: atomoneTally.AbstainCount,
		NoCount:      atomoneTally.NoCount,
	}
}

// ConvertAtomOneVoteToSDK converts an AtomOne v1 Vote to a Cosmos SDK v1 Vote
func ConvertAtomOneVoteToSDK(atomoneVote *Vote) *sdkv1.Vote {
	if atomoneVote == nil {
		return nil
	}

	return &sdkv1.Vote{
		ProposalId: atomoneVote.ProposalId,
		Voter:      atomoneVote.Voter,
		Options:    ConvertAtomOneWeightedVoteOptionsToSDK(atomoneVote.Options),
		Metadata:   atomoneVote.Metadata,
	}
}

// ConvertAtomOneWeightedVoteOptionsToSDK converts AtomOne weighted vote options to SDK
func ConvertAtomOneWeightedVoteOptionsToSDK(atomoneOptions []*WeightedVoteOption) []*sdkv1.WeightedVoteOption {
	if atomoneOptions == nil {
		return nil
	}

	options := make([]*sdkv1.WeightedVoteOption, len(atomoneOptions))
	for i, opt := range atomoneOptions {
		options[i] = &sdkv1.WeightedVoteOption{
			Option: sdkv1.VoteOption(opt.Option),
			Weight: opt.Weight,
		}
	}
	return options
}

// ConvertAtomOneDepositToSDK converts an AtomOne v1 Deposit to a Cosmos SDK v1 Deposit
func ConvertAtomOneDepositToSDK(atomoneDeposit *Deposit) *sdkv1.Deposit {
	if atomoneDeposit == nil {
		return nil
	}

	return &sdkv1.Deposit{
		ProposalId: atomoneDeposit.ProposalId,
		Depositor:  atomoneDeposit.Depositor,
		Amount:     atomoneDeposit.Amount,
	}
}

// ConvertAtomOneParamsToSDK converts AtomOne params to SDK params
func ConvertAtomOneParamsToSDK(atomoneParams *Params) *sdkv1.Params {
	if atomoneParams == nil {
		return nil
	}

	return &sdkv1.Params{
		MinDeposit:                       atomoneParams.MinDeposit,
		MaxDepositPeriod:                 atomoneParams.MaxDepositPeriod,
		VotingPeriod:                     atomoneParams.VotingPeriod,
		Quorum:                           atomoneParams.Quorum,
		Threshold:                        atomoneParams.Threshold,
		MinInitialDepositRatio:           atomoneParams.MinInitialDepositRatio,
		BurnVoteQuorum:                   atomoneParams.BurnVoteQuorum,
		BurnProposalDepositPrevote:       atomoneParams.BurnProposalDepositPrevote,
		MinDepositRatio:                  atomoneParams.MinDepositRatio,
		ConstitutionAmendmentQuorum:      atomoneParams.ConstitutionAmendmentQuorum,
		ConstitutionAmendmentThreshold:   atomoneParams.ConstitutionAmendmentThreshold,
		LawQuorum:                        atomoneParams.LawQuorum,
		LawThreshold:                     atomoneParams.LawThreshold,
		QuorumTimeout:                    atomoneParams.QuorumTimeout,
		MaxVotingPeriodExtension:         atomoneParams.MaxVotingPeriodExtension,
		QuorumCheckCount:                 atomoneParams.QuorumCheckCount,
		MinDepositThrottler:              ConvertAtomOneMinDepositThrottlerToSDK(atomoneParams.MinDepositThrottler),
		MinInitialDepositThrottler:       ConvertAtomOneMinInitialDepositThrottlerToSDK(atomoneParams.MinInitialDepositThrottler),
		BurnDepositNoThreshold:           atomoneParams.BurnDepositNoThreshold,
		QuorumRange:                      ConvertAtomOneQuorumRangeToSDK(atomoneParams.QuorumRange),
		ConstitutionAmendmentQuorumRange: ConvertAtomOneQuorumRangeToSDK(atomoneParams.ConstitutionAmendmentQuorumRange),
		LawQuorumRange:                   ConvertAtomOneQuorumRangeToSDK(atomoneParams.LawQuorumRange),
	}
}

// ConvertAtomOneMinDepositThrottlerToSDK converts AtomOne MinDepositThrottler to SDK
func ConvertAtomOneMinDepositThrottlerToSDK(atomoneThrottler *MinDepositThrottler) *sdkv1.MinDepositThrottler {
	if atomoneThrottler == nil {
		return nil
	}

	return &sdkv1.MinDepositThrottler{
		FloorValue:            atomoneThrottler.FloorValue,
		UpdatePeriod:          atomoneThrottler.UpdatePeriod,
		TargetActiveProposals: atomoneThrottler.TargetActiveProposals,
		IncreaseRatio:         atomoneThrottler.IncreaseRatio,
		DecreaseRatio:         atomoneThrottler.DecreaseRatio,
	}
}

// ConvertAtomOneMinInitialDepositThrottlerToSDK converts AtomOne MinInitialDepositThrottler to SDK
func ConvertAtomOneMinInitialDepositThrottlerToSDK(atomoneThrottler *MinInitialDepositThrottler) *sdkv1.MinInitialDepositThrottler {
	if atomoneThrottler == nil {
		return nil
	}

	return &sdkv1.MinInitialDepositThrottler{
		FloorValue:      atomoneThrottler.FloorValue,
		UpdatePeriod:    atomoneThrottler.UpdatePeriod,
		TargetProposals: atomoneThrottler.TargetProposals,
		IncreaseRatio:   atomoneThrottler.IncreaseRatio,
		DecreaseRatio:   atomoneThrottler.DecreaseRatio,
	}
}

// ConvertAtomOneQuorumRangeToSDK converts AtomOne QuorumRange to SDK
func ConvertAtomOneQuorumRangeToSDK(atomoneRange *QuorumRange) *sdkv1.QuorumRange {
	if atomoneRange == nil {
		return nil
	}

	return &sdkv1.QuorumRange{
		Max: atomoneRange.Max,
		Min: atomoneRange.Min,
	}
}

// ConvertAtomOneProposalsToSDK converts a slice of AtomOne proposals to SDK proposals
func ConvertAtomOneProposalsToSDK(atomoneProposals []*Proposal) []*sdkv1.Proposal {
	if atomoneProposals == nil {
		return nil
	}

	proposals := make([]*sdkv1.Proposal, len(atomoneProposals))
	for i, p := range atomoneProposals {
		proposals[i] = ConvertAtomOneProposalToSDK(p)
	}
	return proposals
}

// ConvertAtomOneVotesToSDK converts a slice of AtomOne votes to SDK votes
func ConvertAtomOneVotesToSDK(atomoneVotes []*Vote) []*sdkv1.Vote {
	if atomoneVotes == nil {
		return nil
	}

	votes := make([]*sdkv1.Vote, len(atomoneVotes))
	for i, v := range atomoneVotes {
		votes[i] = ConvertAtomOneVoteToSDK(v)
	}
	return votes
}

// ConvertAtomOneDepositsToSDK converts a slice of AtomOne deposits to SDK deposits
func ConvertAtomOneDepositsToSDK(atomoneDeposits []*Deposit) []*sdkv1.Deposit {
	if atomoneDeposits == nil {
		return nil
	}

	deposits := make([]*sdkv1.Deposit, len(atomoneDeposits))
	for i, d := range atomoneDeposits {
		deposits[i] = ConvertAtomOneDepositToSDK(d)
	}
	return deposits
}

// ConvertAtomOneVotingParamsToSDK converts AtomOne voting params to SDK voting params
func ConvertAtomOneVotingParamsToSDK(atomoneParams *VotingParams) *sdkv1.VotingParams {
	if atomoneParams == nil {
		return nil
	}

	return &sdkv1.VotingParams{
		VotingPeriod: atomoneParams.VotingPeriod,
	}
}

// ConvertAtomOneDepositParamsToSDK converts AtomOne deposit params to SDK deposit params
func ConvertAtomOneDepositParamsToSDK(atomoneParams *DepositParams) *sdkv1.DepositParams {
	if atomoneParams == nil {
		return nil
	}

	return &sdkv1.DepositParams{
		MinDeposit:       atomoneParams.MinDeposit,
		MaxDepositPeriod: atomoneParams.MaxDepositPeriod,
	}
}

// ConvertAtomOneTallyParamsToSDK converts AtomOne tally params to SDK tally params
func ConvertAtomOneTallyParamsToSDK(atomoneParams *TallyParams) *sdkv1.TallyParams {
	if atomoneParams == nil {
		return nil
	}

	return &sdkv1.TallyParams{
		Quorum:                         atomoneParams.Quorum,
		Threshold:                      atomoneParams.Threshold,
		ConstitutionAmendmentQuorum:    atomoneParams.ConstitutionAmendmentQuorum,
		ConstitutionAmendmentThreshold: atomoneParams.ConstitutionAmendmentThreshold,
		LawQuorum:                      atomoneParams.LawQuorum,
		LawThreshold:                   atomoneParams.LawThreshold,
	}
}
