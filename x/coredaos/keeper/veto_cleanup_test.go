package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	atomoneapp "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/app/helpers"
	coredaoskeeper "github.com/atomone-hub/atomone/x/coredaos/keeper"
	"github.com/atomone-hub/atomone/x/coredaos/types"
)

// TestVetoProposalDeferredCleanupIsConstantCost is the regression test for the
// gas-exhaustion vulnerability: VetoProposal used to refund every deposit and
// delete every vote inline, under the metered transaction gas meter, so a
// proposal with enough participants could no longer be vetoed within
// block.max_gas. The fix makes the handler O(1) and defers the unbounded
// cleanup to the EndBlocker (which runs under the infinite block gas meter).
//
// The test proves three things against the real gov keeper:
//  1. the veto handler's gas cost does not scale with the number of depositors
//     or votes (it is essentially identical with 0 and with many participants);
//  2. the cleanup is deferred — deposits and votes still exist right after the
//     veto handler returns;
//  3. the EndBlocker performs the full cleanup — every deposit is refunded and
//     every vote deleted — and drains the queue.
func TestVetoProposalDeferredCleanupIsConstantCost(t *testing.T) {
	const (
		numDepositors = 1000
		numVotes      = 1000
	)

	oversightDAO := simtestutil.CreateRandomAccounts(1)[0]

	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})
	ms := coredaoskeeper.NewMsgServer(app.CoreDaosKeeper)

	params := types.DefaultParams()
	params.OversightDaoAddress = oversightDAO.String()
	require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, params))

	depositCoin := sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100_000))
	depositCoins := sdk.NewCoins(depositCoin)

	// vetoGas vetoes a proposal under a fresh metered gas meter and returns the
	// gas consumed by the handler alone.
	vetoGas := func(proposalID uint64) uint64 {
		mctx := ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000_000))
		_, err := ms.VetoProposal(mctx, &types.MsgVetoProposal{
			Vetoer:     oversightDAO.String(),
			ProposalId: proposalID,
		})
		require.NoError(t, err)
		return mctx.GasMeter().GasConsumed()
	}

	// Baseline: a proposal with no depositors and no votes.
	small := submitBankSendProposalReal(t, app, ctx, true)
	gasSmall := vetoGas(small.Id)

	// Large: a proposal with many real depositors and many real votes, all
	// created through the gov keeper so the state is reachable in production.
	large := submitBankSendProposalReal(t, app, ctx, true)

	depositors := simtestutil.CreateRandomAccounts(numDepositors)
	for _, d := range depositors {
		require.NoError(t, banktestutil.FundAccount(ctx, app.BankKeeper, d, depositCoins))
		_, err := app.GovKeeper.AddDeposit(ctx, large.Id, d, depositCoins, true)
		require.NoError(t, err)
	}

	voters := simtestutil.CreateRandomAccounts(numVotes)
	for _, v := range voters {
		err := app.GovKeeper.AddVote(ctx, large.Id, v, govv1.NewNonSplitVoteOption(govv1.OptionYes), "")
		require.NoError(t, err)
	}

	gasLarge := vetoGas(large.Id)

	t.Logf("veto handler gas: depositors=0 votes=0 -> %d; depositors=%d votes=%d -> %d",
		gasSmall, numDepositors, numVotes, gasLarge)

	// (1) The handler cost must not scale with participant count. The old inline
	// implementation consumed ~17,580 gas per depositor and ~1,345 per vote, so
	// the large proposal would have cost ~18.9M more than the baseline. Require
	// the marginal cost to stay negligible.
	require.Less(t, gasLarge, gasSmall+100_000,
		"veto handler gas must not scale with the number of depositors/votes")

	// (2) Cleanup is deferred: deposits and votes still present after the veto.
	require.Equal(t, numDepositors, countProposalDeposits(t, ctx, app, large.Id))
	require.Equal(t, numVotes, countProposalVotes(t, ctx, app, large.Id))
	enqueued, err := app.CoreDaosKeeper.VetoCleanupQueue.Has(ctx, large.Id)
	require.NoError(t, err)
	require.True(t, enqueued)

	// (3) The EndBlocker performs the full cleanup under its infinite gas meter.
	require.NoError(t, app.CoreDaosKeeper.EndBlocker(ctx))

	require.Equal(t, 0, countProposalDeposits(t, ctx, app, large.Id), "all deposits must be deleted")
	require.Equal(t, 0, countProposalVotes(t, ctx, app, large.Id), "all votes must be deleted")
	drained, err := app.CoreDaosKeeper.VetoCleanupQueue.Has(ctx, large.Id)
	require.NoError(t, err)
	require.False(t, drained, "cleanup queue must be drained")

	// Every depositor must have been refunded in full (default veto does not burn).
	for _, d := range depositors {
		bal := app.BankKeeper.GetBalance(ctx, d, sdk.DefaultBondDenom)
		require.Equal(t, depositCoin, bal, "depositor %s was not refunded", d)
	}
}

func countProposalDeposits(t *testing.T, ctx sdk.Context, app *atomoneapp.AtomOneApp, proposalID uint64) int {
	t.Helper()
	n := 0
	err := app.GovKeeper.IterateDeposits(ctx, proposalID, func(_ collections.Pair[uint64, sdk.AccAddress], _ govv1.Deposit) (bool, error) {
		n++
		return false, nil
	})
	require.NoError(t, err)
	return n
}

func countProposalVotes(t *testing.T, ctx sdk.Context, app *atomoneapp.AtomOneApp, proposalID uint64) int {
	t.Helper()
	n := 0
	rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposalID)
	err := app.GovKeeper.Votes.Walk(ctx, rng, func(_ collections.Pair[uint64, sdk.AccAddress], _ govv1.Vote) (bool, error) {
		n++
		return false, nil
	})
	require.NoError(t, err)
	return n
}
