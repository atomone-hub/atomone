package keeper_test

import (
	"testing"

	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, _, ctx := testutil.SetupPhotonKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)
	got := k.GetParams(ctx)

	require.EqualValues(t, params, got)
}
