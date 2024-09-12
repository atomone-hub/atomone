package atomone_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"

	atomone "github.com/atomone-hub/atomone/app"
	atomonehelpers "github.com/atomone-hub/atomone/app/helpers"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
)

type EmptyAppOptions struct{}

func (ao EmptyAppOptions) Get(_ string) interface{} {
	return nil
}

func TestAtomOneApp_BlockedModuleAccountAddrs(t *testing.T) {
	encConfig := atomone.RegisterEncodingConfig()
	app := atomone.NewAtomOneApp(
		log.NewNopLogger(),
		db.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		atomone.DefaultNodeHome,
		encConfig,
		EmptyAppOptions{},
	)

	moduleAccountAddresses := app.ModuleAccountAddrs()
	blockedAddrs := app.BlockedModuleAccountAddrs(moduleAccountAddresses)

	require.NotContains(t, blockedAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())
}

func TestAtomOneApp_Export(t *testing.T) {
	app := atomonehelpers.Setup(t)
	// TODO: not all modules are registered with a StoreKey!
	_, err := app.ExportAppStateAndValidators(true, []string{}, []string{"auth"})
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}
