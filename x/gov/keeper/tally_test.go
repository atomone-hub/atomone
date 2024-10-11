package keeper_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov/keeper"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

type tallyFixture struct {
	t *testing.T

	proposal    v1.Proposal
	valAddrs    []sdk.ValAddress
	delAddrs    []sdk.AccAddress
	govAddrs    []types.GovernorAddress
	totalBonded int64
	validators  []stakingtypes.Validator
	delegations []stakingtypes.Delegation

	keeper *keeper.Keeper
	ctx    sdk.Context
	mocks  mocks
}

// newTallyFixture returns a configured fixture for testing the govKeeper.Tally
// method.
// - setup TotalBondedTokens call
// - initiates the validators with a self delegation of 1:
//   - setup IterateBondedValidatorsByPower call
//   - setup IterateDelegations call for validators
func newTallyFixture(t *testing.T, ctx sdk.Context, proposal v1.Proposal,
	valAddrs []sdk.ValAddress, delAddrs []sdk.AccAddress, govAddrs []types.GovernorAddress,
	govKeeper *keeper.Keeper, mocks mocks,
) *tallyFixture {
	s := &tallyFixture{
		t:        t,
		ctx:      ctx,
		proposal: proposal,
		valAddrs: valAddrs,
		delAddrs: delAddrs,
		govAddrs: govAddrs,
		keeper:   govKeeper,
		mocks:    mocks,
	}
	mocks.stakingKeeper.EXPECT().TotalBondedTokens(gomock.Any()).
		DoAndReturn(func(_ context.Context) sdkmath.Int {
			return sdkmath.NewInt(s.totalBonded)
		}).MaxTimes(1)

	// Mocks a bunch of validators
	for i := 0; i < len(valAddrs); i++ {
		s.validators = append(s.validators, stakingtypes.Validator{
			OperatorAddress: valAddrs[i].String(),
			Status:          stakingtypes.Bonded,
			Tokens:          sdkmath.ZeroInt(),
			DelegatorShares: sdkmath.LegacyZeroDec(),
		})
		// validator self delegation
		s.delegate(sdk.AccAddress(valAddrs[i]), valAddrs[i], 1)
	}
	mocks.stakingKeeper.EXPECT().
		IterateBondedValidatorsByPower(ctx, gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, fn func(index int64, validator stakingtypes.ValidatorI) bool) error {
				for i := 0; i < len(valAddrs); i++ {
					fn(int64(i), s.validators[i])
				}
				return nil
			})
	mocks.stakingKeeper.EXPECT().
		IterateDelegations(ctx, gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, voter sdk.AccAddress, fn func(index int64, d stakingtypes.DelegationI) bool) error {
				for i, d := range s.delegations {
					if d.DelegatorAddress == voter.String() {
						fn(int64(i), d)
					}
				}
				return nil
			}).AnyTimes()
	mocks.stakingKeeper.EXPECT().GetValidator(ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, addr sdk.ValAddress) (stakingtypes.ValidatorI, bool) {
			for i := 0; i < len(valAddrs); i++ {
				if valAddrs[i].String() == addr.String() {
					return s.validators[i], true
				}
			}
			return nil, false
		}).AnyTimes()
	mocks.stakingKeeper.EXPECT().GetDelegation(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, del sdk.AccAddress, val sdk.ValAddress) (stakingtypes.Delegation, bool) {
			for _, d := range s.delegations {
				if d.DelegatorAddress == del.String() && d.ValidatorAddress == val.String() {
					return d, true
				}
			}
			return stakingtypes.Delegation{}, false
		}).AnyTimes()

	// Create active governors
	for i := 0; i < len(govAddrs)-1; i++ {
		governor, err := v1.NewGovernor(govAddrs[i].String(), v1.GovernorDescription{}, time.Now())
		require.NoError(t, err)
		govKeeper.SetGovernor(ctx, governor)
		// governor self delegation
		accAddr := sdk.AccAddress(govAddrs[i])
		s.delegate(accAddr, valAddrs[0], 1)
		s.delegate(accAddr, valAddrs[1], 2)
		govKeeper.DelegateToGovernor(ctx, accAddr, govAddrs[i])
	}
	// Create one inactive governor
	inactiveGovAddr := govAddrs[len(govAddrs)-1]
	governor, err := v1.NewGovernor(inactiveGovAddr.String(), v1.GovernorDescription{}, time.Now())
	require.NoError(t, err)
	governor.Status = v1.Inactive
	govKeeper.SetGovernor(ctx, governor)
	accAddr := sdk.AccAddress(inactiveGovAddr)
	s.delegate(accAddr, valAddrs[0], 1)
	s.delegate(accAddr, valAddrs[1], 2)
	govKeeper.DelegateToGovernor(ctx, accAddr, inactiveGovAddr)
	return s
}

// delegate updates the tallyFixture delegations and validators fields.
// WARNING: delegate must be called *after* any calls to govKeeper.DelegateToGovernor
// because the hooks are not invoked in this test setup.
func (s *tallyFixture) delegate(delegator sdk.AccAddress, validator sdk.ValAddress, m int64) {
	// Increment total bonded according to each delegations
	s.totalBonded += m
	delegation := stakingtypes.Delegation{
		DelegatorAddress: delegator.String(),
		ValidatorAddress: validator.String(),
	}
	// Increase validator shares and tokens, compute delegation.Shares
	for i := 0; i < len(s.validators); i++ {
		if s.validators[i].OperatorAddress == validator.String() {
			s.validators[i], delegation.Shares = s.validators[i].AddTokensFromDel(sdk.NewInt(m))
			break
		}
	}
	s.delegations = append(s.delegations, delegation)
}

// vote calls govKeeper.Vote()
func (s *tallyFixture) vote(voter sdk.AccAddress, vote v1.VoteOption) {
	err := s.keeper.AddVote(s.ctx, s.proposal.Id, voter, v1.NewNonSplitVoteOption(vote), "")
	require.NoError(s.t, err)
}

func (s *tallyFixture) validatorVote(voter sdk.ValAddress, vote v1.VoteOption) {
	s.vote(sdk.AccAddress(voter), vote)
}

func (s *tallyFixture) governorVote(voter types.GovernorAddress, vote v1.VoteOption) {
	s.vote(sdk.AccAddress(voter), vote)
}

func TestTally(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*tallyFixture)
		proposalMsgs  []sdk.Msg
		expectedPass  bool
		expectedBurn  bool
		expectedTally v1.TallyResult
		expectedError string
	}{
		{
			name:         "no votes: prop fails/burn deposit",
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:     "0",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "one validator votes: prop fails/burn deposit",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true, // burn because quorum not reached
			expectedTally: v1.TallyResult{
				YesCount:     "0",
				AbstainCount: "0",
				NoCount:      "1",
			},
		},
		{
			name: "one account votes without delegation: prop fails/burn deposit",
			setup: func(s *tallyFixture) {
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true, // burn because quorum not reached
			expectedTally: v1.TallyResult{
				YesCount:     "0",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "one delegator votes: prop fails/burn deposit",
			setup: func(s *tallyFixture) {
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true, // burn because quorum not reached
			expectedTally: v1.TallyResult{
				YesCount:     "2",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "one governor vote w/o delegation: prop fails/burn deposit",
			setup: func(s *tallyFixture) {
				// gov0 VP=3 vote=yes
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true, // burn because quorum not reached
			expectedTally: v1.TallyResult{
				YesCount:     "3",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "one governor vote inherits delegation that didn't vote",
			setup: func(s *tallyFixture) {
				// del0 VP=5 del=gov0 didn't vote
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.delegate(s.delAddrs[0], s.valAddrs[1], 3)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[0])
				// gov0 VP=3 vote=yes
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: true,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "8",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "inactive governor vote doesn't inherit delegation that didn't vote",
			setup: func(s *tallyFixture) {
				// del0 VP=5 del=gov2(inactive) didn't vote
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.delegate(s.delAddrs[0], s.valAddrs[1], 3)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[2])
				// gov2(inactive) VP=3 vote=yes
				s.governorVote(s.govAddrs[2], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:     "3",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "one governor votes yes, one delegator votes yes",
			setup: func(s *tallyFixture) {
				// del0 VP=5 del=gov0 vote=yes
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.delegate(s.delAddrs[0], s.valAddrs[1], 3)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[0])
				// gov0 VP=3 vote=yes
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: true,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "8",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "one governor votes yes, one delegator votes no",
			setup: func(s *tallyFixture) {
				// del0 VP=5 del=gov0 vote=no
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.delegate(s.delAddrs[0], s.valAddrs[1], 3)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[0])
				// gov0 VP=3 vote=yes
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "3",
				AbstainCount: "0",
				NoCount:      "5",
			},
		},
		{
			// Same case as previous one but with reverted vote order
			name: "one delegator votes no, one governor votes yes",
			setup: func(s *tallyFixture) {
				// gov0 VP=3 del=gov0 vote=yes
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				// del0 VP=5 vote=no
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.delegate(s.delAddrs[0], s.valAddrs[1], 3)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[0])
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "3",
				AbstainCount: "0",
				NoCount:      "5",
			},
		},
		{
			name: "one governor votes and some delegations vote",
			setup: func(s *tallyFixture) {
				// del0 VP=2 del=gov0 vote=no
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[0])
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				// del1 VP=3 del=gov0 didn't vote (so VP goes to gov0's vote)
				s.delegate(s.delAddrs[1], s.valAddrs[1], 3)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[1], s.govAddrs[0])
				// del2 VP=4 del=gov0 vote=abstain
				s.delegate(s.delAddrs[2], s.valAddrs[0], 4)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[2], s.govAddrs[0])
				s.vote(s.delAddrs[2], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				// del3 VP=5 del=gov0 vote=yes
				s.delegate(s.delAddrs[3], s.valAddrs[1], 2)
				s.delegate(s.delAddrs[3], s.valAddrs[2], 3)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[3], s.govAddrs[0])
				s.vote(s.delAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				// del4 VP=4 del=gov1 vote=no
				s.delegate(s.delAddrs[4], s.valAddrs[3], 4)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[4], s.govAddrs[1])
				s.vote(s.delAddrs[4], v1.VoteOption_VOTE_OPTION_NO)
				// del5 VP=6 del=gov1 didn't vote (so VP does to gov1's vote)
				s.delegate(s.delAddrs[5], s.valAddrs[3], 6)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[5], s.govAddrs[1])
				// gov0 VP=3 vote=yes
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "11",
				AbstainCount: "4",
				NoCount:      "6",
			},
		},
		{
			name: "two governors vote and some delegations vote",
			setup: func(s *tallyFixture) {
				// del0 VP=2 del=gov0 vote=no
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[0])
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				// del1 VP=3 del=gov0 didn't vote (so VP goes to gov0's vote)
				s.delegate(s.delAddrs[1], s.valAddrs[1], 3)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[1], s.govAddrs[0])
				// del2 VP=4 del=gov0 vote=abstain
				s.delegate(s.delAddrs[2], s.valAddrs[0], 4)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[2], s.govAddrs[0])
				s.vote(s.delAddrs[2], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				// del3 VP=5 del=gov0 vote=yes
				s.delegate(s.delAddrs[3], s.valAddrs[1], 2)
				s.delegate(s.delAddrs[3], s.valAddrs[2], 3)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[3], s.govAddrs[0])
				s.vote(s.delAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				// del4 VP=4 del=gov1 vote=no
				s.delegate(s.delAddrs[4], s.valAddrs[3], 4)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[4], s.govAddrs[1])
				s.vote(s.delAddrs[4], v1.VoteOption_VOTE_OPTION_NO)
				// del5 VP=6 del=gov1 didn't vote (so VP does to gov1's vote)
				s.delegate(s.delAddrs[5], s.valAddrs[3], 6)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[5], s.govAddrs[1])
				// gov0 VP=3 vote=yes
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				// gov1 VP=3 vote=abstain
				s.governorVote(s.govAddrs[1], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "11",
				AbstainCount: "13",
				NoCount:      "6",
			},
		},
		{
			name: "one delegator votes yes, validator votes also yes: prop fails/burn deposit",
			setup: func(s *tallyFixture) {
				s.delegate(s.delAddrs[0], s.valAddrs[0], 1)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true, // burn because quorum not reached
			expectedTally: v1.TallyResult{
				YesCount:     "2",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "one delegator votes yes, validator votes no: prop fails/burn deposit",
			setup: func(s *tallyFixture) {
				s.delegate(s.delAddrs[0], s.valAddrs[0], 1)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true, // burn because quorum not reached
			expectedTally: v1.TallyResult{
				YesCount:     "1",
				AbstainCount: "0",
				NoCount:      "1",
			},
		},
		{
			name: "validator votes yes, doesn't inherit delegations",
			setup: func(s *tallyFixture) {
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.delegate(s.delAddrs[1], s.valAddrs[0], 2)
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: true, // burn because quorum not reached
			expectedTally: v1.TallyResult{
				YesCount:     "0",
				AbstainCount: "0",
				NoCount:      "1",
			},
		},
		{
			// one delegator delegates 42 shares to 2 different validators (21 each)
			// delegator votes yes
			// first validator votes yes
			// second validator votes no
			// third validator (no delegation) votes abstain
			name: "delegator with mixed delegations: prop pass",
			setup: func(s *tallyFixture) {
				s.delegate(s.delAddrs[0], s.valAddrs[0], 2)
				s.delegate(s.delAddrs[0], s.valAddrs[1], 2)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			proposalMsgs: TestProposal,
			expectedPass: true,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "5",
				AbstainCount: "1",
				NoCount:      "1",
			},
		},
		{
			name: "quorum reached with only abstain: prop fails",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[4], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "0",
				AbstainCount: "5",
				NoCount:      "0",
			},
		},
		{
			name: "quorum reached with yes<=.667: prop fails",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[4], v1.VoteOption_VOTE_OPTION_NO)
			},
			proposalMsgs: TestProposal,
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "2",
				AbstainCount: "0",
				NoCount:      "3",
			},
		},
		{
			name: "quorum reached with yes>.667: prop succeeds",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[4], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[5], v1.VoteOption_VOTE_OPTION_NO)
			},
			proposalMsgs: TestProposal,
			expectedPass: true,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "5",
				AbstainCount: "0",
				NoCount:      "1",
			},
		},
		{
			name: "quorum reached thanks to abstain, yes>.667: prop succeeds",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[4], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[5], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			proposalMsgs: TestProposal,
			expectedPass: true,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "3",
				AbstainCount: "2",
				NoCount:      "1",
			},
		},
		{
			name: "amendment quorum not reached: prop fails",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestAmendmentProposal,
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:     "2",
				AbstainCount: "0",
				NoCount:      "0",
			},
		},
		{
			name: "amendment quorum reached and threshold not reached: prop fails",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[4], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[5], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			proposalMsgs: TestAmendmentProposal,
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "2",
				AbstainCount: "3",
				NoCount:      "1",
			},
		},
		{
			name: "amendment quorum reached and threshold reached: prop passes",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[4], v1.VoteOption_VOTE_OPTION_YES)
				s.delegate(s.delAddrs[0], s.valAddrs[5], 2)
				s.delegate(s.delAddrs[0], s.valAddrs[6], 2)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.delegate(s.delAddrs[1], s.valAddrs[5], 1)
				s.delegate(s.delAddrs[1], s.valAddrs[6], 1)
				s.vote(s.delAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs: TestAmendmentProposal,
			expectedPass: true,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "10",
				AbstainCount: "0",
				NoCount:      "1",
			},
		},
		{
			name: "law quorum not reached: prop fails",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
			},
			proposalMsgs: TestLawProposal,
			expectedPass: false,
			expectedBurn: true,
			expectedTally: v1.TallyResult{
				YesCount:     "1",
				AbstainCount: "0",
				NoCount:      "1",
			},
		},
		{
			name: "law quorum reached and threshold not reached: prop fails",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[4], v1.VoteOption_VOTE_OPTION_ABSTAIN)
				s.validatorVote(s.valAddrs[5], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			proposalMsgs: TestLawProposal,
			expectedPass: false,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "2",
				AbstainCount: "3",
				NoCount:      "1",
			},
		},
		{
			name: "law quorum reached and threshold reached: prop passes",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[5], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			proposalMsgs: TestLawProposal,
			expectedPass: true,
			expectedBurn: false,
			expectedTally: v1.TallyResult{
				YesCount:     "4",
				AbstainCount: "1",
				NoCount:      "0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			govKeeper, mocks, _, ctx := setupGovKeeper(t, mockAccountKeeperExpectations)
			params := v1.DefaultParams()
			// Ensure params value are different than false
			params.BurnVoteQuorum = true
			params.MinGovernorSelfDelegation = "1"
			err := govKeeper.SetParams(ctx, params)
			require.NoError(t, err)
			var (
				numVals       = 10
				numDelegators = 6
				numGovernors  = 3
				addrs         = simtestutil.CreateRandomAccounts(numVals + numDelegators + numGovernors)
				valAddrs      = simtestutil.ConvertAddrsToValAddrs(addrs[:numVals])
				delAddrs      = addrs[numVals : numVals+numDelegators]
				govAddrs      = convertAddrsToGovAddrs(addrs[numVals+numDelegators:])
			)
			// Submit and activate a proposal
			proposal, err := govKeeper.SubmitProposal(ctx, tt.proposalMsgs, "", "title", "summary", delAddrs[0])
			require.NoError(t, err)
			govKeeper.ActivateVotingPeriod(ctx, proposal)
			// Create the test fixture
			s := newTallyFixture(t, ctx, proposal, valAddrs, delAddrs, govAddrs, govKeeper, mocks)
			if tt.setup != nil {
				tt.setup(s)
			}

			pass, burn, tally := govKeeper.Tally(ctx, proposal)

			assert.Equal(t, tt.expectedPass, pass, "wrong pass")
			assert.Equal(t, tt.expectedBurn, burn, "wrong burn")
			assert.Equal(t, tt.expectedTally, tally)
			assert.Empty(t, govKeeper.GetVotes(ctx, proposal.Id), "votes not be removed after tally")
		})
	}
}

func TestHasReachedQuorum(t *testing.T) {
	tests := []struct {
		name           string
		proposalMsgs   []sdk.Msg
		setup          func(*tallyFixture)
		expectedQuorum bool
	}{
		{
			name:         "no votes: no quorum",
			proposalMsgs: TestProposal,
			setup: func(s *tallyFixture) {
			},
			expectedQuorum: false,
		},
		{
			name:         "not enough votes: no quorum",
			proposalMsgs: TestProposal,
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
			},
			expectedQuorum: false,
		},
		{
			name:         "enough votes: quorum",
			proposalMsgs: TestProposal,
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.delegate(s.delAddrs[0], s.valAddrs[2], 500000)
				s.delegate(s.delAddrs[0], s.valAddrs[3], 500000)
				s.delegate(s.delAddrs[0], s.valAddrs[4], 500000)
				s.delegate(s.delAddrs[0], s.valAddrs[5], 500000)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			expectedQuorum: true,
		},
		{
			name:         "quorum reached by governor vote inheritance",
			proposalMsgs: TestProposal,
			setup: func(s *tallyFixture) {
				s.delegate(s.delAddrs[0], s.valAddrs[0], 500000)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[0])
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			expectedQuorum: true,
		},
		{
			name: "amendment quorum not reached",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs:   TestAmendmentProposal,
			expectedQuorum: false,
		},
		{
			name: "amendment quorum reached",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[4], v1.VoteOption_VOTE_OPTION_YES)
				s.delegate(s.delAddrs[0], s.valAddrs[5], 2)
				s.delegate(s.delAddrs[0], s.valAddrs[6], 2)
				s.vote(s.delAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.delegate(s.delAddrs[1], s.valAddrs[5], 1)
				s.delegate(s.delAddrs[1], s.valAddrs[6], 1)
				s.vote(s.delAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs:   TestAmendmentProposal,
			expectedQuorum: true,
		},
		{
			name: "law quorum not reached",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
			},
			proposalMsgs:   TestLawProposal,
			expectedQuorum: false,
		},
		{
			name: "law quorum reached",
			setup: func(s *tallyFixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[3], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[5], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			proposalMsgs:   TestLawProposal,
			expectedQuorum: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			govKeeper, mocks, _, ctx := setupGovKeeper(t, mockAccountKeeperExpectations)
			params := v1.DefaultParams()
			params.MinGovernorSelfDelegation = "1"
			err := govKeeper.SetParams(ctx, params)
			require.NoError(t, err)
			var (
				numVals       = 10
				numDelegators = 5
				numGovernors  = 3
				addrs         = simtestutil.CreateRandomAccounts(numVals + numDelegators + numGovernors)
				valAddrs      = simtestutil.ConvertAddrsToValAddrs(addrs[:numVals])
				delAddrs      = addrs[numVals : numVals+numDelegators]
				govAddrs      = convertAddrsToGovAddrs(addrs[numVals+numDelegators:])
			)
			// Submit and activate a proposal
			proposal, err := govKeeper.SubmitProposal(ctx, tt.proposalMsgs, "", "title", "summary", delAddrs[0])
			require.NoError(t, err)
			govKeeper.ActivateVotingPeriod(ctx, proposal)
			suite := newTallyFixture(t, ctx, proposal, valAddrs, delAddrs, govAddrs, govKeeper, mocks)
			tt.setup(suite)

			quorum, err := govKeeper.HasReachedQuorum(ctx, proposal)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedQuorum, quorum)
			if tt.expectedQuorum {
				// Assert votes are still here after HasReachedQuorum
				votes := suite.keeper.GetVotes(suite.ctx, proposal.Id)
				assert.NotEmpty(t, votes, "votes must be kept after HasReachedQuorum")
			}
		})
	}
}

func convertAddrsToGovAddrs(addrs []sdk.AccAddress) []types.GovernorAddress {
	govAddrs := make([]types.GovernorAddress, len(addrs))
	for i, addr := range addrs {
		govAddrs[i] = types.GovernorAddress(addr)
	}
	return govAddrs
}
