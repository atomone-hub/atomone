package keeper_test

import (
	"testing"

	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestKeeperConvertToDenom(t *testing.T) {
	tests := []struct {
		name         string
		coin         sdk.DecCoin
		denom        string
		setup        func(sdk.Context, testutil.Mocks)
		expectedCoin sdk.DecCoin
		expectedErr  string
	}{
		{
			name:         "ok: denom==coin.Denom",
			coin:         sdk.NewInt64DecCoin("uphoton", 10),
			denom:        "uphoton",
			expectedCoin: sdk.NewInt64DecCoin("uphoton", 10),
		},
		{
			name:  "ok: denom==bondDenom",
			coin:  sdk.NewInt64DecCoin("uphoton", 10),
			denom: "uatone",
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")
				m.BankKeeper.EXPECT().GetSupply(ctx, "uatone").Return(sdk.NewInt64Coin("uatone", 100_000_000_000_000))
				m.BankKeeper.EXPECT().GetSupply(ctx, "uphoton").Return(sdk.NewInt64Coin("uphoton", 0))
			},
			expectedCoin: sdk.NewInt64DecCoin("uatone", 1),
		},
		{
			name:  "fail: random denom",
			coin:  sdk.NewInt64DecCoin("uphoton", 10),
			denom: "xxx",
			setup: func(ctx sdk.Context, m testutil.Mocks) {
				m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")
			},
			expectedErr: "error resolving denom 'xxx'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, m, ctx := testutil.SetupPhotonKeeper(t)
			if tt.setup != nil {
				tt.setup(ctx, m)
			}

			coin, err := k.ConvertToDenom(ctx, tt.coin, tt.denom)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedCoin.String(), coin.String())
		})
	}
}

func TestKeeperExtraDenoms(t *testing.T) {
	k, m, ctx := testutil.SetupPhotonKeeper(t)
	m.StakingKeeper.EXPECT().BondDenom(ctx).Return("uatone")

	denoms, err := k.ExtraDenoms(ctx)

	require.NoError(t, err)
	require.Equal(t, []string{"uatone"}, denoms)
}
