package v1beta1

import (
	sdkv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// ConvertSDKProposalToAtomOne converts a Cosmos SDK v1beta1 Proposal to an AtomOne v1beta1 Proposal
func ConvertSDKProposalToAtomOne(sdkProposal *sdkv1beta1.Proposal) *Proposal {
	if sdkProposal == nil {
		return nil
	}

	return &Proposal{
		ProposalId:       sdkProposal.ProposalId,
		Content:          sdkProposal.Content,
		Status:           ProposalStatus(sdkProposal.Status),
		FinalTallyResult: ConvertSDKTallyResultToAtomOne(&sdkProposal.FinalTallyResult),
		SubmitTime:       sdkProposal.SubmitTime,
		DepositEndTime:   sdkProposal.DepositEndTime,
		TotalDeposit:     sdkProposal.TotalDeposit,
		VotingStartTime:  sdkProposal.VotingStartTime,
		VotingEndTime:    sdkProposal.VotingEndTime,
	}
}

// ConvertSDKProposalsToAtomOne converts a slice of SDK proposals to AtomOne proposals
func ConvertSDKProposalsToAtomOne(sdkProposals []sdkv1beta1.Proposal) []Proposal {
	if sdkProposals == nil {
		return nil
	}

	proposals := make([]Proposal, len(sdkProposals))
	for i, p := range sdkProposals {
		converted := ConvertSDKProposalToAtomOne(&p)
		if converted != nil {
			proposals[i] = *converted
		}
	}
	return proposals
}

// ConvertSDKVoteToAtomOne converts a Cosmos SDK v1beta1 Vote to an AtomOne v1beta1 Vote
func ConvertSDKVoteToAtomOne(sdkVote *sdkv1beta1.Vote) *Vote {
	if sdkVote == nil {
		return nil
	}

	return &Vote{
		ProposalId: sdkVote.ProposalId,
		Voter:      sdkVote.Voter,
		Option:     VoteOption(sdkVote.Option),
		Options:    ConvertSDKWeightedVoteOptionsToAtomOne(sdkVote.Options),
	}
}

// ConvertSDKVotesToAtomOne converts a slice of SDK votes to AtomOne votes
func ConvertSDKVotesToAtomOne(sdkVotes []sdkv1beta1.Vote) []Vote {
	if sdkVotes == nil {
		return nil
	}

	votes := make([]Vote, len(sdkVotes))
	for i, v := range sdkVotes {
		converted := ConvertSDKVoteToAtomOne(&v)
		if converted != nil {
			votes[i] = *converted
		}
	}
	return votes
}

// ConvertSDKWeightedVoteOptionsToAtomOne converts SDK weighted vote options to AtomOne
func ConvertSDKWeightedVoteOptionsToAtomOne(sdkOptions []sdkv1beta1.WeightedVoteOption) []WeightedVoteOption {
	if sdkOptions == nil {
		return nil
	}

	options := make([]WeightedVoteOption, len(sdkOptions))
	for i, opt := range sdkOptions {
		options[i] = WeightedVoteOption{
			Option: VoteOption(opt.Option),
			Weight: opt.Weight,
		}
	}
	return options
}

// ConvertSDKDepositToAtomOne converts a Cosmos SDK v1beta1 Deposit to an AtomOne v1beta1 Deposit
func ConvertSDKDepositToAtomOne(sdkDeposit *sdkv1beta1.Deposit) *Deposit {
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
func ConvertSDKDepositsToAtomOne(sdkDeposits []sdkv1beta1.Deposit) []Deposit {
	if sdkDeposits == nil {
		return nil
	}

	deposits := make([]Deposit, len(sdkDeposits))
	for i, d := range sdkDeposits {
		converted := ConvertSDKDepositToAtomOne(&d)
		if converted != nil {
			deposits[i] = *converted
		}
	}
	return deposits
}

// ConvertSDKTallyResultToAtomOne converts a Cosmos SDK v1beta1 TallyResult to an AtomOne v1beta1 TallyResult
func ConvertSDKTallyResultToAtomOne(sdkTally *sdkv1beta1.TallyResult) TallyResult {
	if sdkTally == nil {
		return TallyResult{}
	}

	return TallyResult{
		Yes:        sdkTally.Yes,
		Abstain:    sdkTally.Abstain,
		No:         sdkTally.No,
		NoWithVeto: sdkTally.NoWithVeto,
	}
}

// ConvertSDKVotingParamsToAtomOne converts SDK voting params to AtomOne voting params
func ConvertSDKVotingParamsToAtomOne(sdkParams *sdkv1beta1.VotingParams) *VotingParams {
	if sdkParams == nil {
		return nil
	}

	return &VotingParams{
		VotingPeriod: sdkParams.VotingPeriod,
	}
}

// ConvertSDKDepositParamsToAtomOne converts SDK deposit params to AtomOne deposit params
func ConvertSDKDepositParamsToAtomOne(sdkParams *sdkv1beta1.DepositParams) *DepositParams {
	if sdkParams == nil {
		return nil
	}

	return &DepositParams{
		MinDeposit:       sdkParams.MinDeposit,
		MaxDepositPeriod: sdkParams.MaxDepositPeriod,
	}
}

// ConvertSDKTallyParamsToAtomOne converts SDK tally params to AtomOne tally params
func ConvertSDKTallyParamsToAtomOne(sdkParams *sdkv1beta1.TallyParams) *TallyParams {
	if sdkParams == nil {
		return nil
	}

	return &TallyParams{
		Quorum:        sdkParams.Quorum,
		Threshold:     sdkParams.Threshold,
		VetoThreshold: sdkParams.VetoThreshold,
	}
}

// ConvertAtomOneProposalToSDK converts an AtomOne v1beta1 Proposal to a Cosmos SDK v1beta1 Proposal
func ConvertAtomOneProposalToSDK(atomoneProposal *Proposal) *sdkv1beta1.Proposal {
	if atomoneProposal == nil {
		return nil
	}

	return &sdkv1beta1.Proposal{
		ProposalId:       atomoneProposal.ProposalId,
		Content:          atomoneProposal.Content,
		Status:           sdkv1beta1.ProposalStatus(atomoneProposal.Status),
		FinalTallyResult: ConvertAtomOneTallyResultToSDK(atomoneProposal.FinalTallyResult),
		SubmitTime:       atomoneProposal.SubmitTime,
		DepositEndTime:   atomoneProposal.DepositEndTime,
		TotalDeposit:     atomoneProposal.TotalDeposit,
		VotingStartTime:  atomoneProposal.VotingStartTime,
		VotingEndTime:    atomoneProposal.VotingEndTime,
	}
}

// ConvertAtomOneProposalsToSDK converts a slice of AtomOne proposals to SDK proposals
func ConvertAtomOneProposalsToSDK(atomoneProposals []Proposal) []sdkv1beta1.Proposal {
	if atomoneProposals == nil {
		return nil
	}

	proposals := make([]sdkv1beta1.Proposal, len(atomoneProposals))
	for i, p := range atomoneProposals {
		converted := ConvertAtomOneProposalToSDK(&p)
		if converted != nil {
			proposals[i] = *converted
		}
	}
	return proposals
}

// ConvertAtomOneTallyResultToSDK converts an AtomOne v1beta1 TallyResult to a Cosmos SDK v1beta1 TallyResult
func ConvertAtomOneTallyResultToSDK(atomoneTally TallyResult) sdkv1beta1.TallyResult {
	return sdkv1beta1.TallyResult{
		Yes:        atomoneTally.Yes,
		Abstain:    atomoneTally.Abstain,
		No:         atomoneTally.No,
		NoWithVeto: atomoneTally.NoWithVeto,
	}
}

// ConvertAtomOneVoteToSDK converts an AtomOne v1beta1 Vote to a Cosmos SDK v1beta1 Vote
func ConvertAtomOneVoteToSDK(atomoneVote *Vote) *sdkv1beta1.Vote {
	if atomoneVote == nil {
		return nil
	}

	return &sdkv1beta1.Vote{
		ProposalId: atomoneVote.ProposalId,
		Voter:      atomoneVote.Voter,
		Option:     sdkv1beta1.VoteOption(atomoneVote.Option),
		Options:    ConvertAtomOneWeightedVoteOptionsToSDK(atomoneVote.Options),
	}
}

// ConvertAtomOneVotesToSDK converts a slice of AtomOne votes to SDK votes
func ConvertAtomOneVotesToSDK(atomoneVotes []Vote) []sdkv1beta1.Vote {
	if atomoneVotes == nil {
		return nil
	}

	votes := make([]sdkv1beta1.Vote, len(atomoneVotes))
	for i, v := range atomoneVotes {
		converted := ConvertAtomOneVoteToSDK(&v)
		if converted != nil {
			votes[i] = *converted
		}
	}
	return votes
}

// ConvertAtomOneWeightedVoteOptionsToSDK converts AtomOne weighted vote options to SDK
func ConvertAtomOneWeightedVoteOptionsToSDK(atomoneOptions []WeightedVoteOption) []sdkv1beta1.WeightedVoteOption {
	if atomoneOptions == nil {
		return nil
	}

	options := make([]sdkv1beta1.WeightedVoteOption, len(atomoneOptions))
	for i, opt := range atomoneOptions {
		options[i] = sdkv1beta1.WeightedVoteOption{
			Option: sdkv1beta1.VoteOption(opt.Option),
			Weight: opt.Weight,
		}
	}
	return options
}

// ConvertAtomOneDepositToSDK converts an AtomOne v1beta1 Deposit to a Cosmos SDK v1beta1 Deposit
func ConvertAtomOneDepositToSDK(atomoneDeposit *Deposit) *sdkv1beta1.Deposit {
	if atomoneDeposit == nil {
		return nil
	}

	return &sdkv1beta1.Deposit{
		ProposalId: atomoneDeposit.ProposalId,
		Depositor:  atomoneDeposit.Depositor,
		Amount:     atomoneDeposit.Amount,
	}
}

// ConvertAtomOneDepositsToSDK converts a slice of AtomOne deposits to SDK deposits
func ConvertAtomOneDepositsToSDK(atomoneDeposits []Deposit) []sdkv1beta1.Deposit {
	if atomoneDeposits == nil {
		return nil
	}

	deposits := make([]sdkv1beta1.Deposit, len(atomoneDeposits))
	for i, d := range atomoneDeposits {
		converted := ConvertAtomOneDepositToSDK(&d)
		if converted != nil {
			deposits[i] = *converted
		}
	}
	return deposits
}

// ConvertAtomOneVotingParamsToSDK converts AtomOne voting params to SDK voting params
func ConvertAtomOneVotingParamsToSDK(atomoneParams *VotingParams) *sdkv1beta1.VotingParams {
	if atomoneParams == nil {
		return nil
	}

	return &sdkv1beta1.VotingParams{
		VotingPeriod: atomoneParams.VotingPeriod,
	}
}

// ConvertAtomOneDepositParamsToSDK converts AtomOne deposit params to SDK deposit params
func ConvertAtomOneDepositParamsToSDK(atomoneParams *DepositParams) *sdkv1beta1.DepositParams {
	if atomoneParams == nil {
		return nil
	}

	return &sdkv1beta1.DepositParams{
		MinDeposit:       atomoneParams.MinDeposit,
		MaxDepositPeriod: atomoneParams.MaxDepositPeriod,
	}
}

// ConvertAtomOneTallyParamsToSDK converts AtomOne tally params to SDK tally params
func ConvertAtomOneTallyParamsToSDK(atomoneParams *TallyParams) *sdkv1beta1.TallyParams {
	if atomoneParams == nil {
		return nil
	}

	return &sdkv1beta1.TallyParams{
		Quorum:        atomoneParams.Quorum,
		Threshold:     atomoneParams.Threshold,
		VetoThreshold: atomoneParams.VetoThreshold,
	}
}
