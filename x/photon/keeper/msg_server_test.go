package keeper_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	appparams "github.com/atomone-hub/atomone/app/params"
	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
)

func TestMsgServerMintPhoton(t *testing.T) {
	var (
		toAddress         = sdk.AccAddress("test1")
		atoneSupply int64 = 107_775_332 * 1_000_000 // From genesis
	)
	tests := []struct {
		name             string
		params           types.Params
		msg              *types.MsgMintPhoton
		setup            func(sdk.Context, testutil.Mocks)
		expectedErr      string
		expectedResponse *types.MsgMintPhotonResponse
	}{
		{
			name:        "fail: mint disabled",
			params:      types.Params{MintDisabled: true},
			msg:         &types.MsgMintPhoton{},
			expectedErr: "photon mint disabled",
		},
		{
			name:   "fail: empty Amount field",
			params: types.Params{MintDisabled: false},
			msg:    &types.MsgMintPhoton{},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return(appparams.BondDenom)
			},
			expectedErr: "invalid burned amount denom: expected bond denom",
		},
		{
			name:   "fail: invalid Amount field denom",
			params: types.Params{MintDisabled: false},
			msg: &types.MsgMintPhoton{
				Amount: sdk.NewInt64Coin("xxx", 42),
			},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return(appparams.BondDenom)
			},
			expectedErr: "invalid burned amount denom: expected bond denom",
		},
		{
			name:   "fail: photon_supply=max",
			params: types.Params{MintDisabled: false},
			msg: &types.MsgMintPhoton{
				ToAddress: toAddress.String(),
				Amount:    sdk.NewInt64Coin(appparams.BondDenom, 1),
			},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return(appparams.BondDenom)
				m.BankKeeper.EXPECT().GetSupply(ctx, appparams.BondDenom).
					Return(sdk.NewInt64Coin(appparams.BondDenom, atoneSupply))
				m.BankKeeper.EXPECT().GetSupply(ctx, types.Denom).Return(sdk.NewInt64Coin(types.Denom, types.MaxSupply))
			},
			expectedErr: "no mintable photon after rounding, try higher burn",
		},
		{
			name:   "fail: atone_supply >> photon_supply",
			params: types.Params{MintDisabled: false},
			msg: &types.MsgMintPhoton{
				ToAddress: toAddress.String(),
				Amount:    sdk.NewInt64Coin(appparams.BondDenom, 1),
			},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return(appparams.BondDenom)
				m.BankKeeper.EXPECT().GetSupply(ctx, appparams.BondDenom).
					Return(sdk.NewInt64Coin(appparams.BondDenom, math.MaxInt))
				m.BankKeeper.EXPECT().GetSupply(ctx, types.Denom).Return(sdk.NewInt64Coin(types.Denom, 0))
			},
			expectedErr: "no mintable photon after rounding, try higher burn",
		},
		{
			name:   "ok: photon_supply=0",
			params: types.Params{MintDisabled: false},
			msg: &types.MsgMintPhoton{
				ToAddress: toAddress.String(),
				Amount:    sdk.NewInt64Coin(appparams.BondDenom, 1),
			},
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return(appparams.BondDenom)
				m.BankKeeper.EXPECT().GetSupply(ctx, appparams.BondDenom).
					Return(sdk.NewInt64Coin(appparams.BondDenom, atoneSupply))
				m.BankKeeper.EXPECT().GetSupply(ctx, types.Denom).Return(sdk.NewInt64Coin(types.Denom, 0))
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(
					ctx, toAddress, types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin(appparams.BondDenom, 1)),
				)
				m.BankKeeper.EXPECT().BurnCoins(ctx, types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin(appparams.BondDenom, 1)),
				)
				m.BankKeeper.EXPECT().MintCoins(ctx, types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin(types.Denom, 9)),
				)
				m.BankKeeper.EXPECT().SendCoinsFromModuleToAccount(
					ctx, types.ModuleName, toAddress,
					sdk.NewCoins(sdk.NewInt64Coin(types.Denom, 9)),
				)
			},
			expectedResponse: &types.MsgMintPhotonResponse{
				Minted:         sdk.NewInt64Coin(types.Denom, 9),
				ConversionRate: "9.278561071841560182",
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

			resp, err := ms.MintPhoton(ctx, tt.msg)

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
