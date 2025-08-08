package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// Tally iterates over the votes and updates the tally of a proposal based on the voting power of the
// voters
func (keeper Keeper) Tally(ctx sdk.Context, proposal v1.Proposal) (passes bool, burnDeposits bool, participation math.LegacyDec, tallyResults v1.TallyResult) {
	// fetch all the bonded validators
	currValidators := keeper.getBondedValidatorsByAddress(ctx)
	totalVotingPower, results := keeper.tallyVotes(ctx, proposal, currValidators, true)

	params := keeper.GetParams(ctx)
	tallyResults = v1.NewTallyResultFromMap(results)

	// If there is no staked coins, the proposal fails
	totalBonded := keeper.sk.TotalBondedTokens(ctx)
	if totalBonded.IsZero() {
		return false, false, sdk.ZeroDec(), tallyResults
	}

	// If there is not enough quorum of votes, the proposal fails
	percentVoting := totalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
	quorum, threshold := keeper.getQuorumAndThreshold(ctx, proposal)
	if percentVoting.LT(quorum) {
		return false, params.BurnVoteQuorum, percentVoting, tallyResults
	}

	// Compute non-abstaining voting power, aka active voting power
	activeVotingPower := totalVotingPower.Sub(results[v1.OptionAbstain])

	// If no one votes (everyone abstains), proposal fails
	if activeVotingPower.IsZero() {
		return false, false, percentVoting, tallyResults
	}

	// If more than `threshold` of non-abstaining voters vote Yes, proposal passes.
	yesPercent := results[v1.OptionYes].Quo(activeVotingPower)
	if yesPercent.GT(threshold) {
		return true, false, percentVoting, tallyResults
	}

	// If more than `burnDepositNoThreshold` of non-abstaining voters vote No,
	// proposal is rejected and deposit is burned.
	burnDepositNoThreshold := sdk.MustNewDecFromStr(params.BurnDepositNoThreshold)
	noPercent := results[v1.OptionNo].Quo(activeVotingPower)
	if noPercent.GT(burnDepositNoThreshold) {
		return false, true, percentVoting, tallyResults
	}

	// If less than `burnDepositNoThreshold` of non-abstaining voters vote No,
	// proposal is rejected but deposit is not burned.
	return false, false, percentVoting, tallyResults
}

// HasReachedQuorum returns whether or not a proposal has reached quorum
// this is just a stripped down version of the Tally function above
func (keeper Keeper) HasReachedQuorum(ctx sdk.Context, proposal v1.Proposal) bool {
	// If there is no staked coins, the proposal has not reached quorum
	totalBonded := keeper.sk.TotalBondedTokens(ctx)
	if totalBonded.IsZero() {
		return false
	}

	currValidators := keeper.getBondedValidatorsByAddress(ctx)
	totalVotingPower, _ := keeper.tallyVotes(ctx, proposal, currValidators, false)

	// check and return whether or not the proposal has reached quorum
	percentVoting := totalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
	quorum, _ := keeper.getQuorumAndThreshold(ctx, proposal)
	return percentVoting.GTE(quorum)
}

// getBondedValidatorsByAddress fetches all the bonded validators and return
// them in map using their operator address as the key.
func (keeper Keeper) getBondedValidatorsByAddress(ctx sdk.Context) map[string]stakingtypes.ValidatorI {
	vals := make(map[string]stakingtypes.ValidatorI)
	keeper.sk.IterateBondedValidatorsByPower(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		vals[validator.GetOperator().String()] = validator
		return false
	})
	return vals
}

// tallyVotes returns the total voting power and tally results of the votes
// on a proposal. If `isFinal` is true, results will be stored in `results`
// map and votes will be deleted. Otherwise, only the total voting power
// will be returned and `results` will be nil.
func (keeper Keeper) tallyVotes(
	ctx sdk.Context, proposal v1.Proposal,
	currValidators map[string]stakingtypes.ValidatorI,
	isFinal bool,
) (totalVotingPower math.LegacyDec, results map[v1.VoteOption]math.LegacyDec) {
	totalVotingPower = math.LegacyZeroDec()
	// keeps track of governors that voted or have delegators that voted
	allGovernors := make(map[string]v1.GovernorGovInfo)

	if isFinal {
		results = make(map[v1.VoteOption]math.LegacyDec)
		results[v1.OptionYes] = math.LegacyZeroDec()
		results[v1.OptionAbstain] = math.LegacyZeroDec()
		results[v1.OptionNo] = math.LegacyZeroDec()
	}

	keeper.IterateVotes(ctx, proposal.Id, func(vote v1.Vote) bool {
		var governor v1.GovernorGovInfo

		voter := sdk.MustAccAddressFromBech32(vote.Voter)

		gd, hasGovernor := keeper.GetGovernanceDelegation(ctx, voter)
		if hasGovernor {
			if gi, ok := allGovernors[gd.GovernorAddress]; ok {
				governor = gi
			} else {
				govAddr := types.MustGovernorAddressFromBech32(gd.GovernorAddress)
				governor = v1.NewGovernorGovInfo(
					govAddr,
					keeper.GetAllGovernorValShares(ctx, govAddr),
					v1.WeightedVoteOptions{},
				)
			}
			if gd.GovernorAddress == types.GovernorAddress(voter).String() {
				// voter and governor are the same account, record his vote
				governor.Vote = vote.Options
			}
			// Ensure allGovernors contains the updated governor
			allGovernors[gd.GovernorAddress] = governor
		}

		// iterate over all delegations from voter
		keeper.sk.IterateDelegations(ctx, voter, func(index int64, delegation stakingtypes.DelegationI) (stop bool) {
			valAddrStr := delegation.(stakingtypes.Delegation).ValidatorAddress
			votingPower := math.LegacyZeroDec()

			if val, ok := currValidators[valAddrStr]; ok {
				// delegation shares * bonded / total shares
				votingPower = votingPower.Add(delegation.GetShares().MulInt(val.GetBondedTokens()).Quo(val.GetDelegatorShares()))

				// remove the delegation shares from the governor
				if hasGovernor {
					governor.ValSharesDeductions[valAddrStr] = governor.ValSharesDeductions[valAddrStr].Add(delegation.GetShares())
				}
			}

			totalVotingPower = totalVotingPower.Add(votingPower)
			if isFinal {
				for _, option := range vote.Options {
					subPower := option.Power(votingPower)
					results[option.Option] = results[option.Option].Add(subPower)
				}
			}

			return false
		})

		if isFinal {
			keeper.deleteVote(ctx, vote.ProposalId, voter)
		}
		return false
	})

	// get only the voting governors that are active and have the niminum self-delegation requirement met.
	currGovernors := keeper.getCurrGovernors(ctx, allGovernors)

	// iterate over the governors again to tally their voting power
	// As active governor are simply voters that need to have 100% of their bonded tokens
	// delegated to them and their shares were deducted when iterating over votes
	// we don't need to handle special cases.
	for _, gov := range currGovernors {
		votingPower := getGovernorVotingPower(gov, currValidators)

		if isFinal {
			for _, option := range gov.Vote {
				subPower := option.Power(votingPower)
				results[option.Option] = results[option.Option].Add(subPower)
			}
		}
		totalVotingPower = totalVotingPower.Add(votingPower)
	}
	return totalVotingPower, results
}

// getQuorumAndThreshold returns the appropriate quorum and threshold according
// to proposal kind. If the proposal contains multiple kinds, the highest
// quorum and threshold is returned.
func (keeper Keeper) getQuorumAndThreshold(ctx sdk.Context, proposal v1.Proposal) (quorum sdk.Dec, threshold sdk.Dec) {
	params := keeper.GetParams(ctx)
	kinds := keeper.ProposalKinds(proposal)

	// start with the default quorum and threshold
	quorum = keeper.GetQuorum(ctx)
	threshold = sdk.MustNewDecFromStr(params.Threshold)

	// Check for Constitution Amendment and update if higher
	if kinds.HasKindConstitutionAmendment() {
		constitutionQuorum := keeper.GetConstitutionAmendmentQuorum(ctx)
		constitutionThreshold := sdk.MustNewDecFromStr(params.ConstitutionAmendmentThreshold)

		if constitutionQuorum.GT(quorum) {
			quorum = constitutionQuorum
		}
		if constitutionThreshold.GT(threshold) {
			threshold = constitutionThreshold
		}
	}

	// Check for Law and update if higher
	if kinds.HasKindLaw() {
		lawQuorum := keeper.GetLawQuorum(ctx)
		lawThreshold := sdk.MustNewDecFromStr(params.LawThreshold)

		if lawQuorum.GT(quorum) {
			quorum = lawQuorum
		}
		if lawThreshold.GT(threshold) {
			threshold = lawThreshold
		}
	}

	return quorum, threshold
}

// getCurrGovernors returns the governors that voted, are active and meet the minimum self-delegation requirement
func (k Keeper) getCurrGovernors(ctx sdk.Context, allGovernors map[string]v1.GovernorGovInfo) (governors []v1.GovernorGovInfo) {
	governorsInfos := make([]v1.GovernorGovInfo, 0)
	for _, govInfo := range allGovernors {
		governor, _ := k.GetGovernor(ctx, govInfo.Address)

		if k.ValidateGovernorMinSelfDelegation(ctx, governor) && len(govInfo.Vote) > 0 {
			governorsInfos = append(governorsInfos, govInfo)
		}
	}

	return governorsInfos
}

func getGovernorVotingPower(governor v1.GovernorGovInfo, currValidators map[string]stakingtypes.ValidatorI) (votingPower math.LegacyDec) {
	votingPower = math.LegacyZeroDec()
	for valAddrStr, shares := range governor.ValShares {
		if val, ok := currValidators[valAddrStr]; ok {
			sharesAfterDeductions := shares.Sub(governor.ValSharesDeductions[valAddrStr])
			votingPower = votingPower.Add(sharesAfterDeductions.MulInt(val.GetBondedTokens()).Quo(val.GetDelegatorShares()))
		}
	}
	return votingPower
}
