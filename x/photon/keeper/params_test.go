package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/atomone-hub/atomone/testutil/keeper"
	"github.com/atomone-hub/atomone/x/photon/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.PhotonKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
