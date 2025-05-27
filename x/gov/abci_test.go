package gov_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov"
	"github.com/atomone-hub/atomone/x/gov/keeper"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func TestTickExpiredDepositPeriod(t *testing.T) {
	suite := createTestSuite(t)
	app := suite.App
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	addrs := simtestutil.AddTestAddrs(suite.BankKeeper, suite.StakingKeeper, ctx, 10, valTokens)

	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})

	govMsgSvr := keeper.NewMsgServerImpl(suite.GovKeeper)

	inactiveQueue := suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newProposalMsg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{mkTestLegacyContent(t)},
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 100000)},
		addrs[0].String(),
		"",
		"Proposal",
		"description of proposal",
	)
	require.NoError(t, err)

	res, err := govMsgSvr.SubmitProposal(sdk.WrapSDKContext(ctx), newProposalMsg)
	require.NoError(t, err)
	require.NotNil(t, res)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(*suite.GovKeeper.GetParams(ctx).MaxDepositPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	gov.EndBlocker(ctx, suite.GovKeeper)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()
}

func TestTickMultipleExpiredDepositPeriod(t *testing.T) {
	suite := createTestSuite(t)
	app := suite.App
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	addrs := simtestutil.AddTestAddrs(suite.BankKeeper, suite.StakingKeeper, ctx, 10, valTokens)

	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})

	govMsgSvr := keeper.NewMsgServerImpl(suite.GovKeeper)

	inactiveQueue := suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newProposalMsg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{mkTestLegacyContent(t)},
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 100000)},
		addrs[0].String(),
		"",
		"Proposal",
		"description of proposal",
	)
	require.NoError(t, err)

	res, err := govMsgSvr.SubmitProposal(sdk.WrapSDKContext(ctx), newProposalMsg)
	require.NoError(t, err)
	require.NotNil(t, res)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(2) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newProposalMsg2, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{mkTestLegacyContent(t)},
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 100000)},
		addrs[0].String(),
		"",
		"Proposal",
		"description of proposal",
	)
	require.NoError(t, err)

	res, err = govMsgSvr.SubmitProposal(sdk.WrapSDKContext(ctx), newProposalMsg2)
	require.NoError(t, err)
	require.NotNil(t, res)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(*suite.GovKeeper.GetParams(ctx).MaxDepositPeriod).Add(time.Duration(-1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	gov.EndBlocker(ctx, suite.GovKeeper)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(5) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	gov.EndBlocker(ctx, suite.GovKeeper)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()
}

func TestTickPassedDepositPeriod(t *testing.T) {
	suite := createTestSuite(t)
	app := suite.App
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	addrs := simtestutil.AddTestAddrs(suite.BankKeeper, suite.StakingKeeper, ctx, 10, valTokens)

	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})

	govMsgSvr := keeper.NewMsgServerImpl(suite.GovKeeper)

	inactiveQueue := suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()
	activeQueue := suite.GovKeeper.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()

	newProposalMsg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{mkTestLegacyContent(t)},
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 100000)},
		addrs[0].String(),
		"",
		"Proposal",
		"description of proposal",
	)
	require.NoError(t, err)

	res, err := govMsgSvr.SubmitProposal(sdk.WrapSDKContext(ctx), newProposalMsg)
	require.NoError(t, err)
	require.NotNil(t, res)

	proposalID := res.ProposalId

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	newDepositMsg := v1.NewMsgDeposit(addrs[1], proposalID, sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 100000)})

	res1, err := govMsgSvr.Deposit(sdk.WrapSDKContext(ctx), newDepositMsg)
	require.NoError(t, err)
	require.NotNil(t, res1)

	activeQueue = suite.GovKeeper.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()
}

func TestTickPassedVotingPeriod(t *testing.T) {
	suite := createTestSuite(t)
	app := suite.App
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	addrs := simtestutil.AddTestAddrs(suite.BankKeeper, suite.StakingKeeper, ctx, 10, valTokens)

	SortAddresses(addrs)

	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})

	govMsgSvr := keeper.NewMsgServerImpl(suite.GovKeeper)

	inactiveQueue := suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()
	activeQueue := suite.GovKeeper.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()

	proposalCoins := sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, suite.StakingKeeper.TokensFromConsensusPower(ctx, 5))}
	newProposalMsg, err := v1.NewMsgSubmitProposal([]sdk.Msg{mkTestLegacyContent(t)}, proposalCoins, addrs[0].String(), "", "Proposal", "description of proposal")
	require.NoError(t, err)

	wrapCtx := sdk.WrapSDKContext(ctx)

	res, err := govMsgSvr.SubmitProposal(wrapCtx, newProposalMsg)
	require.NoError(t, err)
	require.NotNil(t, res)

	proposalID := res.ProposalId

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	newDepositMsg := v1.NewMsgDeposit(addrs[1], proposalID, proposalCoins)

	res1, err := govMsgSvr.Deposit(wrapCtx, newDepositMsg)
	require.NoError(t, err)
	require.NotNil(t, res1)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(*suite.GovKeeper.GetParams(ctx).MaxDepositPeriod).Add(*suite.GovKeeper.GetParams(ctx).VotingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	inactiveQueue = suite.GovKeeper.InactiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, inactiveQueue.Valid())
	inactiveQueue.Close()

	activeQueue = suite.GovKeeper.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.True(t, activeQueue.Valid())

	activeProposalID := types.GetProposalIDFromBytes(activeQueue.Value())
	proposal, ok := suite.GovKeeper.GetProposal(ctx, activeProposalID)
	require.True(t, ok)
	require.Equal(t, v1.StatusVotingPeriod, proposal.Status)

	activeQueue.Close()

	gov.EndBlocker(ctx, suite.GovKeeper)

	activeQueue = suite.GovKeeper.ActiveProposalQueueIterator(ctx, ctx.BlockHeader().Time)
	require.False(t, activeQueue.Valid())
	activeQueue.Close()
}

func TestProposalPassedEndblocker(t *testing.T) {
	suite := createTestSuite(t)
	app := suite.App
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	addrs := simtestutil.AddTestAddrs(suite.BankKeeper, suite.StakingKeeper, ctx, 10, valTokens)

	SortAddresses(addrs)

	govMsgSvr := keeper.NewMsgServerImpl(suite.GovKeeper)
	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(suite.StakingKeeper)

	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})

	valAddr := sdk.ValAddress(addrs[0])

	createValidators(t, stakingMsgSvr, ctx, []sdk.ValAddress{valAddr}, []int64{10})
	staking.EndBlocker(ctx, suite.StakingKeeper)

	macc := suite.GovKeeper.GetGovernanceAccount(ctx)
	require.NotNil(t, macc)
	initialModuleAccCoins := suite.BankKeeper.GetAllBalances(ctx, macc.GetAddress())

	proposal, err := suite.GovKeeper.SubmitProposal(ctx, []sdk.Msg{mkTestLegacyContent(t)}, "", "title", "summary", addrs[0])
	require.NoError(t, err)

	proposalCoins := sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, suite.StakingKeeper.TokensFromConsensusPower(ctx, 10))}
	newDepositMsg := v1.NewMsgDeposit(addrs[0], proposal.Id, proposalCoins)

	res, err := govMsgSvr.Deposit(sdk.WrapSDKContext(ctx), newDepositMsg)
	require.NoError(t, err)
	require.NotNil(t, res)

	macc = suite.GovKeeper.GetGovernanceAccount(ctx)
	require.NotNil(t, macc)
	moduleAccCoins := suite.BankKeeper.GetAllBalances(ctx, macc.GetAddress())

	deposits := initialModuleAccCoins.Add(proposal.TotalDeposit...).Add(proposalCoins...)
	require.True(t, moduleAccCoins.IsEqual(deposits))

	err = suite.GovKeeper.AddVote(ctx, proposal.Id, addrs[0], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	require.NoError(t, err)

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(*suite.GovKeeper.GetParams(ctx).MaxDepositPeriod).Add(*suite.GovKeeper.GetParams(ctx).VotingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	gov.EndBlocker(ctx, suite.GovKeeper)

	macc = suite.GovKeeper.GetGovernanceAccount(ctx)
	require.NotNil(t, macc)
	require.True(t, suite.BankKeeper.GetAllBalances(ctx, macc.GetAddress()).IsEqual(initialModuleAccCoins))
}

func TestEndBlockerProposalHandlerFailed(t *testing.T) {
	suite := createTestSuite(t)
	app := suite.App
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	addrs := simtestutil.AddTestAddrs(suite.BankKeeper, suite.StakingKeeper, ctx, 1, valTokens)

	SortAddresses(addrs)

	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(suite.StakingKeeper)
	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})

	valAddr := sdk.ValAddress(addrs[0])

	createValidators(t, stakingMsgSvr, ctx, []sdk.ValAddress{valAddr}, []int64{10})
	staking.EndBlocker(ctx, suite.StakingKeeper)

	msg := banktypes.NewMsgSend(authtypes.NewModuleAddress(types.ModuleName), addrs[0], sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100000))))
	proposal, err := suite.GovKeeper.SubmitProposal(ctx, []sdk.Msg{msg}, "", "Bank Msg Send", "send message", addrs[0])
	require.NoError(t, err)

	proposalCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, suite.StakingKeeper.TokensFromConsensusPower(ctx, 10)))
	newDepositMsg := v1.NewMsgDeposit(addrs[0], proposal.Id, proposalCoins)

	govMsgSvr := keeper.NewMsgServerImpl(suite.GovKeeper)
	res, err := govMsgSvr.Deposit(sdk.WrapSDKContext(ctx), newDepositMsg)
	require.NoError(t, err)
	require.NotNil(t, res)

	err = suite.GovKeeper.AddVote(ctx, proposal.Id, addrs[0], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	require.NoError(t, err)

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(*suite.GovKeeper.GetParams(ctx).MaxDepositPeriod).Add(*suite.GovKeeper.GetParams(ctx).VotingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	// validate that the proposal fails/has been rejected
	gov.EndBlocker(ctx, suite.GovKeeper)

	proposal, ok := suite.GovKeeper.GetProposal(ctx, proposal.Id)
	require.True(t, ok)
	require.Equal(t, v1.StatusFailed, proposal.Status)
}

func TestEndBlockerQuorumCheck(t *testing.T) {
	params := v1.DefaultParams()
	params.QuorumCheckCount = 10 // enable quorum check
	quorumTimeout := *params.VotingPeriod - time.Hour*8
	params.QuorumTimeout = &quorumTimeout
	oneHour := time.Hour
	testcases := []struct {
		name string
		// the value of the MaxVotingPeriodExtension param
		maxVotingPeriodExtension *time.Duration
		// the duration after which the proposal reaches quorum
		reachQuorumAfter time.Duration
		// the expected status of the proposal after the original voting period has elapsed
		expectedStatusAfterVotingPeriod v1.ProposalStatus
		// the expected final voting period after the original period has elapsed
		// the value would be modified if voting period is extended due to quorum being reached
		// after the quorum timeout
		expectedVotingPeriod time.Duration
	}{
		{
			name:                            "reach quorum before timeout: voting period not extended",
			maxVotingPeriodExtension:        params.MaxVotingPeriodExtension,
			reachQuorumAfter:                quorumTimeout - time.Hour,
			expectedStatusAfterVotingPeriod: v1.StatusPassed,
			expectedVotingPeriod:            *params.VotingPeriod,
		},
		{
			name:                            "reach quorum exactly at timeout: voting period not extended",
			maxVotingPeriodExtension:        params.MaxVotingPeriodExtension,
			reachQuorumAfter:                quorumTimeout,
			expectedStatusAfterVotingPeriod: v1.StatusPassed,
			expectedVotingPeriod:            *params.VotingPeriod,
		},
		{
			name:                            "quorum never reached: voting period not extended",
			maxVotingPeriodExtension:        params.MaxVotingPeriodExtension,
			reachQuorumAfter:                0,
			expectedStatusAfterVotingPeriod: v1.StatusRejected,
			expectedVotingPeriod:            *params.VotingPeriod,
		},
		{
			name:                            "reach quorum after timeout, voting period extended",
			maxVotingPeriodExtension:        params.MaxVotingPeriodExtension,
			reachQuorumAfter:                quorumTimeout + time.Hour,
			expectedStatusAfterVotingPeriod: v1.StatusVotingPeriod,
			expectedVotingPeriod: *params.VotingPeriod + *params.MaxVotingPeriodExtension -
				(*params.VotingPeriod - quorumTimeout - time.Hour),
		},
		{
			name:                            "reach quorum exactly at voting period, voting period extended",
			maxVotingPeriodExtension:        params.MaxVotingPeriodExtension,
			reachQuorumAfter:                *params.VotingPeriod,
			expectedStatusAfterVotingPeriod: v1.StatusVotingPeriod,
			expectedVotingPeriod:            *params.VotingPeriod + *params.MaxVotingPeriodExtension,
		},
		{
			name:                            "reach quorum after timeout but voting period extension too short, voting period not extended",
			maxVotingPeriodExtension:        &oneHour,
			reachQuorumAfter:                quorumTimeout + time.Hour,
			expectedStatusAfterVotingPeriod: v1.StatusPassed,
			expectedVotingPeriod:            *params.VotingPeriod,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			suite := createTestSuite(t)
			app := suite.App
			ctx := app.BaseApp.NewContext(false, tmproto.Header{})
			params.MaxVotingPeriodExtension = tc.maxVotingPeriodExtension
			err := suite.GovKeeper.SetParams(ctx, params)
			require.NoError(t, err)
			addrs := simtestutil.AddTestAddrs(suite.BankKeeper, suite.StakingKeeper, ctx, 10, valTokens)
			// _, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
			// Height: app.LastBlockHeight() + 1,
			// Hash:   app.LastCommitID().Hash,
			// })
			// require.NoError(t, err)
			// Create a validator
			valAddr := sdk.ValAddress(addrs[0])
			stakingMsgSvr := stakingkeeper.NewMsgServerImpl(suite.StakingKeeper)
			createValidators(t, stakingMsgSvr, ctx, []sdk.ValAddress{valAddr}, []int64{10})
			staking.EndBlocker(ctx, suite.StakingKeeper)
			// Create a proposal
			govMsgSvr := keeper.NewMsgServerImpl(suite.GovKeeper)
			deposit := v1.DefaultMinDepositTokens.ToLegacyDec().Mul(v1.DefaultMinDepositRatio)
			newProposalMsg, err := v1.NewMsgSubmitProposal(
				[]sdk.Msg{mkTestLegacyContent(t)},
				sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, deposit.RoundInt())},
				addrs[0].String(), "", "Proposal", "description of proposal",
			)
			require.NoError(t, err)
			res, err := govMsgSvr.SubmitProposal(ctx, newProposalMsg)
			require.NoError(t, err)
			require.NotNil(t, res)
			// Activate proposal
			newDepositMsg := v1.NewMsgDeposit(addrs[1], res.ProposalId, params.MinDeposit)
			res1, err := govMsgSvr.Deposit(ctx, newDepositMsg)
			require.NoError(t, err)
			require.NotNil(t, res1)
			prop, ok := suite.GovKeeper.GetProposal(ctx, res.ProposalId)
			require.True(t, ok, "prop not found")

			// Call EndBlock until the initial voting period has elapsed
			// Tick is one hour
			var (
				startTime = ctx.BlockTime()
				tick      = time.Hour
			)
			for ctx.BlockTime().Sub(startTime) < *params.VotingPeriod {
				// Forward in time
				newTime := ctx.BlockTime().Add(tick)
				ctx = ctx.WithBlockTime(newTime)
				if tc.reachQuorumAfter != 0 && newTime.Sub(startTime) >= tc.reachQuorumAfter {
					// Set quorum as reached
					err := suite.GovKeeper.AddVote(ctx, prop.Id, addrs[0], v1.NewNonSplitVoteOption(v1.OptionYes), "")
					require.NoError(t, err)
					// Don't go there again
					tc.reachQuorumAfter = 0
				}

				gov.EndBlocker(ctx, suite.GovKeeper)

			}

			// Assertions
			prop, ok = suite.GovKeeper.GetProposal(ctx, prop.Id) // Get fresh prop
			if assert.True(t, ok, "prop not found") {
				assert.Equal(t, tc.expectedStatusAfterVotingPeriod.String(), prop.Status.String())
				assert.Equal(t, tc.expectedVotingPeriod, prop.VotingEndTime.Sub(*prop.VotingStartTime))
				assert.False(t, suite.GovKeeper.QuorumCheckQueueIterator(ctx, *prop.VotingStartTime).Valid(), "quorum check queue invalid")
			}
		})
	}
}

func createValidators(t *testing.T, stakingMsgSvr stakingtypes.MsgServer, ctx sdk.Context, addrs []sdk.ValAddress, powerAmt []int64) {
	require.True(t, len(addrs) <= len(pubkeys), "Not enough pubkeys specified at top of file.")

	for i := 0; i < len(addrs); i++ {
		valTokens := sdk.TokensFromConsensusPower(powerAmt[i], sdk.DefaultPowerReduction)
		valCreateMsg, err := stakingtypes.NewMsgCreateValidator(
			addrs[i], pubkeys[i], sdk.NewCoin(sdk.DefaultBondDenom, valTokens),
			TestDescription, TestCommissionRates, math.OneInt(),
		)
		require.NoError(t, err)
		res, err := stakingMsgSvr.CreateValidator(sdk.WrapSDKContext(ctx), valCreateMsg)
		require.NoError(t, err)
		require.NotNil(t, res)
	}
}
