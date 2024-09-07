package staking_test

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	simtestutil "github.com/atomone-hub/atomone/testutil/sims"
	authKeeper "github.com/atomone-hub/atomone/x/auth/keeper"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	"github.com/atomone-hub/atomone/x/staking/testutil"
	"github.com/atomone-hub/atomone/x/staking/types"
)

func TestItCreatesModuleAccountOnInitBlock(t *testing.T) {
	var accountKeeper authKeeper.AccountKeeper
	app, err := simtestutil.SetupAtGenesis(testutil.AppConfig, &accountKeeper)
	require.NoError(t, err)

	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	acc := accountKeeper.GetAccount(ctx, authtypes.NewModuleAddress(types.BondedPoolName))
	require.NotNil(t, acc)

	acc = accountKeeper.GetAccount(ctx, authtypes.NewModuleAddress(types.NotBondedPoolName))
	require.NotNil(t, acc)
}
