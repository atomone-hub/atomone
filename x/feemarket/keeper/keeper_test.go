package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	"github.com/atomone-hub/atomone/x/feemarket/testutil"
	"github.com/atomone-hub/atomone/x/feemarket/types"
)

func TestState(t *testing.T) {
	k, _, ctx := testutil.SetupKeeper(t, 0)
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
	k, _, ctx := testutil.SetupKeeper(t, 0)
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
			Alpha:                  math.LegacyMustNewDecFromStr("0.1"),
			Beta:                   math.LegacyMustNewDecFromStr("0.1"),
			Gamma:                  math.LegacyMustNewDecFromStr("0.1"),
			MinBaseGasPrice:        math.LegacyNewDec(10),
			TargetBlockUtilization: math.LegacyMustNewDecFromStr("0.1"),
			MinLearningRate:        math.LegacyMustNewDecFromStr("0.1"),
			MaxLearningRate:        math.LegacyMustNewDecFromStr("0.1"),
			Window:                 1,
			Enabled:                true,
		}

		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		gotParams, err := k.GetParams(ctx)
		require.NoError(t, err)

		require.Equal(t, params, gotParams)
	})
}

func TestEnabledHeight(t *testing.T) {
	k, _, ctx := testutil.SetupKeeper(t, 0)
	t.Run("get and set values", func(t *testing.T) {
		k.SetEnabledHeight(ctx, 10)

		got, err := k.GetEnabledHeight(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(10), got)
	})
}
