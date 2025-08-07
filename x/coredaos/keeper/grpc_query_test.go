package keeper_test

import (
	"github.com/atomone-hub/atomone/x/coredaos/testutil"
	"github.com/atomone-hub/atomone/x/coredaos/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParamsQuery(t *testing.T) {
	k, _, ctx := testutil.SetupCoredaosKeeper(t)
	params := types.DefaultParams()
	k.SetParams(ctx, params)

	resp, err := k.Params(ctx, &types.QueryParamsRequest{})

	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, resp)
}
