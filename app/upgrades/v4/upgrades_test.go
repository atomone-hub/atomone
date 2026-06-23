package v4_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/app/helpers"
	v4 "github.com/atomone-hub/atomone/app/upgrades/v4"
)

func TestMigrateStakingParams(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{})
	sk := app.StakingKeeper

	require.NoError(t, v4.MigrateStakingParams(ctx, sk))

	got, err := sk.GetParams(ctx)
	require.NoError(t, err)
	require.NoError(t, got.Validate())
	require.Equal(t, sdk.NewCoin(got.BondDenom, math.NewInt(100_000000)), got.KeyRotationFee)
	fivePercent := math.LegacyMustNewDecFromStr("0.05")
	require.Equal(t, fivePercent, got.MinCommissionRate)
	require.Equal(t, fivePercent, got.MaxCommissionRate)
}
