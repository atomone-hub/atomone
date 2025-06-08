package atomone_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	simulation2 "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"

	atomone "github.com/atomone-hub/atomone/app"
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

	app := atomone.NewAtomOneApp(
		logger,
		db,
		nil,
		true,
		appOptions,
		interBlockCacheOpt(),
		baseapp.SetChainID(AppChainID),
	)

	// Run randomized simulation:w
	_, simParams, simErr := simulation.SimulateFromSeed(
		b,
		os.Stdout,
		app.BaseApp,
		simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()),
		simulation2.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
		simtestutil.SimulationOperations(app, app.AppCodec(), config),
		app.ModuleAccountAddrs(),
		config,
		app.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
	if err = simtestutil.CheckExportSimulation(app, config, simParams); err != nil {
		b.Fatal(err)
	}

	if simErr != nil {
		b.Fatal(simErr)
	}

	if config.Commit {
		simtestutil.PrintStats(db)
	}
}
