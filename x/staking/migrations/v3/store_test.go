package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/testutil"
	sdk "github.com/atomone-hub/atomone/types"
	moduletestutil "github.com/atomone-hub/atomone/types/module/testutil"
	paramtypes "github.com/atomone-hub/atomone/x/params/types"
	v3 "github.com/atomone-hub/atomone/x/staking/migrations/v3"
	"github.com/atomone-hub/atomone/x/staking/types"
)

func TestStoreMigration(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	stakingKey := sdk.NewKVStoreKey("staking")
	tStakingKey := sdk.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(stakingKey, tStakingKey)
	paramstore := paramtypes.NewSubspace(encCfg.Codec, encCfg.Amino, stakingKey, tStakingKey, "staking")

	// Check no params
	require.False(t, paramstore.Has(ctx, types.KeyMinCommissionRate))

	// Run migrations.
	err := v3.MigrateStore(ctx, stakingKey, encCfg.Codec, paramstore)
	require.NoError(t, err)

	// Make sure the new params are set.
	require.True(t, paramstore.Has(ctx, types.KeyMinCommissionRate))
}
