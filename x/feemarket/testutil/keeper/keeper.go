// Package keeper provides methods to initialize SDK keepers with local storage for test purposes
package keeper

import (
	"testing"

	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	testkeeper "github.com/skip-mev/chaintestutil/keeper"
	"github.com/stretchr/testify/require"

	feemarketkeeper "github.com/atomone-hub/atomone/x/feemarket/keeper"
	feemarkettypes "github.com/atomone-hub/atomone/x/feemarket/types"
)

// TestKeepers holds all keepers used during keeper tests for all modules
type TestKeepers struct {
	testkeeper.TestKeepers
	FeeMarketKeeper *feemarketkeeper.Keeper
}

// TestMsgServers holds all message servers used during keeper tests for all modules
type TestMsgServers struct {
	testkeeper.TestMsgServers
	FeeMarketMsgServer feemarkettypes.MsgServer
}

var additionalMaccPerms = map[string][]string{
	feemarkettypes.ModuleName: nil,
}

// NewTestSetup returns initialized instances of all the keepers and message servers of the modules
func NewTestSetup(t testing.TB, options ...testkeeper.SetupOption) (sdk.Context, TestKeepers, TestMsgServers) {
	options = append(options, testkeeper.WithAdditionalModuleAccounts(additionalMaccPerms))

	_, tk, tms := testkeeper.NewTestSetup(t, options...)

	// initialize extra keeper
	feeMarketKeeper := FeeMarket(tk.Initializer)
	require.NoError(t, tk.Initializer.LoadLatest())

	// initialize msg servers
	feeMarketMsgSrv := feemarketkeeper.NewMsgServer(feeMarketKeeper)

	ctx := sdk.NewContext(tk.Initializer.StateStore, tmproto.Header{
		Time:   testkeeper.ExampleTimestamp,
		Height: testkeeper.ExampleHeight,
	}, false, log.NewNopLogger())

	err := feeMarketKeeper.SetState(ctx, feemarkettypes.DefaultState())
	require.NoError(t, err)
	err = feeMarketKeeper.SetParams(ctx, feemarkettypes.DefaultParams())
	require.NoError(t, err)

	testKeepers := TestKeepers{
		TestKeepers:     tk,
		FeeMarketKeeper: feeMarketKeeper,
	}

	testMsgServers := TestMsgServers{
		TestMsgServers:     tms,
		FeeMarketMsgServer: feeMarketMsgSrv,
	}

	return ctx, testKeepers, testMsgServers
}

// FeeMarket initializes the fee market module using the testkeepers intializer.
func FeeMarket(
	initializer *testkeeper.Initializer,
) *feemarketkeeper.Keeper {
	storeKey := sdk.NewKVStoreKey(feemarkettypes.StoreKey)
	initializer.StateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, initializer.DB)

	return feemarketkeeper.NewKeeper(
		initializer.Codec,
		storeKey,
		&feemarkettypes.TestDenomResolver{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
}
