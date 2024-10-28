package keeper_test

import (
	"testing"

	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	k, _, ctx := testutil.SetupPhotonKeeper(t)
	params := types.DefaultParams()
	k.SetParams(ctx, params)

	response, err := k.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)

	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
