package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/atomone-hub/atomone/x/photon/types"
)

const (
	MintDisabled    = "mint_disabled"
	TxFeeExceptions = "tx_fee_exceptions"
)

// GenMintDisabled returns a randomized MintDisabled param.
func GenMintDisabled(r *rand.Rand) bool {
	return r.Int63n(101) <= 15 // 15% chance of mint being disabled
}

// GenTxFeeExceptions returns a wildcard to allow all transactions to use any
// fee denom.
// This is needed because other modules' simulations do not allow the fee coins
// to be changed, so w/o this configuration all transactions would fail.
func GenTxFeeExceptions(r *rand.Rand) []string {
	return []string{"*"}
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	var mintDisabled bool
	simState.AppParams.GetOrGenerate(
		simState.Cdc, MintDisabled, &mintDisabled, simState.Rand,
		func(r *rand.Rand) { mintDisabled = GenMintDisabled(r) },
	)
	var txFeeExceptions []string
	simState.AppParams.GetOrGenerate(
		simState.Cdc, TxFeeExceptions, &txFeeExceptions, simState.Rand,
		func(r *rand.Rand) { txFeeExceptions = GenTxFeeExceptions(r) },
	)

	photonGenesis := types.NewGenesisState(
		types.NewParams(mintDisabled, txFeeExceptions),
	)

	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(photonGenesis)
}
