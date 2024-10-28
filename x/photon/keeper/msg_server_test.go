package keeper_test

import (
	"testing"

	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgServerBurn(t *testing.T) {
	toAddress := sdk.AccAddress("test1")
	tests := []struct {
		name             string
		params           types.Params
		msg              *types.MsgBurn
		setup            func(sdk.Context, testutil.Mocks)
		expectedErr      string
		expectedResponse *types.MsgBurnResponse
	}{
		{
			name:        "fail: mint disabled",
			params:      types.Params{MintDisabled: true},
			msg:         &types.MsgBurn{},
			expectedErr: "photon mint disabled",
		},
		{
			name:   "fail: empty Burn field",
			params: types.Params{MintDisabled: false},
			msg:    &types.MsgBurn{},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")
			},
			expectedErr: "invalid burned amount denom: expected bond denom",
		},
		{
			name:   "fail: invalid Burn field denom",
			params: types.Params{MintDisabled: false},
			msg: &types.MsgBurn{
				Amount: sdk.NewInt64Coin("xxx", 42),
			},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")
			},
			expectedErr: "invalid burned amount denom: expected bond denom",
		},
		{
			name:   "fail: photon_supply=max",
			params: types.Params{MintDisabled: false},
			msg: &types.MsgBurn{
				ToAddress: toAddress.String(),
				Amount:    sdk.NewInt64Coin("uatone", 1),
			},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")
				m.BankKeeper.EXPECT().GetSupply(ctx, "uatone").Return(sdk.NewInt64Coin("uatone", 1_000_000))
				m.BankKeeper.EXPECT().GetSupply(ctx, "uphoton").Return(sdk.NewInt64Coin("uphoton", 1_000_000_000))
			},
			expectedErr: "no more photon can be minted",
		},
		{
			name:   "fail: photon_supply+minted>max",
			params: types.Params{MintDisabled: false},
			msg: &types.MsgBurn{
				ToAddress: toAddress.String(),
				Amount:    sdk.NewInt64Coin("uatone", 1_000_001),
			},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")
				m.BankKeeper.EXPECT().GetSupply(ctx, "uatone").Return(sdk.NewInt64Coin("uatone", 1_000_000))
				m.BankKeeper.EXPECT().GetSupply(ctx, "uphoton").Return(sdk.NewInt64Coin("uphoton", 999_999_999))
			},
			expectedErr: "not enough photon can be minted",
		},
		{
			name:   "ok: photon_supply=0",
			params: types.Params{MintDisabled: false},
			msg: &types.MsgBurn{
				ToAddress: toAddress.String(),
				Amount:    sdk.NewInt64Coin("uatone", 1),
			},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")
				m.BankKeeper.EXPECT().GetSupply(ctx, "uatone").Return(sdk.NewInt64Coin("uatone", 1_000_000))
				m.BankKeeper.EXPECT().GetSupply(ctx, "uphoton").Return(sdk.NewInt64Coin("uphoton", 0))
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(
					ctx, toAddress, types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin("uatone", 1)),
				)
				m.BankKeeper.EXPECT().BurnCoins(ctx, types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin("uatone", 1)),
				)
				m.BankKeeper.EXPECT().MintCoins(ctx, types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin("uphoton", 1000)),
				)
				m.BankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.ModuleName, toAddress,
					sdk.NewCoins(sdk.NewInt64Coin("uphoton", 1000)),
				)
			},
			expectedResponse: &types.MsgBurnResponse{
				Minted:         sdk.NewInt64Coin("uphoton", 1000),
				ConversionRate: "1000.000000000000000000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, mocks, ctx := testutil.SetupMsgServer(t)
			k.SetParams(ctx, tt.params)
			if tt.setup != nil {
				tt.setup(ctx, mocks)
			}

			resp, err := ms.Burn(ctx, tt.msg)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, resp, tt.expectedResponse)
		})
	}
}

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
