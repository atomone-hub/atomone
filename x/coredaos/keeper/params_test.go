package keeper_test

import (
	"testing"

	"github.com/atomone-hub/atomone/x/coredaos/testutil"
	"github.com/atomone-hub/atomone/x/coredaos/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, _, ctx := testutil.SetupCoredaosKeeper(t)
	params := types.DefaultParams()

	k.Params.Set(ctx, params)
	got := k.GetParams(ctx)

	require.EqualValues(t, params, got)
}
