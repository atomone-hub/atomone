package simulation

import (
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/atomone-hub/atomone/x/coredaos/types"
)

const (
	VotingPeriodExtensionsLimit   = "voting_period_extensions_limit"
	VotingPeriodExtensionDuration = "voting_period_extension_duration"
)

// GenVotingPeriodExtensionsLimit generates a random voting period extensions limit
func GenVotingPeriodExtensionsLimit(r *rand.Rand) uint32 {
	return uint32(r.Intn(10)) // Random limit between 0 and 9
}

// GenVotingPeriodExtensionDuration generates a random voting period extension duration
// The duration is between 1 second and 6 hours
func GenVotingPeriodExtensionDuration(r *rand.Rand) time.Duration {
	return time.Duration(simulation.RandIntBetween(r, 1, 60*60*6)) * time.Second
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	var votingPeriodExtensionsLimit uint32
	simState.AppParams.GetOrGenerate(
		VotingPeriodExtensionsLimit, &votingPeriodExtensionsLimit, simState.Rand,
		func(r *rand.Rand) { votingPeriodExtensionsLimit = GenVotingPeriodExtensionsLimit(r) },
	)
	var votingPeriodExtensionDuration time.Duration
	simState.AppParams.GetOrGenerate(
		VotingPeriodExtensionDuration, &votingPeriodExtensionDuration, simState.Rand,
		func(r *rand.Rand) { votingPeriodExtensionDuration = GenVotingPeriodExtensionDuration(r) },
	)

	coredaosGenesis := types.NewGenesisState(
		types.NewParams(
			"", // Steering DAO address, empty means disabled, TODO: FIXME if possible
			"", // Oversight DAO address, empty means disabled, TODO: FIXME if possible
			votingPeriodExtensionsLimit,
			votingPeriodExtensionDuration,
		),
	)

	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(coredaosGenesis)
}
