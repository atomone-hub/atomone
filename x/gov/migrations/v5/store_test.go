package v5_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/bank"

	appparams "github.com/atomone-hub/atomone/app/params"
	"github.com/atomone-hub/atomone/x/gov"
	v5 "github.com/atomone-hub/atomone/x/gov/migrations/v5"
	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func TestMigrateStore(t *testing.T) {
	cdc := moduletestutil.MakeTestEncodingConfig(gov.AppModuleBasic{}, bank.AppModuleBasic{}).Codec
	govKey := storetypes.NewKVStoreKey("gov")
	ctx := testutil.DefaultContext(govKey, storetypes.NewTransientStoreKey("transient_test"))
	store := ctx.KVStore(govKey)

	var params govv1.Params
	bz := store.Get(v5.ParamsKey)
	require.NoError(t, cdc.Unmarshal(bz, &params))
	require.NotNil(t, params)
	require.Nil(t, params.MinDepositThrottler)
	require.Nil(t, params.MinInitialDepositThrottler)
	require.Equal(t, "", params.BurnDepositNoThreshold)

	// Run migrations.
	err := v5.MigrateStore(ctx, govKey, cdc)
	require.NoError(t, err)

	// Check params
	bz = store.Get(v5.ParamsKey)
	require.NoError(t, cdc.Unmarshal(bz, &params))
	require.NotNil(t, params)
	expectedMinDepositThrottler := govv1.DefaultParams().MinDepositThrottler
	expectedMinDepositThrottler.FloorValue[0].Denom = appparams.BondDenom
	require.Equal(t, expectedMinDepositThrottler, params.MinDepositThrottler)
	expectedMinInitialDepositThrottler := govv1.DefaultParams().MinInitialDepositThrottler
	expectedMinInitialDepositThrottler.FloorValue[0].Denom = appparams.BondDenom
	require.Equal(t, expectedMinInitialDepositThrottler, params.MinInitialDepositThrottler)
	require.Equal(t, govv1.DefaultParams().BurnDepositNoThreshold, params.BurnDepositNoThreshold)
}
