package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/atomone-hub/atomone/x/photon/types"
)

const (
	MintDisabled = "mint_disabled"
)

// GenMintDisabled returns a randomized MintDisabled param.
func GenMintDisabled(r *rand.Rand) bool {
	return r.Int63n(101) <= 15 // 15% chance of mint being disabled
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	var mintDisabled bool
	simState.AppParams.GetOrGenerate(
		simState.Cdc, MintDisabled, &mintDisabled, simState.Rand,
		func(r *rand.Rand) { mintDisabled = GenMintDisabled(r) },
	)

	photonGenesis := types.NewGenesisState(
		types.NewParams(mintDisabled),
	)

	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(photonGenesis)
}
