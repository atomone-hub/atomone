package photon_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/x/photon"
	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}
	k, _, ctx := testutil.SetupPhotonKeeper(t)

	photon.InitGenesis(ctx, *k, genesisState)
	got := photon.ExportGenesis(ctx, *k)

	require.NotNil(t, got)
	require.Equal(t, genesisState, *got)
}
