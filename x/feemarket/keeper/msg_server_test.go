package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/x/feemarket/testutil"
	"github.com/atomone-hub/atomone/x/feemarket/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
)

func TestMsgUpdateParams(t *testing.T) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	t.Run("accepts a req with no params", func(t *testing.T) {
		require := require.New(t)
		msgServer, k, ctx := testutil.SetupMsgServer(t, 0)
		req := &types.MsgUpdateParams{
			Authority: authority,
		}
		resp, err := msgServer.UpdateParams(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		params, err := k.GetParams(ctx)
		require.NoError(err)
		require.False(params.Enabled)
	})

	t.Run("accepts a req with params", func(t *testing.T) {
		require := require.New(t)
		msgServer, k, ctx := testutil.SetupMsgServer(t, 0)
		req := &types.MsgUpdateParams{
			Authority: authority,
			Params:    types.DefaultParams(),
		}
		resp, err := msgServer.UpdateParams(ctx, req)
		require.NoError(err)
		require.NotNil(resp)

		params, err := k.GetParams(ctx)
		require.NoError(err)
		require.Equal(req.Params, params)
	})

	t.Run("rejects a req with invalid signer", func(t *testing.T) {
		require := require.New(t)
		msgServer, _, ctx := testutil.SetupMsgServer(t, 0)
		req := &types.MsgUpdateParams{
			Authority: "invalid",
		}
		_, err := msgServer.UpdateParams(ctx, req)
		require.Error(err)
	})

	t.Run("sets enabledHeight when transitioning from disabled -> enabled", func(t *testing.T) {
		require := require.New(t)
		msgServer, k, ctx := testutil.SetupMsgServer(t, 0)
		ctx = ctx.WithBlockHeight(ctx.BlockHeight())
		enabledParams := types.DefaultParams()

		req := &types.MsgUpdateParams{
			Authority: authority,
			Params:    enabledParams,
		}
		_, err := msgServer.UpdateParams(ctx, req)
		require.NoError(err)

		disableParams := types.DefaultParams()
		disableParams.Enabled = false

		req = &types.MsgUpdateParams{
			Authority: authority,
			Params:    disableParams,
		}
		_, err = msgServer.UpdateParams(ctx, req)
		require.NoError(err)

		gotHeight, err := k.GetEnabledHeight(ctx)
		require.NoError(err)
		require.Equal(ctx.BlockHeight(), gotHeight)

		// now that the markets are disabled, enable and check block height
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 10)

		req = &types.MsgUpdateParams{
			Authority: authority,
			Params:    enabledParams,
		}
		_, err = msgServer.UpdateParams(ctx, req)
		require.NoError(err)

		newHeight, err := k.GetEnabledHeight(ctx)
		require.NoError(err)
		require.Equal(ctx.BlockHeight(), newHeight)
	})

	t.Run("resets state after new params request", func(t *testing.T) {
		require := require.New(t)
		msgServer, k, ctx := testutil.SetupMsgServer(t, 0)
		params, err := k.GetParams(ctx)
		require.NoError(err)
		err = k.SetState(ctx, types.DefaultState())
		require.NoError(err)

		state, err := k.GetState(ctx)
		require.NoError(err)

		err = state.Update(testutil.MaxBlockGas, testutil.MaxBlockGas)
		require.NoError(err)

		err = k.SetState(ctx, state)
		require.NoError(err)

		params.Window = 100
		req := &types.MsgUpdateParams{
			Authority: authority,
			Params:    params,
		}
		_, err = msgServer.UpdateParams(ctx, req)
		require.NoError(err)

		state, err = k.GetState(ctx)
		require.NoError(err)
		require.Equal(params.Window, uint64(len(state.Window)))
		require.Equal(state.Window[0], uint64(0))
	})
}
