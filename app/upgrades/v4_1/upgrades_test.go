package v4_1_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/app/helpers"
	v4_1 "github.com/atomone-hub/atomone/app/upgrades/v4_1"
)

func TestInitKeyRotationFee(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{})
	sk := app.StakingKeeper

	require.NoError(t, v4_1.InitKeyRotationFee(ctx, sk))

	got, err := sk.GetParams(ctx)
	require.NoError(t, err)
	require.NoError(t, got.Validate())
	require.Equal(t, sdk.NewCoin(got.BondDenom, math.NewInt(100_000000)), got.KeyRotationFee)
}
