package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtime "github.com/cometbft/cometbft/types/time"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/x/feemarket/keeper"
	"github.com/atomone-hub/atomone/x/feemarket/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
)

func setupKeeper(t *testing.T) (*keeper.Keeper, sdk.Context) {
	t.Helper()
	key := sdk.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, sdk.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(tmproto.Header{Time: tmtime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	// banktypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	k := keeper.NewKeeper(encCfg.Codec, key, &types.ErrorDenomResolver{}, authority)
	return k, ctx
}

func TestState(t *testing.T) {
	k, ctx := setupKeeper(t)
	t.Run("set and get default eip1559 state", func(t *testing.T) {
		state := types.DefaultState()

		err := k.SetState(ctx, state)
		require.NoError(t, err)

		gotState, err := k.GetState(ctx)
		require.NoError(t, err)

		require.Equal(t, state, gotState)
	})

	t.Run("set and get aimd eip1559 state", func(t *testing.T) {
		state := types.DefaultAIMDState()

		err := k.SetState(ctx, state)
		require.NoError(t, err)

		gotState, err := k.GetState(ctx)
		require.NoError(t, err)

		require.Equal(t, state, gotState)
	})
}

func TestParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	t.Run("set and get default params", func(t *testing.T) {
		params := types.DefaultParams()

		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		gotParams, err := k.GetParams(ctx)
		require.NoError(t, err)

		require.Equal(t, params, gotParams)
	})

	t.Run("set and get custom params", func(t *testing.T) {
		params := types.Params{
			Alpha:               math.LegacyMustNewDecFromStr("0.1"),
			Beta:                math.LegacyMustNewDecFromStr("0.1"),
			Gamma:               math.LegacyMustNewDecFromStr("0.1"),
			Delta:               math.LegacyMustNewDecFromStr("0.1"),
			MinBaseGasPrice:     math.LegacyNewDec(10),
			MinLearningRate:     math.LegacyMustNewDecFromStr("0.1"),
			MaxLearningRate:     math.LegacyMustNewDecFromStr("0.1"),
			MaxBlockUtilization: 10,
			Window:              1,
			Enabled:             true,
		}

		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		gotParams, err := k.GetParams(ctx)
		require.NoError(t, err)

		require.Equal(t, params, gotParams)
	})
}

func TestEnabledHeight(t *testing.T) {
	k, ctx := setupKeeper(t)
	t.Run("get and set values", func(t *testing.T) {
		k.SetEnabledHeight(ctx, 10)

		got, err := k.GetEnabledHeight(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(10), got)
	})
}
