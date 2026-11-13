package v5_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/atomone-hub/atomone/app/helpers"
	"github.com/atomone-hub/atomone/app/upgrades"
	v5 "github.com/atomone-hub/atomone/app/upgrades/v5"
	"github.com/atomone-hub/atomone/modules/spectre"
)

// TestAppendSpectreToAllowedClients verifies the spectre client type is
// appended to the AllowedClients param, that the operation is idempotent,
// and that the allow-all wildcard is left untouched.
func TestAppendSpectreToAllowedClients(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{})

	// Start from an explicit list without spectre.
	app.IBCKeeper.ClientKeeper.SetParams(ctx, ibcclienttypes.NewParams(exported.Tendermint))
	require.False(t, app.IBCKeeper.ClientKeeper.GetParams(ctx).IsAllowedClient(spectre.ModuleName))

	require.NoError(t, upgrades.AppendAllowedClient(ctx, app.IBCKeeper, spectre.ModuleName))
	params := app.IBCKeeper.ClientKeeper.GetParams(ctx)
	require.True(t, params.IsAllowedClient(spectre.ModuleName))
	require.Equal(t, []string{exported.Tendermint, spectre.ModuleName}, params.AllowedClients)

	// Idempotent: a second call does not duplicate the entry.
	require.NoError(t, upgrades.AppendAllowedClient(ctx, app.IBCKeeper, spectre.ModuleName))
	params = app.IBCKeeper.ClientKeeper.GetParams(ctx)
	require.Len(t, params.AllowedClients, 2)

	// The allow-all wildcard already permits spectre; it must stay untouched.
	app.IBCKeeper.ClientKeeper.SetParams(ctx, ibcclienttypes.NewParams(ibcclienttypes.AllowAllClients))
	require.NoError(t, upgrades.AppendAllowedClient(ctx, app.IBCKeeper, spectre.ModuleName))
	params = app.IBCKeeper.ClientKeeper.GetParams(ctx)
	require.Equal(t, []string{ibcclienttypes.AllowAllClients}, params.AllowedClients)
}

// TestCreateUpgradeHandler verifies the v5 handler runs the module
// migrations and registers the spectre client type end to end.
func TestCreateUpgradeHandler(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{})

	// Start from an explicit list without spectre, as mainnet will have it.
	app.IBCKeeper.ClientKeeper.SetParams(ctx, ibcclienttypes.NewParams(exported.Tendermint))

	handler := v5.CreateUpgradeHandler(app.ModuleManager(), app.AppCodec(), app.Configurator(), &app.AppKeepers)
	_, err := handler(ctx, upgradetypes.Plan{Name: v5.UpgradeName}, app.ModuleManager().GetVersionMap())
	require.NoError(t, err)

	require.True(t, app.IBCKeeper.ClientKeeper.GetParams(ctx).IsAllowedClient(spectre.ModuleName),
		"the handler registers the spectre client type")
}
