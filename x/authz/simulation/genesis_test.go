package simulation_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	"github.com/atomone-hub/atomone/types/module"
	simtypes "github.com/atomone-hub/atomone/types/simulation"
	"github.com/atomone-hub/atomone/x/authz"
	authzmodule "github.com/atomone-hub/atomone/x/authz/module"
	"github.com/atomone-hub/atomone/x/authz/simulation"
	banktypes "github.com/atomone-hub/atomone/x/bank/types"

	moduletestutil "github.com/atomone-hub/atomone/types/module/testutil"
)

func TestRandomizedGenState(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig(authzmodule.AppModuleBasic{})
	banktypes.RegisterInterfaces(encCfg.InterfaceRegistry)

	s := rand.NewSource(1)
	r := rand.New(s)

	simState := module.SimulationState{
		AppParams:    make(simtypes.AppParams),
		Cdc:          encCfg.Codec,
		Rand:         r,
		NumBonded:    3,
		Accounts:     simtypes.RandomAccounts(r, 3),
		InitialStake: sdkmath.NewInt(1000),
		GenState:     make(map[string]json.RawMessage),
	}

	simulation.RandomizedGenState(&simState)
	var authzGenesis authz.GenesisState
	simState.Cdc.MustUnmarshalJSON(simState.GenState[authz.ModuleName], &authzGenesis)

	require.Len(t, authzGenesis.Authorization, len(simState.Accounts)-1)
}
