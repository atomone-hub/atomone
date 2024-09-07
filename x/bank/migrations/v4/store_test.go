package v4_test

import (
	"testing"

	"github.com/atomone-hub/atomone/testutil"
	sdk "github.com/atomone-hub/atomone/types"
	moduletestutil "github.com/atomone-hub/atomone/types/module/testutil"
	"github.com/atomone-hub/atomone/x/bank"
	"github.com/atomone-hub/atomone/x/bank/exported"
	v4 "github.com/atomone-hub/atomone/x/bank/migrations/v4"
	"github.com/atomone-hub/atomone/x/bank/types"
	"github.com/stretchr/testify/require"
)

type mockSubspace struct {
	ps types.Params
}

func newMockSubspace(ps types.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSet(ctx sdk.Context, ps exported.ParamSet) {
	*ps.(*types.Params) = ms.ps
}

func (ms mockSubspace) Get(ctx sdk.Context, key []byte, ptr interface{}) {}

func TestMigrate(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(bank.AppModuleBasic{})
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(v4.ModuleName)
	tKey := sdk.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	store := ctx.KVStore(storeKey)

	legacySubspace := newMockSubspace(types.DefaultParams())
	require.NoError(t, v4.MigrateStore(ctx, storeKey, legacySubspace, cdc))

	var res types.Params
	bz := store.Get(v4.ParamsKey)
	require.NoError(t, cdc.Unmarshal(bz, &res))
	require.Equal(t, legacySubspace.ps, res)
}
