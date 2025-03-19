package post_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/feemarket/post"
	"github.com/atomone-hub/atomone/x/feemarket/testutil"
	"github.com/atomone-hub/atomone/x/feemarket/types"
)

func TestPostHandle(t *testing.T) {
	tests := []struct {
		name             string
		genTx            bool
		simulate         bool
		disableFeemarket bool
		enabledHeight    int64
		expectedState    func() types.State
		expectedError    string
	}{
		{
			name:          "ok: skip gentx",
			genTx:         true,
			expectedState: types.DefaultState, // no state update
		},
		{
			name:             "ok: feemarket disabled",
			disableFeemarket: true,
			expectedState:    types.DefaultState, // no state update
		},
		{
			name:          "ok: enabled height not reached",
			enabledHeight: 2,
			expectedState: types.DefaultState, // no state update
		},
		{
			name: "ok: state updated",
			expectedState: func() types.State {
				s := types.DefaultState()
				s.Window[0] = 11986
				return s
			},
		},
		{
			name:     "ok: simulate && state updated",
			simulate: true,
			expectedState: func() types.State {
				s := types.DefaultState()
				s.Window[0] = 11986
				return s
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, _, ctx := testutil.SetupFeemarketKeeper(t)
			// set default params and state
			params := types.DefaultParams()
			params.Enabled = !tt.disableFeemarket
			err := k.SetParams(ctx, params)
			require.NoError(t, err)
			err = k.SetState(ctx, types.DefaultState())
			require.NoError(t, err)
			if tt.enabledHeight > 0 {
				k.SetEnabledHeight(ctx, tt.enabledHeight)
			}
			var (
				nextInvoked bool
				next        = func(ctx sdk.Context, tx sdk.Tx, simulate, success bool) (sdk.Context, error) {
					nextInvoked = true
					return ctx, nil
				}
			)
			dfd := post.NewFeemarketStateUpdateDecorator(k)
			if tt.genTx {
				ctx = ctx.WithBlockHeight(0)
			} else {
				ctx = ctx.WithBlockHeight(1)
			}

			_, err = dfd.PostHandle(ctx, nil, tt.simulate, true, next)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			assert.True(t, nextInvoked, "next is not invoked")
			updatedState, err := k.GetState(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedState(), updatedState)
		})
	}
}
