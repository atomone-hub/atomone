package simulation

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/x/coredaos/types"
)

const (
	VotingPeriodExtensionsLimit   = "voting_period_extensions_limit"
	VotingPeriodExtensionDuration = "voting_period_extension_duration"
	SteeringDaoAddress            = "steering_dao_address"
	OversightDaoAddress           = "steering_dao_address"
	DAOAccountsNumber             = 10
)

// DAO addresses need to be separated from the other simulation account
// otherwise the staking module simulation could use the account for
// staking tokens
var (
	SteeringDaoAccount  simulation.Account
	OversightDaoAccount simulation.Account
	DAOAccounts         []simulation.Account
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

// GenSteeringDaoAddress picks a random address to be used for a DAO
// with a probability of 50%, otherwise returns an empty string account (meaning that
// the Dao is disabled
func GenSteeringDaoAddress(r *rand.Rand, simState *module.SimulationState) string {
	randInt := r.Intn(2)
	if randInt%2 == 0 {
		randInt := r.Intn(DAOAccountsNumber)
		SteeringDaoAccount = DAOAccounts[randInt]
		address := SteeringDaoAccount.Address.String()
		return address
	} else {
		return ""
	}
}

// GenSteeringDaoAddress picks a random address to be used for a DAO
// with a probability of 50%, otherwise returns an empty string account (meaning that
// the Dao is disabled
func GenOversightDaoAddress(r *rand.Rand, simState *module.SimulationState) string {
	randInt := r.Intn(2)
	if randInt%2 == 0 {
		randInt := r.Intn(DAOAccountsNumber)
		OversightDaoAccount = DAOAccounts[randInt]
		address := OversightDaoAccount.Address.String()
		return address
	} else {
		return ""
	}
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	GenDaoAccounts(simState.Rand, simState)
	var votingPeriodExtensionsLimit uint32
	simState.AppParams.GetOrGenerate(
		simState.Cdc, VotingPeriodExtensionsLimit, &votingPeriodExtensionsLimit, simState.Rand,
		func(r *rand.Rand) { votingPeriodExtensionsLimit = GenVotingPeriodExtensionsLimit(r) },
	)
	var votingPeriodExtensionDuration time.Duration
	simState.AppParams.GetOrGenerate(
		simState.Cdc, VotingPeriodExtensionDuration, &votingPeriodExtensionDuration, simState.Rand,
		func(r *rand.Rand) { votingPeriodExtensionDuration = GenVotingPeriodExtensionDuration(r) },
	)

	var steeringDaoAddress string
	simState.AppParams.GetOrGenerate(
		simState.Cdc, SteeringDaoAddress, &steeringDaoAddress, simState.Rand,
		func(r *rand.Rand) { steeringDaoAddress = GenSteeringDaoAddress(r, simState) },
	)

	var oversightDaoAddress string
	simState.AppParams.GetOrGenerate(
		simState.Cdc, OversightDaoAddress, &oversightDaoAddress, simState.Rand,
		func(r *rand.Rand) { oversightDaoAddress = GenOversightDaoAddress(r, simState) },
	)

	coredaosGenesis := types.NewGenesisState(
		types.NewParams(
			steeringDaoAddress,
			oversightDaoAddress,
			votingPeriodExtensionsLimit,
			votingPeriodExtensionDuration,
		),
	)
	bz, err := json.MarshalIndent(&coredaosGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated coredaos parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(coredaosGenesis)
}

func GenDaoAccounts(r *rand.Rand, simState *module.SimulationState) {
	DAOAccounts = simulation.RandomAccounts(r, DAOAccountsNumber)
	genesisAccs := make([]authtypes.GenesisAccount, len(simState.Accounts)+DAOAccountsNumber)
	for i, defaultAccs := range simState.Accounts {
		genesisAccs[i] = authtypes.NewBaseAccountWithAddress(defaultAccs.Address)
	}
	for i, daoAccount := range DAOAccounts {
		genesisAccs[i+len(simState.Accounts)] = authtypes.NewBaseAccountWithAddress(daoAccount.Address)
	}
	// Build auth genesis
	authGenesis := authtypes.NewGenesisState(
		authtypes.DefaultParams(),
		genesisAccs,
	)

	// Encode into simState
	bz, err := simState.Cdc.MarshalJSON(authGenesis)
	if err != nil {
		panic(err)
	}
	simState.GenState[authtypes.ModuleName] = bz
}
