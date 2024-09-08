package atomone_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"

	"cosmossdk.io/math"

	"github.com/atomone-hub/atomone/baseapp"
	"github.com/atomone-hub/atomone/client/flags"
	"github.com/atomone-hub/atomone/server"
	"github.com/atomone-hub/atomone/store"
	simtestutil "github.com/atomone-hub/atomone/testutil/sims"
	simulation2 "github.com/atomone-hub/atomone/types/simulation"
	"github.com/atomone-hub/atomone/x/simulation"
	simcli "github.com/atomone-hub/atomone/x/simulation/client/cli"

	"github.com/atomone-hub/atomone/ante"
	atomone "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/app/sim"
)

// AppChainID hardcoded chainID for simulation
const AppChainID = "atomone-app"

func init() {
	sim.GetSimulatorFlags()
}

// interBlockCacheOpt returns a BaseApp option function that sets the persistent
// inter-block write-through cache.
func interBlockCacheOpt() func(*baseapp.BaseApp) {
	return baseapp.SetInterBlockCache(store.NewCommitKVStoreCacheManager())
}

// TODO: Make another test for the fuzzer itself, which just has noOp txs
// and doesn't depend on the application.
func TestAppStateDeterminism(t *testing.T) {
	if !sim.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	config := sim.NewConfigFromFlags()
	config.InitialBlockHeight = 1
	config.ExportParamsPath = ""
	config.OnOperation = false
	config.AllInvariants = false
	config.ChainID = AppChainID

	numSeeds := 3
	numTimesToRunPerSeed := 5

	// We will be overriding the random seed and just run a single simulation on the provided seed value
	if config.Seed != simcli.DefaultSeedValue {
		numSeeds = 1
	}

	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = atomone.DefaultNodeHome
	appOptions[server.FlagInvCheckPeriod] = sim.FlagPeriodValue

	for i := 0; i < numSeeds; i++ {
		if config.Seed == simcli.DefaultSeedValue {
			config.Seed = rand.Int63()
		}

		fmt.Println("config.Seed: ", config.Seed)

		for j := 0; j < numTimesToRunPerSeed; j++ {
			var logger log.Logger
			if sim.FlagVerboseValue {
				logger = log.TestingLogger()
			} else {
				logger = log.NewNopLogger()
			}

			db := dbm.NewMemDB()
			encConfig := atomone.RegisterEncodingConfig()
			app := atomone.NewAtomOneApp(
				logger,
				db,
				nil,
				true,
				map[int64]bool{},
				atomone.DefaultNodeHome,
				encConfig,
				appOptions,
				interBlockCacheOpt(),
				baseapp.SetChainID(AppChainID),
			)

			// NOTE: setting to zero to avoid failing the simulation
			// due to the minimum staked tokens required to submit a vote
			ante.SetMinStakedTokens(math.LegacyZeroDec())

			fmt.Printf(
				"running non-determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
			)

			blockedAddresses := app.BlockedModuleAccountAddrs(app.ModuleAccountAddrs())

			_, _, err := simulation.SimulateFromSeed(
				t,
				os.Stdout,
				app.BaseApp,
				simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), atomone.NewDefaultGenesisState(encConfig)),
				simulation2.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
				simtestutil.SimulationOperations(app, app.AppCodec(), config),
				blockedAddresses,
				config,
				app.AppCodec(),
			)
			require.NoError(t, err)

			if config.Commit {
				sim.PrintStats(db)
			}

			appHash := app.LastCommitID().Hash
			appHashList[j] = appHash

			if j != 0 {
				require.Equal(
					t, string(appHashList[0]), string(appHashList[j]),
					"non-determinism in seed %d: %d/%d, attempt: %d/%d\n", config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
				)
			}
		}
	}
}
