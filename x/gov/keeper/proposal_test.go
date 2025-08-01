package keeper_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/atomone-hub/atomone/x/gov/client/testutil"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

func (suite *KeeperTestSuite) TestGetSetProposal() {
	tp := TestProposal
	proposal, err := suite.govKeeper.SubmitProposal(suite.ctx, tp, "", "test", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
	suite.Require().NoError(err)
	proposalID := proposal.Id
	suite.govKeeper.SetProposal(suite.ctx, proposal)

	gotProposal, ok := suite.govKeeper.GetProposal(suite.ctx, proposalID)
	suite.Require().True(ok)
	suite.Require().Equal(proposal, gotProposal)
}

func (suite *KeeperTestSuite) TestDeleteProposal() {
	// delete non-existing proposal
	suite.Require().PanicsWithValue(fmt.Sprintf("couldn't find proposal with id#%d", 10),
		func() {
			suite.govKeeper.DeleteProposal(suite.ctx, 10)
		},
	)
	tp := TestProposal
	proposal, err := suite.govKeeper.SubmitProposal(suite.ctx, tp, "", "test", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
	suite.Require().NoError(err)
	proposalID := proposal.Id
	suite.govKeeper.SetProposal(suite.ctx, proposal)
	suite.Require().NotPanics(func() {
		suite.govKeeper.DeleteProposal(suite.ctx, proposalID)
	}, "")
}

func (suite *KeeperTestSuite) TestActivateVotingPeriod() {
	params := v1.DefaultParams()
	params.QuorumCheckCount = 10 // enable quorum check
	quorumTimeout := *params.VotingPeriod - time.Hour*16
	params.QuorumTimeout = &quorumTimeout
	maxVotingPeriodExtension := time.Hour * 16
	params.MaxVotingPeriodExtension = &maxVotingPeriodExtension
	err := suite.govKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)
	currentProposalNumber := suite.govKeeper.GetActiveProposalsNumber(suite.ctx)

	tp := TestProposal
	proposal, err := suite.govKeeper.SubmitProposal(suite.ctx, tp, "", "test", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
	suite.Require().NoError(err)

	suite.Require().Nil(proposal.VotingStartTime)

	suite.govKeeper.ActivateVotingPeriod(suite.ctx, proposal)

	suite.Require().Equal(currentProposalNumber+1, suite.govKeeper.GetActiveProposalsNumber(suite.ctx),
		"active proposal number not incremented after proposal activation")
	proposal, ok := suite.govKeeper.GetProposal(suite.ctx, proposal.Id)
	suite.Require().True(ok)
	suite.Require().True(proposal.VotingStartTime.Equal(suite.ctx.BlockHeader().Time))

	activeIterator := suite.govKeeper.ActiveProposalQueueIterator(suite.ctx, *proposal.VotingEndTime)
	suite.Require().True(activeIterator.Valid())

	proposalID := types.GetProposalIDFromBytes(activeIterator.Value())
	suite.Require().Equal(proposalID, proposal.Id)
	activeIterator.Close()

	quorumTimeoutTime := proposal.VotingStartTime.Add(*params.QuorumTimeout)
	quorumCheckEntry, ok := testutil.GetQuorumCheckQueueEntry(suite.ctx, suite.govKeeper, proposal.Id, quorumTimeoutTime)
	suite.Require().True(ok)
	suite.Require().EqualValues(quorumTimeoutTime, *quorumCheckEntry.QuorumTimeoutTime)
	suite.Require().EqualValues(params.QuorumCheckCount, quorumCheckEntry.QuorumCheckCount)
	suite.Require().Zero(quorumCheckEntry.QuorumChecksDone)

	// delete the proposal to avoid issues with other tests
	suite.Require().NotPanics(func() {
		suite.govKeeper.DeleteProposal(suite.ctx, proposalID)
	}, "")
}

func (suite *KeeperTestSuite) TestDeleteProposalInVotingPeriod() {
	suite.reset()
	tp := TestProposal
	proposal, err := suite.govKeeper.SubmitProposal(suite.ctx, tp, "", "test", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
	suite.Require().NoError(err)
	suite.Require().Nil(proposal.VotingStartTime)

	suite.govKeeper.ActivateVotingPeriod(suite.ctx, proposal)

	proposal, ok := suite.govKeeper.GetProposal(suite.ctx, proposal.Id)
	suite.Require().True(ok)
	suite.Require().True(proposal.VotingStartTime.Equal(suite.ctx.BlockHeader().Time))

	activeIterator := suite.govKeeper.ActiveProposalQueueIterator(suite.ctx, *proposal.VotingEndTime)
	suite.Require().True(activeIterator.Valid())

	proposalID := types.GetProposalIDFromBytes(activeIterator.Value())
	suite.Require().Equal(proposalID, proposal.Id)
	activeIterator.Close()

	// add vote
	voteOptions := []*v1.WeightedVoteOption{{Option: v1.OptionYes, Weight: "1.0"}}
	err = suite.govKeeper.AddVote(suite.ctx, proposal.Id, sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"), voteOptions, "")
	suite.Require().NoError(err)

	suite.Require().NotPanics(func() {
		suite.govKeeper.DeleteProposal(suite.ctx, proposalID)
	}, "")

	// add vote but proposal is deleted along with its VotingPeriodProposalKey
	err = suite.govKeeper.AddVote(suite.ctx, proposal.Id, sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"), voteOptions, "")
	suite.Require().ErrorContains(err, ": inactive proposal")
}

type invalidProposalRoute struct{ v1beta1.TextProposal }

func (invalidProposalRoute) ProposalRoute() string { return "nonexistingroute" }

func (suite *KeeperTestSuite) TestSubmitProposal() {
	govAcct := suite.govKeeper.GetGovernanceAccount(suite.ctx).GetAddress().String()
	_, _, randomAddr := testdata.KeyTestPubAddr()
	tp := v1beta1.TextProposal{Title: "title", Description: "description"}

	testCases := []struct {
		content     v1beta1.Content
		authority   string
		metadata    string
		expectedErr error
	}{
		{&tp, govAcct, "", nil},
		// Keeper does not check the validity of title and description, no error
		{&v1beta1.TextProposal{Title: "", Description: "description"}, govAcct, "", nil},
		{&v1beta1.TextProposal{Title: strings.Repeat("1234567890", 100), Description: "description"}, govAcct, "", nil},
		{&v1beta1.TextProposal{Title: "title", Description: ""}, govAcct, "", nil},
		{&v1beta1.TextProposal{Title: "title", Description: strings.Repeat("1234567890", 1000)}, govAcct, "", nil},
		// error when metadata is too long (>10000)
		{&tp, govAcct, strings.Repeat("a", 100001), types.ErrMetadataTooLong},
		// error when signer is not gov acct
		{&tp, randomAddr.String(), "", types.ErrInvalidSigner},
		// error only when invalid route
		{&invalidProposalRoute{}, govAcct, "", types.ErrNoProposalHandlerExists},
	}

	for i, tc := range testCases {
		prop, err := v1.NewLegacyContent(tc.content, tc.authority)
		suite.Require().NoError(err)
		_, err = suite.govKeeper.SubmitProposal(suite.ctx, []sdk.Msg{prop}, tc.metadata, "title", "", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
		suite.Require().True(errors.Is(tc.expectedErr, err), "tc #%d; got: %v, expected: %v", i, err, tc.expectedErr)
	}
}

func (suite *KeeperTestSuite) TestGetProposalsFiltered() {
	proposalID := uint64(1)

	addr1 := sdk.AccAddress("foo_________________")

	testCases := []struct {
		params             v1.QueryProposalsParams
		expectedNumResults int
	}{
		{v1.NewQueryProposalsParams(1, 50, v1.StatusNil, nil, nil), 50},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusDepositPeriod, nil, nil), 50},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusVotingPeriod, nil, nil), 50},
		{v1.NewQueryProposalsParams(1, 25, v1.StatusNil, nil, nil), 25},
		{v1.NewQueryProposalsParams(2, 25, v1.StatusNil, nil, nil), 25},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusRejected, nil, nil), 0},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusNil, addr1, nil), 50},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusNil, nil, addr1), 50},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusNil, addr1, addr1), 50},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusDepositPeriod, addr1, addr1), 25},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusDepositPeriod, nil, nil), 50},
		{v1.NewQueryProposalsParams(1, 50, v1.StatusVotingPeriod, nil, nil), 50},
	}

	for i, tc := range testCases {
		suite.Run(fmt.Sprintf("Test Case %d", i), func() {
			status := []v1.ProposalStatus{v1.StatusDepositPeriod, v1.StatusVotingPeriod}
			for _, s := range status {
				for i := 0; i < 50; i++ {
					p, err := v1.NewProposal(TestProposal, proposalID, time.Now(), time.Now(), "", "title", "summary", sdk.AccAddress("cosmos1ghekyjucln7y67ntx7cf27m9dpuxxemn4c8g4r"))
					suite.Require().NoError(err)

					p.Status = s

					if i%2 == 0 {
						d := v1.NewDeposit(proposalID, addr1, nil)
						v := v1.NewVote(proposalID, addr1, v1.NewNonSplitVoteOption(v1.OptionYes), "")
						suite.govKeeper.SetDeposit(suite.ctx, d)
						suite.govKeeper.SetVote(suite.ctx, v)
					}

					suite.govKeeper.SetProposal(suite.ctx, p)
					proposalID++
				}
			}

			proposals := suite.govKeeper.GetProposalsFiltered(suite.ctx, tc.params)
			suite.Require().Len(proposals, tc.expectedNumResults)

			for _, p := range proposals {
				if v1.ValidProposalStatus(tc.params.ProposalStatus) {
					suite.Require().Equal(tc.params.ProposalStatus, p.Status)
				}
			}
		})
	}
}

func TestMigrateProposalMessages(t *testing.T) {
	content := v1beta1.NewTextProposal("Test", "description")
	contentMsg, err := v1.NewLegacyContent(content, sdk.AccAddress("test1").String())
	require.NoError(t, err)
	content, err = v1.LegacyContentFromMessage(contentMsg)
	require.NoError(t, err)
	require.Equal(t, "Test", content.GetTitle())
	require.Equal(t, "description", content.GetDescription())
}

func TestProposalKinds(t *testing.T) {
	tests := []struct {
		name          string
		proposal      v1.Proposal
		expectedKinds v1.ProposalKinds
	}{
		{
			name:          "no message",
			proposal:      v1.Proposal{},
			expectedKinds: v1.ProposalKindAny,
		},
		{
			name: "Send message",
			proposal: v1.Proposal{
				Messages: setMsgs(t, []sdk.Msg{&banktypes.MsgSend{}}),
			},
			expectedKinds: v1.ProposalKindAny,
		},
		{
			name: "Law message",
			proposal: v1.Proposal{
				Messages: setMsgs(t, []sdk.Msg{&v1.MsgProposeLaw{}}),
			},
			expectedKinds: v1.ProposalKindLaw,
		},
		{
			name: "Law+Send messages",
			proposal: v1.Proposal{
				Messages: setMsgs(t, []sdk.Msg{
					&v1.MsgProposeLaw{},
					&banktypes.MsgSend{},
				}),
			},
			expectedKinds: v1.ProposalKindLaw | v1.ProposalKindAny,
		},
		{
			name: "ConstitutionAmendment message",
			proposal: v1.Proposal{
				Messages: setMsgs(t, []sdk.Msg{&v1.MsgProposeConstitutionAmendment{}}),
			},
			expectedKinds: v1.ProposalKindConstitutionAmendment,
		},
		{
			name: "ConstitutionAmendment+Law messages",
			proposal: v1.Proposal{
				Messages: setMsgs(t, []sdk.Msg{
					&v1.MsgProposeConstitutionAmendment{},
					&v1.MsgProposeLaw{},
				}),
			},
			expectedKinds: v1.ProposalKindConstitutionAmendment | v1.ProposalKindLaw,
		},
		{
			name: "Law+ConstitutionAmendment messages",
			proposal: v1.Proposal{
				Messages: setMsgs(t, []sdk.Msg{
					&v1.MsgProposeLaw{},
					&v1.MsgProposeConstitutionAmendment{},
				}),
			},
			expectedKinds: v1.ProposalKindConstitutionAmendment | v1.ProposalKindLaw,
		},
		{
			name: "Law+ConstitutionAmendment+Send messages",
			proposal: v1.Proposal{
				Messages: setMsgs(t, []sdk.Msg{
					&v1.MsgProposeLaw{},
					&v1.MsgProposeConstitutionAmendment{},
					&banktypes.MsgSend{},
				}),
			},
			expectedKinds: v1.ProposalKindConstitutionAmendment | v1.ProposalKindLaw | v1.ProposalKindAny,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, _, _, _ := setupGovKeeper(t)
			assert := assert.New(t)

			kinds := k.ProposalKinds(tt.proposal)

			assert.Equal(tt.expectedKinds, kinds)
		})
	}
}
