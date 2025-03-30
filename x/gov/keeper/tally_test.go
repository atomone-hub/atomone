package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func TestTally(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*fixture)
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
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
			// Create the test fixture
			s := newFixture(t, ctx, 10, 6, 3, govKeeper, mocks)
			// Setup governor self delegation
			for _, govAddr := range s.govAddrs {
				accAddr := sdk.AccAddress(govAddr)
				s.delegate(accAddr, s.valAddrs[0], 1)
				s.delegate(accAddr, s.valAddrs[1], 2)
				govKeeper.DelegateToGovernor(ctx, accAddr, govAddr)
			}
			// Submit and activate a proposal
			proposal, err := govKeeper.SubmitProposal(ctx, tt.proposalMsgs, "", "title", "summary", s.delAddrs[0])
			require.NoError(t, err)
			govKeeper.ActivateVotingPeriod(ctx, proposal)
			s.proposal = proposal
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
		setup          func(*fixture)
		expectedQuorum bool
	}{
		{
			name:         "no votes: no quorum",
			proposalMsgs: TestProposal,
			setup: func(s *fixture) {
			},
			expectedQuorum: false,
		},
		{
			name:         "not enough votes: no quorum",
			proposalMsgs: TestProposal,
			setup: func(s *fixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_NO)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
			},
			expectedQuorum: false,
		},
		{
			name:         "enough votes: quorum",
			proposalMsgs: TestProposal,
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
				s.delegate(s.delAddrs[0], s.valAddrs[0], 500000)
				s.keeper.DelegateToGovernor(s.ctx, s.delAddrs[0], s.govAddrs[0])
				s.governorVote(s.govAddrs[0], v1.VoteOption_VOTE_OPTION_ABSTAIN)
			},
			expectedQuorum: true,
		},
		{
			name: "amendment quorum not reached",
			setup: func(s *fixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[1], v1.VoteOption_VOTE_OPTION_YES)
			},
			proposalMsgs:   TestAmendmentProposal,
			expectedQuorum: false,
		},
		{
			name: "amendment quorum reached",
			setup: func(s *fixture) {
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
			setup: func(s *fixture) {
				s.validatorVote(s.valAddrs[0], v1.VoteOption_VOTE_OPTION_YES)
				s.validatorVote(s.valAddrs[2], v1.VoteOption_VOTE_OPTION_NO)
			},
			proposalMsgs:   TestLawProposal,
			expectedQuorum: false,
		},
		{
			name: "law quorum reached",
			setup: func(s *fixture) {
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
			// Submit and activate a proposal
			s := newFixture(t, ctx, 10, 5, 3, govKeeper, mocks)
			// Setup governor self delegation
			for _, govAddr := range s.govAddrs {
				accAddr := sdk.AccAddress(govAddr)
				s.delegate(accAddr, s.valAddrs[0], 1)
				s.delegate(accAddr, s.valAddrs[1], 2)
				govKeeper.DelegateToGovernor(ctx, accAddr, govAddr)
			}
			proposal, err := govKeeper.SubmitProposal(ctx, tt.proposalMsgs, "", "title", "summary", s.delAddrs[0])
			require.NoError(t, err)
			s.proposal = proposal
			govKeeper.ActivateVotingPeriod(ctx, proposal)
			tt.setup(s)

			quorum, err := govKeeper.HasReachedQuorum(ctx, proposal)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedQuorum, quorum)
			if tt.expectedQuorum {
				// Assert votes are still here after HasReachedQuorum
				votes := s.keeper.GetVotes(s.ctx, proposal.Id)
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
