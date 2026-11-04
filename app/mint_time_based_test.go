package atomone_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"github.com/atomone-hub/atomone/app/helpers"
)

func TestMintingIsProportionalToElapsedTime(t *testing.T) {
	// helpers.Setup finalizes block 1 without a block time, so the minter
	// records the zero time as its reference point.
	app := helpers.Setup(t)
	_, err := app.Commit()
	require.NoError(t, err)

	committedCtx := func() sdk.Context {
		return app.NewUncachedContext(false, cmtproto.Header{Height: app.LastBlockHeight()})
	}
	params, err := app.MintKeeper.Params.Get(committedCtx())
	require.NoError(t, err)
	denom := params.MintDenom

	// finalize runs and commits a block at the given height and time, and
	// returns the amount of denom minted by that block and the minter state
	// stored after it.
	finalize := func(height int64, blockTime time.Time) (math.Int, minttypes.Minter) {
		t.Helper()
		before := app.BankKeeper.GetSupply(committedCtx(), denom).Amount
		_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: height,
			Time:   blockTime,
			Hash:   app.LastCommitID().Hash,
		})
		require.NoError(t, err)
		_, err = app.Commit()
		require.NoError(t, err)

		ctx := committedCtx()
		after := app.BankKeeper.GetSupply(ctx, denom).Amount
		minter, err := app.MintKeeper.Minter.Get(ctx)
		require.NoError(t, err)
		return after.Sub(before), minter
	}

	// expected is the share of the minter's annual provisions for elapsed.
	expected := func(minter minttypes.Minter, elapsed time.Duration) math.Int {
		return minter.AnnualProvisions.MulInt64(elapsed.Nanoseconds()).QuoInt64(minttypes.NanosecondsPerYear).TruncateInt()
	}

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Block 2: the elapsed time since the zero reference saturates and is
	// capped at MaxBlockElapsed.
	minted, minter := finalize(2, t0)
	require.True(t, minter.AnnualProvisions.IsPositive())
	require.Equal(t, expected(minter, minttypes.MaxBlockElapsed), minted)

	// Block 3: six seconds later mints six seconds' worth of provisions.
	t1 := t0.Add(6 * time.Second)
	minted, minter = finalize(3, t1)
	require.True(t, minted.IsPositive())
	require.Equal(t, expected(minter, 6*time.Second), minted)

	// Block 4: a ten-minute gap mints only MaxBlockElapsed worth.
	t2 := t1.Add(10 * time.Minute)
	minted, minter = finalize(4, t2)
	require.Equal(t, expected(minter, minttypes.MaxBlockElapsed), minted)

	// Block 5: no time elapsed, nothing minted.
	minted, minter = finalize(5, t2)
	require.True(t, minted.IsZero(), "minted %s", minted)
	require.NotNil(t, minter.PreviousBlockTime)
	require.True(t, minter.PreviousBlockTime.Equal(t2), "previous block time %s", minter.PreviousBlockTime)
}
