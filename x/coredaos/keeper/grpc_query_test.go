package keeper_test

import (
	"testing"

	"github.com/atomone-hub/atomone/x/coredaos/keeper"
	"github.com/atomone-hub/atomone/x/coredaos/testutil"
	"github.com/atomone-hub/atomone/x/coredaos/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	k, _, ctx := testutil.SetupCoredaosKeeper(t)
	params := types.DefaultParams()
	k.Params.Set(ctx, params)
	q := keeper.NewQuerier(*k)
	resp, err := q.Params(ctx, &types.QueryParamsRequest{})

	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, resp)
}
