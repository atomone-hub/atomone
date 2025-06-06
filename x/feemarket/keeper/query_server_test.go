package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	"github.com/atomone-hub/atomone/x/feemarket/testutil"
	"github.com/atomone-hub/atomone/x/feemarket/types"
)

func TestParamsRequest(t *testing.T) {
	t.Run("can get default params", func(t *testing.T) {
		require := require.New(t)
		queryServer, k, _, ctx := testutil.SetupQueryServer(t, 0)
		err := k.SetParams(ctx, types.DefaultParams())
		require.NoError(err)
		req := &types.ParamsRequest{}
		resp, err := queryServer.Params(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		require.Equal(types.DefaultParams(), resp.Params)

		params, err := k.GetParams(ctx)
		require.NoError(err)

		require.Equal(resp.Params, params)
	})

	t.Run("can get updated params", func(t *testing.T) {
		require := require.New(t)
		queryServer, k, _, ctx := testutil.SetupQueryServer(t, 0)
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
		require.NoError(err)

		req := &types.ParamsRequest{}
		resp, err := queryServer.Params(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		require.Equal(params, resp.Params)

		params, err = k.GetParams(ctx)
		require.NoError(err)

		require.Equal(resp.Params, params)
	})
}

func TestStateRequest(t *testing.T) {
	t.Run("can get default state", func(t *testing.T) {
		require := require.New(t)
		queryServer, k, _, ctx := testutil.SetupQueryServer(t, 0)
		err := k.SetState(ctx, types.DefaultState())
		require.NoError(err)
		req := &types.StateRequest{}
		resp, err := queryServer.State(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		require.Equal(types.DefaultState(), resp.State)

		state, err := k.GetState(ctx)
		require.NoError(err)

		require.Equal(resp.State, state)
	})

	t.Run("can get updated state", func(t *testing.T) {
		require := require.New(t)
		queryServer, k, _, ctx := testutil.SetupQueryServer(t, 0)
		state := types.State{
			BaseGasPrice: math.LegacyOneDec(),
			LearningRate: math.LegacyOneDec(),
			Window:       []uint64{1},
			Index:        0,
		}
		err := k.SetState(ctx, state)
		require.NoError(err)

		req := &types.StateRequest{}
		resp, err := queryServer.State(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		require.Equal(state, resp.State)

		state, err = k.GetState(ctx)
		require.NoError(err)

		require.Equal(resp.State, state)
	})
}

func TestGasPriceRequest(t *testing.T) {
	t.Run("can get gas price", func(t *testing.T) {
		require := require.New(t)
		queryServer, k, _, ctx := testutil.SetupQueryServer(t, 0)
		err := k.SetParams(ctx, types.DefaultParams())
		require.NoError(err)
		err = k.SetState(ctx, types.DefaultState())
		require.NoError(err)
		req := &types.GasPriceRequest{
			Denom: types.DefaultFeeDenom,
		}
		resp, err := queryServer.GasPrice(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		gasPrice, err := k.GetMinGasPrice(ctx, req.GetDenom())
		require.NoError(err)

		require.Equal(resp.GetPrice(), gasPrice)
	})

	t.Run("can get updated gas price", func(t *testing.T) {
		require := require.New(t)
		queryServer, k, _, ctx := testutil.SetupQueryServer(t, 0)
		state := types.State{
			BaseGasPrice: math.LegacyOneDec(),
		}
		err := k.SetState(ctx, state)
		require.NoError(err)

		params := types.Params{
			FeeDenom: "test",
		}
		err = k.SetParams(ctx, params)
		require.NoError(err)

		req := &types.GasPriceRequest{
			Denom: "test",
		}
		resp, err := queryServer.GasPrice(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		gasPrice, err := k.GetMinGasPrice(ctx, req.GetDenom())
		require.NoError(err)

		require.Equal(resp.GetPrice(), gasPrice)
	})

	t.Run("can get updated gas price < 1", func(t *testing.T) {
		require := require.New(t)
		queryServer, k, _, ctx := testutil.SetupQueryServer(t, 0)
		state := types.State{
			BaseGasPrice: math.LegacyMustNewDecFromStr("0.005"),
		}
		err := k.SetState(ctx, state)
		require.NoError(err)

		params := types.Params{
			FeeDenom: "test",
		}
		err = k.SetParams(ctx, params)
		require.NoError(err)

		req := &types.GasPriceRequest{
			Denom: "test",
		}
		resp, err := queryServer.GasPrice(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		fee, err := k.GetMinGasPrice(ctx, req.GetDenom())
		require.NoError(err)

		require.Equal(resp.GetPrice(), fee)
	})
}
