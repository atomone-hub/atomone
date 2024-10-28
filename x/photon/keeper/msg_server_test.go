package keeper_test

import (
	"testing"

	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
	"github.com/stretchr/testify/require"
)

func TestMsgServerUpdateParams(t *testing.T) {
	tests := []struct {
		name        string
		msg         *types.MsgUpdateParams
		expectedErr string
	}{
		{
			name: "empty authority field",
			msg: &types.MsgUpdateParams{
				Authority: "",
				Params:    types.Params{MintDisabled: true},
			},
			expectedErr: "invalid authority; expected cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn, got : expected gov account as only signer for proposal message",
		},
		{
			name: "invalid authority field",
			msg: &types.MsgUpdateParams{
				Authority: "xxx",
				Params:    types.Params{MintDisabled: true},
			},
			expectedErr: "invalid authority; expected cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn, got xxx: expected gov account as only signer for proposal message",
		},
		{
			name: "ok",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params:    types.Params{MintDisabled: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, _, ctx := testutil.SetupMsgServer(t)
			params := types.DefaultParams()
			k.SetParams(ctx, params)

			_, err := ms.UpdateParams(ctx, tt.msg)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			got := k.GetParams(ctx)
			require.Equal(t, got, tt.msg.Params)
		})
	}
}
