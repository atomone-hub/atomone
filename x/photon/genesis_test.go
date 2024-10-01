package photon_test

import (
	"testing"

	keepertest "github.com/atomone-hub/atomone/testutil/keeper"
	"github.com/atomone-hub/atomone/testutil/nullify"
	"github.com/atomone-hub/atomone/x/photon"
	"github.com/atomone-hub/atomone/x/photon/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.PhotonKeeper(t)
	photon.InitGenesis(ctx, *k, genesisState)
	got := photon.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
