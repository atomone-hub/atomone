package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/photon/keeper"
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
	k, m, ctx := testutil.SetupPhotonKeeper(t)
	m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")
	uatoneSupply := int64(100_000_000_000_000) // 100,000,000atone
	m.BankKeeper.EXPECT().GetSupply(ctx, "uatone").Return(sdk.NewInt64Coin("uatone", uatoneSupply))
	uphotonSupply := int64(100_000_000_000) // 100,000photon
	m.BankKeeper.EXPECT().GetSupply(ctx, "uphoton").Return(sdk.NewInt64Coin("uatone", uphotonSupply))

	resp, err := k.ConversionRate(ctx, &types.QueryConversionRateRequest{})

	require.NoError(t, err)
	expectedConversionRate := sdk.NewDec(keeper.UphotonMaxSupply - uphotonSupply).
		QuoInt64(uatoneSupply).String()
	require.Equal(t, expectedConversionRate, resp.ConversionRate)
}
