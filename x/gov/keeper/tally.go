package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// TODO: Break into several smaller functions for clarity

// Tally iterates over the votes and updates the tally of a proposal based on the voting power of the
// voters
func (keeper Keeper) Tally(ctx sdk.Context, proposal v1.Proposal) (passes bool, burnDeposits bool, tallyResults v1.TallyResult) {
	results := make(map[v1.VoteOption]sdk.Dec)
	results[v1.OptionYes] = math.LegacyZeroDec()
	results[v1.OptionAbstain] = math.LegacyZeroDec()
	results[v1.OptionNo] = math.LegacyZeroDec()
	results[v1.OptionNoWithVeto] = math.LegacyZeroDec()

	totalVotingPower := math.LegacyZeroDec()
	currValidators := make(map[string]stakingtypes.ValidatorI)
	currGovernors := make(map[string]v1.GovernorGovInfo)

	// fetch all the bonded validators, insert them into currValidators
	keeper.sk.IterateBondedValidatorsByPower(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		currValidators[validator.GetOperator().String()] = validator
		return false
	})

	// fetch all the governors, insert them into currGovernors
	keeper.IterateGovernorsByPower(ctx, func(index int64, governor v1.GovernorI) (stop bool) {
		currGovernors[governor.GetAddress().String()] = v1.NewGovernorGovInfo(
			governor.GetAddress(),
			governor.GetDelegations(),
			[]v1.ValidatorDelegation{},
			v1.WeightedVoteOptions{},
		)
		return false
	})

	keeper.IterateVotes(ctx, proposal.Id, func(vote v1.Vote) bool {
		var governor v1.GovernorI = nil

		voter := sdk.MustAccAddressFromBech32(vote.Voter)

		// if voter is a governor record it in the map
		govAddrStr := v1.GovernorAddress(voter.Bytes()).String()
		if gov, ok := currGovernors[govAddrStr]; ok {
			gov.Vote = vote.Options
			currGovernors[govAddrStr] = gov
		}

		g, governorDelegationPercentage := keeper.GetGovernor(ctx, voter)
		if g != nil {
			if g, ok := currGovernors[g.GetAddress().String()]; ok {
				governor = g
			}
		}

		// iterate over all delegations from voter
		keeper.sk.IterateDelegations(ctx, voter, func(index int64, delegation stakingtypes.DelegationI) (stop bool) {
			valAddrStr := delegation.GetValidatorAddr().String()

			if val, ok := currValidators[valAddrStr]; ok {
				// delegation shares * bonded / total shares
				votingPower := delegation.GetShares().MulInt(val.GetBondedTokens()).Quo(val.GetDelegatorShares())

				for _, option := range vote.Options {
					weight, _ := sdk.NewDecFromStr(option.Weight)
					subPower := votingPower.Mul(weight)
					results[option.Option] = results[option.Option].Add(subPower)
				}
				totalVotingPower = totalVotingPower.Add(votingPower)

				// remove the delegation shares from the governor
				if governor != nil {
					d := governor.GetDelegationDeductions(delegation.GetValidatorAddr())
					d = d.Add(delegation.GetShares().Mul(governorDelegationPercentage))
					governor.SetDelegatorDeductions(delegation.GetValidatorAddr(), d)
				}
			}

			return false
		})

		keeper.deleteVote(ctx, vote.ProposalId, voter)
		return false
	})

	/* DISABLED on AtomOne - Voting can only be done with your own stake
	// iterate over the validators again to tally their voting power
	for _, val := range currValidators {
		if len(val.Vote) == 0 {
			continue
		}

		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := sharesAfterDeductions.MulInt(val.BondedTokens).Quo(val.DelegatorShares)

		for _, option := range val.Vote {
			weight, _ := sdk.NewDecFromStr(option.Weight)
			subPower := votingPower.Mul(weight)
			results[option.Option] = results[option.Option].Add(subPower)
		}
		totalVotingPower = totalVotingPower.Add(votingPower)
	}
	*/

	// iterate over the governors again to tally their voting power
	for _, gov := range currGovernors {
		if len(gov.Vote) == 0 {
			continue
		}

		// Calculate the voting power of governors that have voted.
		// Iterate over all validators the governor has delegation shares assigned to.
		// As governor are simply voters that need to have 100% of their bonded tokens
		// delegated to them and their shares were deducted when iterating over votes
		// we don't need to handle special cases.
		for _, d := range gov.Delegations {
			valAddrStr := d.ValidatorAddress
			shares := d.Shares
			if val, ok := currValidators[valAddrStr]; ok {
				sharesAfterDeductions := shares.Sub(gov.GetDelegationDeductions(val.GetOperator()))
				votingPower := sharesAfterDeductions.MulInt(val.GetBondedTokens()).Quo(val.GetDelegatorShares())

				for _, option := range gov.Vote {
					weight, _ := sdk.NewDecFromStr(option.Weight)
					subPower := votingPower.Mul(weight)
					results[option.Option] = results[option.Option].Add(subPower)
				}
				totalVotingPower = totalVotingPower.Add(votingPower)
			}
		}
	}

	params := keeper.GetParams(ctx)
	tallyResults = v1.NewTallyResultFromMap(results)

	// TODO: Upgrade the spec to cover all of these cases & remove pseudocode.
	// If there is no staked coins, the proposal fails
	totalBondedTokens := keeper.sk.TotalBondedTokens(ctx)
	if totalBondedTokens.IsZero() {
		return false, false, tallyResults
	}

	// If there is not enough quorum of votes, the proposal fails
	percentVoting := totalVotingPower.Quo(sdk.NewDecFromInt(totalBondedTokens))
	quorum, _ := sdk.NewDecFromStr(params.Quorum)
	if percentVoting.LT(quorum) {
		return false, params.BurnVoteQuorum, tallyResults
	}

	// If no one votes (everyone abstains), proposal fails
	if totalVotingPower.Sub(results[v1.OptionAbstain]).Equal(math.LegacyZeroDec()) {
		return false, false, tallyResults
	}

	// If more than 1/3 of voters veto, proposal fails
	vetoThreshold, _ := sdk.NewDecFromStr(params.VetoThreshold)
	if results[v1.OptionNoWithVeto].Quo(totalVotingPower).GT(vetoThreshold) {
		return false, params.BurnVoteVeto, tallyResults
	}

	// If more than 1/2 of non-abstaining voters vote Yes, proposal passes
	threshold, _ := sdk.NewDecFromStr(params.Threshold)
	if results[v1.OptionYes].Quo(totalVotingPower.Sub(results[v1.OptionAbstain])).GT(threshold) {
		return true, false, tallyResults
	}

	// If more than 1/2 of non-abstaining voters vote No, proposal fails
	return false, false, tallyResults
}
