package atomone_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"github.com/atomone-hub/atomone/app/helpers"
	v5 "github.com/atomone-hub/atomone/app/upgrades/v5"
)

// TestMintUpgradeBlock runs the v5 upgrade against x/mint state as the
// previous binary stored it, and checks the upgrade block and the one after.
func TestMintUpgradeBlock(t *testing.T) {
	app := helpers.Setup(t)
	_, err := app.Commit()
	require.NoError(t, err)

	cdc := app.AppCodec()
	committedCtx := func() sdk.Context {
		return app.NewUncachedContext(false, cmtproto.Header{Height: app.LastBlockHeight()})
	}
	ctx := committedCtx()
	store := ctx.KVStore(app.GetKey(minttypes.StoreKey))

	// Params as stored by x/mint consensus version 2: no max_block_elapsed,
	// and the varint field 6 (blocks_per_year) present.
	params := minttypes.DefaultParams()
	params.MaxBlockElapsed = 0
	bz, err := cdc.Marshal(&params)
	require.NoError(t, err)
	bz = protowire.AppendTag(bz, 6, protowire.VarintType)
	bz = protowire.AppendVarint(bz, 4_360_000)
	store.Set(minttypes.ParamsKey, bz)

	// Minter as stored by the previous binary: no previous_block_time.
	minter := minttypes.NewMinter(math.LegacyNewDecWithPrec(20, 2), math.LegacyZeroDec())
	bz, err = cdc.Marshal(&minter)
	require.NoError(t, err)
	store.Set(minttypes.MinterKey, bz)

	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	vm[minttypes.ModuleName] = 2
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))

	upgradeHeight := app.LastBlockHeight() + 1
	plan := upgradetypes.Plan{Name: v5.UpgradeName, Height: upgradeHeight}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))

	denom := params.MintDenom
	finalize := func(height int64, blockTime time.Time) (math.Int, []abci.Event) {
		t.Helper()
		before := app.BankKeeper.GetSupply(committedCtx(), denom).Amount
		res, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: height,
			Time:   blockTime,
			Hash:   app.LastCommitID().Hash,
		})
		require.NoError(t, err)
		_, err = app.Commit()
		require.NoError(t, err)
		after := app.BankKeeper.GetSupply(committedCtx(), denom).Amount
		return after.Sub(before), res.Events
	}

	// The upgrade block runs the migration in PreBlocker; the minter then
	// finds no reference point, records the block time, mints nothing and
	// emits no mint event.
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	minted, events := finalize(upgradeHeight, t0)
	require.True(t, minted.IsZero(), "minted %s", minted)
	for _, event := range events {
		require.NotEqual(t, minttypes.EventTypeMint, event.Type)
	}

	ctx = committedCtx()
	vm, err = app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 3, vm[minttypes.ModuleName])
	got, err := app.MintKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, minttypes.DefaultMaxBlockElapsed, got.MaxBlockElapsed)
	gotMinter, err := app.MintKeeper.Minter.Get(ctx)
	require.NoError(t, err)
	require.NotNil(t, gotMinter.PreviousBlockTime)
	require.True(t, gotMinter.PreviousBlockTime.Equal(t0), "previous block time %s", gotMinter.PreviousBlockTime)

	// The next block mints six seconds' worth of provisions.
	minted, _ = finalize(upgradeHeight+1, t0.Add(6*time.Second))
	gotMinter, err = app.MintKeeper.Minter.Get(committedCtx())
	require.NoError(t, err)
	expected := gotMinter.AnnualProvisions.MulInt64((6 * time.Second).Nanoseconds()).QuoInt64(minttypes.NanosecondsPerYear).TruncateInt()
	require.True(t, minted.IsPositive())
	require.Equal(t, expected, minted)
}
