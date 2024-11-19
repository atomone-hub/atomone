package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	appparams "github.com/atomone-hub/atomone/app/params"
	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	k, _, ctx := testutil.SetupPhotonKeeper(t)
	params := types.DefaultParams()
	k.SetParams(ctx, params)

	resp, err := k.Params(ctx, &types.QueryParamsRequest{})

	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, resp)
}

func TestConversionRateQuery(t *testing.T) {
	tests := []struct {
		name             string
		uatoneSupply     int64
		uphotonSupply    int64
		expectedResponse *types.QueryConversionRateResponse
	}{
		{
			name:          "nominal case",
			uatoneSupply:  100_000_000_000_000, // 100,000,000atone
			uphotonSupply: 100_000_000_000,     // 100,000photon
			expectedResponse: &types.QueryConversionRateResponse{
				ConversionRate: "9.999000000000000000",
			},
		},
		{
			name:          "max supply of photon exceeded",
			uatoneSupply:  100_000_000_000_000, // 100,000,000atone
			uphotonSupply: types.MaxSupply + 1,
			expectedResponse: &types.QueryConversionRateResponse{
				ConversionRate: "0.000000000000000000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, m, ctx := testutil.SetupPhotonKeeper(t)
			m.StakingKeeper.EXPECT().BondDenom(ctx).Return(appparams.BondDenom)
			m.BankKeeper.EXPECT().GetSupply(ctx, appparams.BondDenom).
				Return(sdk.NewInt64Coin(appparams.BondDenom, tt.uatoneSupply))
			m.BankKeeper.EXPECT().GetSupply(ctx, types.Denom).
				Return(sdk.NewInt64Coin(appparams.BondDenom, tt.uphotonSupply))

			resp, err := k.ConversionRate(ctx, &types.QueryConversionRateRequest{})

			require.NoError(t, err)
			require.Equal(t, tt.expectedResponse, resp)
		})
	}
}
