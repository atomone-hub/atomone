package dynamicfee

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/atomone-hub/atomone/x/dynamicfee/types"
	"github.com/atomone-hub/atomone/x/gov/simulation"
)

// GenerateGenesisState returns a disabled dynamicfee module because the module
// does not work well with simulations. Especially the dynamicfee ante handler
// does not accept 0 fee coins which is quite common during simulation's
// operations.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	params := types.DefaultParams()
	params.Enabled = false
	genesis := types.NewGenesisState(params, types.DefaultState())
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(genesis)
	fmt.Println("Dynamicfee module is disabled")
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simulation.NewDecodeStore(am.cdc)
}

// WeightedOperations returns the all the module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return nil
}
