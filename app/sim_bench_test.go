package atomone_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/baseapp"
	"github.com/atomone-hub/atomone/server"
	simtestutil "github.com/atomone-hub/atomone/testutil/sims"
	simulation2 "github.com/atomone-hub/atomone/types/simulation"
	"github.com/atomone-hub/atomone/x/simulation"
	simcli "github.com/atomone-hub/atomone/x/simulation/client/cli"

	atomone "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/app/sim"
)

// Profile with:
// /usr/local/go/bin/go test -benchmem -run=^$ github.com/cosmos/cosmos-sdk/AtomOneApp -bench ^BenchmarkFullAppSimulation$ -Commit=true -cpuprofile cpu.out
func BenchmarkFullAppSimulation(b *testing.B) {
	b.ReportAllocs()

	config := simcli.NewConfigFromFlags()
	config.ChainID = AppChainID

	db, dir, logger, skip, err := simtestutil.SetupSimulation(config, "goleveldb-app-sim", "Simulation", simcli.FlagVerboseValue, simcli.FlagEnabledValue)
	if err != nil {
		b.Fatalf("simulation setup failed: %s", err.Error())
	}

	if skip {
		b.Skip("skipping benchmark application simulation")
	}

	defer func() {
		require.NoError(b, db.Close())
		require.NoError(b, os.RemoveAll(dir))
	}()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

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

	// Run randomized simulation:w
	_, simParams, simErr := simulation.SimulateFromSeed(
		b,
		os.Stdout,
		app.BaseApp,
		sim.AppStateFn(encConfig, app.SimulationManager()),
		simulation2.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
		sim.SimulationOperations(app, app.AppCodec(), config),
		app.ModuleAccountAddrs(),
		config,
		app.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
	if err = sim.CheckExportSimulation(app, config, simParams); err != nil {
		b.Fatal(err)
	}

	if simErr != nil {
		b.Fatal(simErr)
	}

	if config.Commit {
		sim.PrintStats(db)
	}
}
