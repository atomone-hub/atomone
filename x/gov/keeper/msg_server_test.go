package keeper_test

import (
	"fmt"
	"strings"
	"time"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

func (suite *KeeperTestSuite) TestSubmitProposalReq() {
	suite.reset()
	govAcct := suite.govKeeper.GetGovernanceAccount(suite.ctx).GetAddress()
	addrs := suite.addrs
	proposer := addrs[0]

	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))
	initialDeposit := coins
	minDeposit := suite.govKeeper.GetParams(suite.ctx).MinDeposit
	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	cases := map[string]struct {
		preRun    func() (*v1.MsgSubmitProposal, error)
		expErr    bool
		expErrMsg string
	}{
		"metadata too long": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					initialDeposit,
					proposer.String(),
					strings.Repeat("1", 300),
					"Proposal",
					"description of proposal",
				)
			},
			expErr:    true,
			expErrMsg: "metadata too long",
		},
		"many signers": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{testdata.NewTestMsg(govAcct, addrs[0])},
					initialDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr:    true,
			expErrMsg: "expected gov account as only signer for proposal message",
		},
		"signer isn't gov account": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{testdata.NewTestMsg(addrs[0])},
					initialDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr:    true,
			expErrMsg: "expected gov account as only signer for proposal message",
		},
		"invalid msg handler": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{testdata.NewTestMsg(govAcct)},
					initialDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr:    true,
			expErrMsg: "proposal message not recognized by router",
		},
		"all good": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					initialDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr: false,
		},
		"all good with min deposit": {
			preRun: func() (*v1.MsgSubmitProposal, error) {
				return v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					minDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
			},
			expErr: false,
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			msg, err := tc.preRun()
			suite.Require().NoError(err)
			res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestVoteReq() {
	suite.reset()
	govAcct := suite.govKeeper.GetGovernanceAccount(suite.ctx).GetAddress()
	addrs := suite.addrs
	proposer := addrs[0]

	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))
	minDeposit := suite.govKeeper.GetParams(suite.ctx).MinDeposit
	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		minDeposit,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	suite.Require().NoError(err)

	res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res.ProposalId)
	proposalId := res.ProposalId

	cases := map[string]struct {
		preRun    func() uint64
		expErr    bool
		expErrMsg string
		option    v1.VoteOption
		metadata  string
		voter     sdk.AccAddress
	}{
		"vote on inactive proposal": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				suite.Require().NoError(err)

				res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
				return res.ProposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     proposer,
			metadata:  "",
			expErr:    true,
			expErrMsg: "inactive proposal",
		},
		"metadata too long": {
			preRun: func() uint64 {
				return proposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     proposer,
			metadata:  strings.Repeat("a", 300),
			expErr:    true,
			expErrMsg: "metadata too long",
		},
		"voter error": {
			preRun: func() uint64 {
				return proposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     sdk.AccAddress(strings.Repeat("a", 300)),
			metadata:  "",
			expErr:    true,
			expErrMsg: "address max length is 255",
		},
		"all good": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					minDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				suite.Require().NoError(err)

				res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
				return res.ProposalId
			},
			option:   v1.VoteOption_VOTE_OPTION_YES,
			voter:    proposer,
			metadata: "",
			expErr:   false,
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			pId := tc.preRun()
			voteReq := v1.NewMsgVote(tc.voter, pId, tc.option, tc.metadata)
			_, err := suite.msgSrvr.Vote(suite.ctx, voteReq)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestVoteWeightedReq() {
	suite.reset()
	govAcct := suite.govKeeper.GetGovernanceAccount(suite.ctx).GetAddress()
	addrs := suite.addrs
	proposer := addrs[0]

	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))
	minDeposit := suite.govKeeper.GetParams(suite.ctx).MinDeposit
	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		minDeposit,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	suite.Require().NoError(err)

	res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res.ProposalId)
	proposalId := res.ProposalId

	cases := map[string]struct {
		preRun    func() uint64
		vote      *v1.MsgVote
		expErr    bool
		expErrMsg string
		option    v1.VoteOption
		metadata  string
		voter     sdk.AccAddress
	}{
		"vote on inactive proposal": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				suite.Require().NoError(err)

				res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
				return res.ProposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     proposer,
			metadata:  "",
			expErr:    true,
			expErrMsg: "inactive proposal",
		},
		"metadata too long": {
			preRun: func() uint64 {
				return proposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     proposer,
			metadata:  strings.Repeat("a", 300),
			expErr:    true,
			expErrMsg: "metadata too long",
		},
		"voter error": {
			preRun: func() uint64 {
				return proposalId
			},
			option:    v1.VoteOption_VOTE_OPTION_YES,
			voter:     sdk.AccAddress(strings.Repeat("a", 300)),
			metadata:  "",
			expErr:    true,
			expErrMsg: "address max length is 255",
		},
		"all good": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					minDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				suite.Require().NoError(err)

				res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
				return res.ProposalId
			},
			option:   v1.VoteOption_VOTE_OPTION_YES,
			voter:    proposer,
			metadata: "",
			expErr:   false,
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			pId := tc.preRun()
			voteReq := v1.NewMsgVoteWeighted(tc.voter, pId, v1.NewNonSplitVoteOption(tc.option), tc.metadata)
			_, err := suite.msgSrvr.VoteWeighted(suite.ctx, voteReq)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDepositReq() {
	govAcct := suite.govKeeper.GetGovernanceAccount(suite.ctx).GetAddress()
	addrs := suite.addrs
	proposer := addrs[0]

	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))
	minDeposit := sdk.Coins(suite.govKeeper.GetParams(suite.ctx).MinDeposit)
	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		coins,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	suite.Require().NoError(err)

	res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res.ProposalId)
	pId := res.ProposalId

	cases := map[string]struct {
		preRun     func() uint64
		expErr     bool
		proposalId uint64
		depositor  sdk.AccAddress
		deposit    sdk.Coins
		options    v1.WeightedVoteOptions
	}{
		"wrong proposal id": {
			preRun: func() uint64 {
				return 0
			},
			depositor: proposer,
			deposit:   coins,
			expErr:    true,
			options:   v1.NewNonSplitVoteOption(v1.OptionYes),
		},
		"all good": {
			preRun: func() uint64 {
				return pId
			},
			depositor: proposer,
			deposit:   minDeposit,
			expErr:    false,
			options:   v1.NewNonSplitVoteOption(v1.OptionYes),
		},
		"invalid deposited coin ": {
			preRun: func() uint64 {
				return pId
			},
			depositor: proposer,
			deposit:   minDeposit.Add(sdk.NewCoin("ibc/badcoin", sdk.NewInt(1000))), expErr: true,
			options: v1.NewNonSplitVoteOption(v1.OptionYes),
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			proposalId := tc.preRun()
			depositReq := v1.NewMsgDeposit(tc.depositor, proposalId, tc.deposit)
			_, err := suite.msgSrvr.Deposit(suite.ctx, depositReq)
			if tc.expErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

// legacy msg server tests
func (suite *KeeperTestSuite) TestLegacyMsgSubmitProposal() {
	addrs := suite.addrs
	proposer := addrs[0]

	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))
	initialDeposit := coins
	minDeposit := suite.govKeeper.GetParams(suite.ctx).MinDeposit

	cases := map[string]struct {
		preRun func() (*v1beta1.MsgSubmitProposal, error)
		expErr bool
	}{
		"all good": {
			preRun: func() (*v1beta1.MsgSubmitProposal, error) {
				return v1beta1.NewMsgSubmitProposal(
					v1beta1.NewTextProposal("test", "I am test"),
					initialDeposit,
					proposer,
				)
			},
			expErr: false,
		},
		"all good with min deposit": {
			preRun: func() (*v1beta1.MsgSubmitProposal, error) {
				return v1beta1.NewMsgSubmitProposal(
					v1beta1.NewTextProposal("test", "I am test"),
					minDeposit,
					proposer,
				)
			},
			expErr: false,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			msg, err := c.preRun()
			suite.Require().NoError(err)
			res, err := suite.legacyMsgSrvr.SubmitProposal(suite.ctx, msg)
			if c.expErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestLegacyMsgVote() {
	govAcct := suite.govKeeper.GetGovernanceAccount(suite.ctx).GetAddress()
	addrs := suite.addrs
	proposer := addrs[0]

	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))
	minDeposit := suite.govKeeper.GetParams(suite.ctx).MinDeposit
	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		minDeposit,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	suite.Require().NoError(err)

	res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res.ProposalId)
	proposalId := res.ProposalId

	cases := map[string]struct {
		preRun    func() uint64
		expErr    bool
		expErrMsg string
		option    v1beta1.VoteOption
		metadata  string
		voter     sdk.AccAddress
	}{
		"vote on inactive proposal": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				suite.Require().NoError(err)

				res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
				return res.ProposalId
			},
			option:    v1beta1.OptionYes,
			voter:     proposer,
			metadata:  "",
			expErr:    true,
			expErrMsg: "inactive proposal",
		},
		"voter error": {
			preRun: func() uint64 {
				return proposalId
			},
			option:    v1beta1.OptionYes,
			voter:     sdk.AccAddress(strings.Repeat("a", 300)),
			metadata:  "",
			expErr:    true,
			expErrMsg: "address max length is 255",
		},
		"all good": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					minDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				suite.Require().NoError(err)

				res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
				return res.ProposalId
			},
			option:   v1beta1.OptionYes,
			voter:    proposer,
			metadata: "",
			expErr:   false,
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			pId := tc.preRun()
			voteReq := v1beta1.NewMsgVote(tc.voter, pId, tc.option)
			_, err := suite.legacyMsgSrvr.Vote(suite.ctx, voteReq)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestLegacyVoteWeighted() {
	suite.reset()
	govAcct := suite.govKeeper.GetGovernanceAccount(suite.ctx).GetAddress()
	addrs := suite.addrs
	proposer := addrs[0]

	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))
	minDeposit := suite.govKeeper.GetParams(suite.ctx).MinDeposit
	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		minDeposit,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	suite.Require().NoError(err)

	res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res.ProposalId)
	proposalId := res.ProposalId

	cases := map[string]struct {
		preRun    func() uint64
		vote      *v1beta1.MsgVote
		expErr    bool
		expErrMsg string
		option    v1beta1.VoteOption
		metadata  string
		voter     sdk.AccAddress
	}{
		"vote on inactive proposal": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					coins,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				suite.Require().NoError(err)

				res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
				return res.ProposalId
			},
			option:    v1beta1.OptionYes,
			voter:     proposer,
			metadata:  "",
			expErr:    true,
			expErrMsg: "inactive proposal",
		},
		"voter error": {
			preRun: func() uint64 {
				return proposalId
			},
			option:    v1beta1.OptionYes,
			voter:     sdk.AccAddress(strings.Repeat("a", 300)),
			metadata:  "",
			expErr:    true,
			expErrMsg: "address max length is 255",
		},
		"all good": {
			preRun: func() uint64 {
				msg, err := v1.NewMsgSubmitProposal(
					[]sdk.Msg{bankMsg},
					minDeposit,
					proposer.String(),
					"",
					"Proposal",
					"description of proposal",
				)
				suite.Require().NoError(err)

				res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(res.ProposalId)
				return res.ProposalId
			},
			option:   v1beta1.OptionYes,
			voter:    proposer,
			metadata: "",
			expErr:   false,
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			pId := tc.preRun()
			voteReq := v1beta1.NewMsgVoteWeighted(tc.voter, pId, v1beta1.NewNonSplitVoteOption(v1beta1.VoteOption(tc.option)))
			_, err := suite.legacyMsgSrvr.VoteWeighted(suite.ctx, voteReq)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestLegacyMsgDeposit() {
	govAcct := suite.govKeeper.GetGovernanceAccount(suite.ctx).GetAddress()
	addrs := suite.addrs
	proposer := addrs[0]

	coins := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))
	minDeposit := suite.govKeeper.GetParams(suite.ctx).MinDeposit
	bankMsg := &banktypes.MsgSend{
		FromAddress: govAcct.String(),
		ToAddress:   proposer.String(),
		Amount:      coins,
	}

	msg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{bankMsg},
		coins,
		proposer.String(),
		"",
		"Proposal",
		"description of proposal",
	)
	suite.Require().NoError(err)

	res, err := suite.msgSrvr.SubmitProposal(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res.ProposalId)
	pId := res.ProposalId

	cases := map[string]struct {
		preRun     func() uint64
		expErr     bool
		proposalId uint64
		depositor  sdk.AccAddress
		deposit    sdk.Coins
		options    v1beta1.WeightedVoteOptions
	}{
		"wrong proposal id": {
			preRun: func() uint64 {
				return 0
			},
			depositor: proposer,
			deposit:   coins,
			expErr:    true,
			options:   v1beta1.NewNonSplitVoteOption(v1beta1.OptionYes),
		},
		"all good": {
			preRun: func() uint64 {
				return pId
			},
			depositor: proposer,
			deposit:   minDeposit,
			expErr:    false,
			options:   v1beta1.NewNonSplitVoteOption(v1beta1.OptionYes),
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			proposalId := tc.preRun()
			depositReq := v1beta1.NewMsgDeposit(tc.depositor, proposalId, tc.deposit)
			_, err := suite.legacyMsgSrvr.Deposit(suite.ctx, depositReq)
			if tc.expErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgUpdateParams() {
	authority := suite.govKeeper.GetAuthority()
	minVotingPeriod, _ := time.ParseDuration(v1.MinVotingPeriod)
	params := v1.DefaultParams()
	testCases := []struct {
		name      string
		input     func() *v1.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid",
			input: func() *v1.MsgUpdateParams {
				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params,
				}
			},
			expErr: false,
		},
		{
			name: "invalid authority",
			input: func() *v1.MsgUpdateParams {
				return &v1.MsgUpdateParams{
					Authority: "authority",
					Params:    params,
				}
			},
			expErr:    true,
			expErrMsg: "invalid authority address",
		},
		{
			name: "invalid min deposit",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.MinDeposit = nil

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid minimum deposit",
		},
		{
			name: "negative deposit",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.MinDeposit = sdk.Coins{{
					Denom:  sdk.DefaultBondDenom,
					Amount: sdk.NewInt(-100),
				}}

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid minimum deposit",
		},
		{
			name: "invalid max deposit period",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.MaxDepositPeriod = nil

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "maximum deposit period must not be nil",
		},
		{
			name: "zero max deposit period",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				duration := time.Duration(0)
				params1.MaxDepositPeriod = &duration

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "maximum deposit period must be positive",
		},
		{
			name: "invalid quorum",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.Quorum = "abc"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid quorum string",
		},
		{
			name: "negative quorum",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.Quorum = "-0.1"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "quorum must be positive",
		},
		{
			name: "quorum > 1",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.Quorum = "2"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "quorum too large",
		},
		{
			name: "invalid threshold",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.Threshold = "abc"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "invalid threshold string",
		},
		{
			name: "negative threshold",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.Threshold = "-0.1"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "vote threshold must be positive",
		},
		{
			name: "threshold > 1",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.Threshold = "2"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "vote threshold too large",
		},
		{
			name: "negative constitution amendment quorum",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.ConstitutionAmendmentQuorum = "-0.1"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "constitution amendment quorum must be positive",
		},
		{
			name: "constitution amendments quorum > 1",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.ConstitutionAmendmentQuorum = "2"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "constitution amendment quorum too large",
		},
		{
			name: "negative constitution amendment threshold",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.ConstitutionAmendmentThreshold = "-0.1"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "constitution amendment threshold must be positive",
		},
		{
			name: "constitution amendments threshold > 1",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.ConstitutionAmendmentThreshold = "2"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "constitution amendment threshold too large",
		},
		{
			name: "negative law quorum",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.LawQuorum = "-0.1"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "law quorum must be positive",
		},
		{
			name: "law quorum > 1",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.LawQuorum = "2"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "law quorum too large",
		},
		{
			name: "negative law threshold",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.LawThreshold = "-0.1"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "law threshold must be positive",
		},
		{
			name: "law threshold > 1",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.LawThreshold = "2"

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "law threshold too large",
		},
		{
			name: "invalid voting period",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.VotingPeriod = nil

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "voting period must not be nil",
		},
		{
			name: "zero voting period",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				duration := time.Duration(0)
				params1.VotingPeriod = &duration

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: fmt.Sprintf("voting period must be at least %s: 0s", minVotingPeriod),
		},
		{
			name: "invalid quorum timeout",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.QuorumCheckCount = 1 // enable quorum check
				params1.QuorumTimeout = nil

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "quorum timeout must not be nil: 0",
		},
		{
			name: "negative quorum timeout",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.QuorumCheckCount = 1 // enable quorum check
				d := time.Duration(-1)
				params1.QuorumTimeout = &d

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "quorum timeout must be 0 or greater: -1ns",
		},
		{
			name: "quorum timeout exceeds voting period",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.QuorumCheckCount = 1 // enable quorum check
				d := *params.VotingPeriod + 1
				params1.QuorumTimeout = &d

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "quorum timeout 504h0m0.000000001s must be strictly less than the voting period 504h0m0s",
		},
		{
			name: "invalid max voting period extension",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.QuorumCheckCount = 1 // enable quorum check
				d := *params.VotingPeriod - time.Hour*2
				params1.QuorumTimeout = &d
				params1.MaxVotingPeriodExtension = nil

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "max voting period extension must not be nil: 0",
		},
		{
			name: "voting period extension below voting period - quorum timeout",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.QuorumCheckCount = 1 // enable quorum check
				d := *params.VotingPeriod - time.Hour*2
				params1.QuorumTimeout = &d
				d2 := *params1.VotingPeriod - *params1.QuorumTimeout - 1
				params1.MaxVotingPeriodExtension = &d2

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
			expErr:    true,
			expErrMsg: "max voting period extension 1h59m59.999999999s must be greater than or equal to the difference between the voting period 504h0m0s and the quorum timeout 502h0m0s",
		},
		{
			name: "valid with quorum check enabled",
			input: func() *v1.MsgUpdateParams {
				params1 := params
				params1.QuorumCheckCount = 1 // enable quorum check
				d := *params.VotingPeriod - time.Hour*2
				params1.QuorumTimeout = &d
				d2 := *params1.VotingPeriod - *params1.QuorumTimeout + time.Hour*24
				params1.MaxVotingPeriodExtension = &d2

				return &v1.MsgUpdateParams{
					Authority: authority,
					Params:    params1,
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg := tc.input()
			exec := func(updateParams *v1.MsgUpdateParams) error {
				if err := msg.ValidateBasic(); err != nil {
					return err
				}

				if _, err := suite.msgSrvr.UpdateParams(suite.ctx, updateParams); err != nil {
					return err
				}
				return nil
			}

			err := exec(msg)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSubmitProposal_InitialDeposit() {
	const meetsDepositValue = baseDepositTestAmount * baseDepositTestPercent / 100
	baseDepositRatioDec := sdk.NewDec(baseDepositTestPercent).Quo(sdk.NewDec(100))

	testcases := map[string]struct {
		minDeposit             sdk.Coins
		minInitialDepositRatio sdk.Dec
		initialDeposit         sdk.Coins
		accountBalance         sdk.Coins

		expectError bool
	}{
		"meets initial deposit, enough balance - success": {
			minDeposit:             sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseDepositTestAmount))),
			minInitialDepositRatio: baseDepositRatioDec,
			initialDeposit:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue))),
			accountBalance:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue))),
		},
		"does not meet initial deposit, enough balance - error": {
			minDeposit:             sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseDepositTestAmount))),
			minInitialDepositRatio: baseDepositRatioDec,
			initialDeposit:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue-1))),
			accountBalance:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue))),

			expectError: true,
		},
		"meets initial deposit, not enough balance - error": {
			minDeposit:             sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseDepositTestAmount))),
			minInitialDepositRatio: baseDepositRatioDec,
			initialDeposit:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue))),
			accountBalance:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue-1))),

			expectError: true,
		},
		"does not meet initial deposit and not enough balance - error": {
			minDeposit:             sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(baseDepositTestAmount))),
			minInitialDepositRatio: baseDepositRatioDec,
			initialDeposit:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue-1))),
			accountBalance:         sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(meetsDepositValue-1))),

			expectError: true,
		},
	}

	for name, tc := range testcases {
		suite.Run(name, func() {
			// Setup
			govKeeper, ctx := suite.govKeeper, suite.ctx
			address := simtestutil.AddTestAddrs(suite.bankKeeper, suite.stakingKeeper, ctx, 1, tc.accountBalance[0].Amount)[0]

			params := v1.DefaultParams()
			params.MinDeposit = tc.minDeposit
			params.MinInitialDepositRatio = tc.minInitialDepositRatio.String()
			govKeeper.SetParams(ctx, params)

			msg, err := v1.NewMsgSubmitProposal(TestProposal, tc.initialDeposit, address.String(), "test", "Proposal", "description of proposal")
			suite.Require().NoError(err)

			// System under test
			_, err = suite.msgSrvr.SubmitProposal(sdk.WrapSDKContext(ctx), msg)

			// Assertions
			if tc.expectError {
				suite.Require().Error(err)
				return
			}
			suite.Require().NoError(err)
		})
	}
}

func (suite *KeeperTestSuite) TestProposeConstitutionAmendment() {
	ctx := suite.ctx
	suite.govKeeper.SetConstitution(ctx, "Hello World")

	cases := map[string]struct {
		msg       *v1.MsgProposeConstitutionAmendment
		expErr    bool
		expErrMsg string
		expResult string
	}{
		"successful amendment": {
			msg: v1.NewMsgProposeConstitutionAmendment(
				suite.govKeeper.GetGovernanceAccount(ctx).GetAddress(),
				"@@ -1 +1 @@\n-Hello World\n+Hi  World",
			),
			expErr:    false,
			expResult: "Hi  World",
		},
		"failed amendment": {
			msg: v1.NewMsgProposeConstitutionAmendment(
				suite.govKeeper.GetGovernanceAccount(ctx).GetAddress(),
				"@@ -1 +1 @@\n-Hello  World\n+Hi  World",
			),
			expErr: true,
		},
		"invalid patch": {
			msg: v1.NewMsgProposeConstitutionAmendment(
				suite.govKeeper.GetGovernanceAccount(ctx).GetAddress(),
				"invalid patch",
			),
			expErr: true,
		},
		"invalid authority": {
			msg: v1.NewMsgProposeConstitutionAmendment(
				sdk.AccAddress("invalid"),
				"@@ -1 +1 @@\n-Hello  World\n+Hi  World",
			),
			expErr:    true,
			expErrMsg: govtypes.ErrInvalidSigner.Error(),
		},
	}

	for name, tc := range cases {
		suite.Run(name, func() {
			_, err := suite.msgSrvr.ProposeConstitutionAmendment(sdk.WrapSDKContext(ctx), tc.msg)
			if tc.expErr {
				suite.Require().Error(err)
				if tc.expErrMsg != "" {
					suite.Require().Contains(err.Error(), tc.expErrMsg)
				}
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expResult, suite.govKeeper.GetConstitution(ctx))
			}
		})
	}
}
