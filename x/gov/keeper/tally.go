package keeper

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// Tally iterates over the votes and updates the tally of a proposal based on the voting power of the
// voters
func (keeper Keeper) Tally(ctx sdk.Context, proposal v1.Proposal) (passes bool, burnDeposits bool, tallyResults v1.TallyResult) {
	currValidators := keeper.getBondedValidatorsByAddress(ctx)
	totalVotingPower, results := keeper.tallyVotes(ctx, proposal, currValidators, true)

	params := keeper.GetParams(ctx)
	tallyResults = v1.NewTallyResultFromMap(results)

	// If there is no staked coins, the proposal fails
	totalBonded := keeper.sk.TotalBondedTokens(ctx)
	if totalBonded.IsZero() {
		return false, false, tallyResults
	}

	// If there is not enough quorum of votes, the proposal fails
	percentVoting := totalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
	quorum, threshold, err := keeper.getQuorumAndThreshold(ctx, proposal)
	if err != nil {
		return false, false, tallyResults
	}
	if percentVoting.LT(quorum) {
		return false, params.BurnVoteQuorum, tallyResults
	}

	// If no one votes (everyone abstains), proposal fails
	if totalVotingPower.Sub(results[v1.OptionAbstain]).Equal(math.LegacyZeroDec()) {
		return false, false, tallyResults
	}

	if results[v1.OptionYes].Quo(totalVotingPower.Sub(results[v1.OptionAbstain])).GT(threshold) {
		return true, false, tallyResults
	}

	// If more than 1/2 of non-abstaining voters vote No, proposal fails
	return false, false, tallyResults
}

// HasReachedQuorum returns whether or not a proposal has reached quorum
// this is just a stripped down version of the Tally function above
func (keeper Keeper) HasReachedQuorum(ctx sdk.Context, proposal v1.Proposal) (quorumPassed bool, err error) {
	// If there is no staked coins, the proposal has not reached quorum
	totalBonded := keeper.sk.TotalBondedTokens(ctx)
	if totalBonded.IsZero() {
		return false, nil
	}

	quorum, _, err := keeper.getQuorumAndThreshold(ctx, proposal)
	currValidators := keeper.getBondedValidatorsByAddress(ctx)
	totalVotingPower, _ := keeper.tallyVotes(ctx, proposal, currValidators, false)

	// check and return whether or not the proposal has reached quorum
	percentVoting := totalVotingPower.Quo(math.LegacyNewDecFromInt(totalBonded))
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
				gov, _ := keeper.GetGovernor(ctx, govAddr)
				governor = v1.NewGovernorGovInfo(
					govAddr,
					keeper.GetAllGovernorValShares(ctx, types.MustGovernorAddressFromBech32(gd.GovernorAddress)),
					v1.WeightedVoteOptions{},
					gov.IsActive(),
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
			valAddrStr := delegation.GetValidatorAddr().String()
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
					weight, _ := math.LegacyNewDecFromStr(option.Weight)
					subPower := votingPower.Mul(weight)
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
				weight, _ := sdk.NewDecFromStr(option.Weight)
				subPower := votingPower.Mul(weight)
				results[option.Option] = results[option.Option].Add(subPower)
			}
		}
		totalVotingPower = totalVotingPower.Add(votingPower)
	}
	return totalVotingPower, results
}

// getQuorumAndThreshold iterates over the proposal's messages to returns the
// appropriate quorum and threshold.
func (keeper Keeper) getQuorumAndThreshold(ctx sdk.Context, proposal v1.Proposal) (sdk.Dec, sdk.Dec, error) {
	params := keeper.GetParams(ctx)
	quorum, err := sdk.NewDecFromStr(params.Quorum)
	if err != nil {
		return sdk.Dec{}, sdk.Dec{}, fmt.Errorf("parsing params.Quorum: %w", err)
	}
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
