package keeper

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

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
	quorum, threshold, err := keeper.getQuorumAndThreshold(ctx, proposal)
	if err != nil {
		return false, false, percentVoting, tallyResults
	}
	if percentVoting.LT(quorum) {
		return false, params.BurnVoteQuorum, percentVoting, tallyResults
	}

	// If no one votes (everyone abstains), proposal fails
	if totalVotingPower.Sub(results[v1.OptionAbstain]).Equal(math.LegacyZeroDec()) {
		return false, false, percentVoting, tallyResults
	}

	if results[v1.OptionYes].Quo(totalVotingPower.Sub(results[v1.OptionAbstain])).GT(threshold) {
		return true, false, percentVoting, tallyResults
	}

	// If more than 1/2 of non-abstaining voters vote No, proposal fails
	return false, false, percentVoting, tallyResults
}

// HasReachedQuorum returns whether or not a proposal has reached quorum
// this is just a stripped down version of the Tally function above
func (keeper Keeper) HasReachedQuorum(ctx sdk.Context, proposal v1.Proposal) (quorumPassed bool, err error) {
	// If there is no staked coins, the proposal has not reached quorum
	totalBonded := keeper.sk.TotalBondedTokens(ctx)
	if totalBonded.IsZero() {
		return false, nil
	}

	/* DISABLED on AtomOne - no possible increase of computation speed by
	 iterating over validators since vote inheritance is disabled.
	 Keeping as comment because this should be adapted with governors loop

	// we check first if voting power of validators alone is enough to pass quorum
	// and if so, we return true skipping the iteration over all votes
	// can speed up computation in case quorum is already reached by validator votes alone
	approxTotalVotingPower := math.LegacyZeroDec()
	for _, val := range currValidators {
		_, ok := keeper.GetVote(ctx, proposal.Id, sdk.AccAddress(val.GetOperator()))
		if ok {
			approxTotalVotingPower = approxTotalVotingPower.Add(math.LegacyNewDecFromInt(val.GetBondedTokens()))
		}
	}
	// check and return whether or not the proposal has reached quorum
	approxPercentVoting := approxTotalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
	if approxPercentVoting.GTE(quorum) {
		return true, nil
	}
	*/

	// voting power of validators does not reach quorum, let's tally all votes
	currValidators := keeper.getBondedValidatorsByAddress(ctx)
	totalVotingPower, _ := keeper.tallyVotes(ctx, proposal, currValidators, false)

	// check and return whether or not the proposal has reached quorum
	percentVoting := totalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
	quorum, _, err := keeper.getQuorumAndThreshold(ctx, proposal)
	if err != nil {
		return false, err
	}
	return percentVoting.GTE(quorum), nil
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
	currValidators map[string]stakingtypes.ValidatorI, isFinal bool,
) (totalVotingPower math.LegacyDec, results map[v1.VoteOption]math.LegacyDec) {
	totalVotingPower = math.LegacyZeroDec()
	if isFinal {
		results = make(map[v1.VoteOption]math.LegacyDec)
		results[v1.OptionYes] = math.LegacyZeroDec()
		results[v1.OptionAbstain] = math.LegacyZeroDec()
		results[v1.OptionNo] = math.LegacyZeroDec()
	}

	keeper.IterateVotes(ctx, proposal.Id, func(vote v1.Vote) bool {
		voter := sdk.MustAccAddressFromBech32(vote.Voter)
		// iterate over all delegations from voter, deduct from any delegated-to validators
		keeper.sk.IterateDelegations(ctx, voter, func(index int64, delegation stakingtypes.DelegationI) (stop bool) {
			valAddrStr := delegation.GetValidatorAddr().String()

			if val, ok := currValidators[valAddrStr]; ok {
				// delegation shares * bonded / total shares
				votingPower := delegation.GetShares().MulInt(val.GetBondedTokens()).Quo(val.GetDelegatorShares())

				if isFinal {
					for _, option := range vote.Options {
						weight, _ := math.LegacyNewDecFromStr(option.Weight)
						subPower := votingPower.Mul(weight)
						results[option.Option] = results[option.Option].Add(subPower)
					}
				}
				totalVotingPower = totalVotingPower.Add(votingPower)
			}

			return false
		})

		if isFinal {
			keeper.deleteVote(ctx, vote.ProposalId, voter)
		}
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

	return totalVotingPower, results
}

// getQuorumAndThreshold iterates over the proposal's messages to returns the
// appropriate quorum and threshold.
func (keeper Keeper) getQuorumAndThreshold(ctx sdk.Context, proposal v1.Proposal) (sdk.Dec, sdk.Dec, error) {
	params := keeper.GetParams(ctx)
	quorum := keeper.GetQuorum(ctx)
	threshold, err := sdk.NewDecFromStr(params.Threshold)
	if err != nil {
		return sdk.Dec{}, sdk.Dec{}, fmt.Errorf("parsing params.Threshold: %w", err)
	}

	// Check if a proposal message is an ExecLegacyContent message
	if len(proposal.Messages) > 0 {
		var sdkMsg sdk.Msg
		for _, msg := range proposal.Messages {
			if err := keeper.cdc.UnpackAny(msg, &sdkMsg); err == nil {
				// Check if proposal is a law or constitution amendment and adjust the
				// quorum and threshold accordingly
				switch sdkMsg.(type) {
				case *v1.MsgProposeConstitutionAmendment:
					q, err := sdk.NewDecFromStr(params.ConstitutionAmendmentQuorum)
					if err != nil {
						return sdk.Dec{}, sdk.Dec{}, fmt.Errorf("parsing params.ConstitutionAmendmentQuorum: %w", err)
					}
					if quorum.LT(q) {
						quorum = q
					}
					t, err := sdk.NewDecFromStr(params.ConstitutionAmendmentThreshold)
					if err != nil {
						return sdk.Dec{}, sdk.Dec{}, fmt.Errorf("parsing params.ConstitutionAmendmentThreshold: %w", err)
					}
					if threshold.LT(t) {
						threshold = t
					}
				case *v1.MsgProposeLaw:
					q, err := sdk.NewDecFromStr(params.LawQuorum)
					if err != nil {
						return sdk.Dec{}, sdk.Dec{}, fmt.Errorf("parsing params.LawQuorum: %w", err)
					}
					if quorum.LT(q) {
						quorum = q
					}
					t, err := sdk.NewDecFromStr(params.LawThreshold)
					if err != nil {
						return sdk.Dec{}, sdk.Dec{}, fmt.Errorf("parsing params.LawThreshold: %w", err)
					}
					if threshold.LT(t) {
						threshold = t
					}
				}
			}
		}
	}
	return quorum, threshold, nil
}
