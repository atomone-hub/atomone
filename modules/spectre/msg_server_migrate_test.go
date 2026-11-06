package spectre_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/app/helpers"
	"github.com/atomone-hub/atomone/modules/spectre"
)

// TestMigrateForkParameters drives the governance message through the real
// app: a spectre client created with Gloas unscheduled can learn the announced
// Gloas epoch while staying active, and nothing else about the client changes.
func TestMigrateForkParameters(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{})
	app.IBCKeeper.ClientKeeper.SetParams(ctx, ibcclienttypes.NewParams(ibcclienttypes.AllowAllClients))

	csBz, consBz, headers := spectre.ExportLoadFixtureBootstrap(t, "Test_TimeoutPacketFromCosmos")
	require.NotEmpty(t, headers)

	clientID, err := app.IBCKeeper.ClientKeeper.CreateClient(ctx, spectre.ModuleName, csBz, consBz)
	require.NoError(t, err)

	// The fixture schedule is whatever the devnet ran; read it back through
	// the store and apply an updated schedule through the message.
	msgServer := spectre.NewMsgServer(spectre.NewKeeper(
		app.AppCodec(), app.IBCKeeper.ClientKeeper.GetStoreProvider(),
		authtypes.NewModuleAddress("gov").String()))

	var cs spectre.ClientState
	require.NoError(t, cs.Unmarshal(csBz))
	updated := cs.ForkParameters.Clone()
	// The fixture devnet schedules forks at degenerate epochs (all at
	// genesis, or unscheduled), so changing an epoch can wrap or collide;
	// flipping the Gloas version changes the schedule under every layout.
	updated.Gloas.Version[0] ^= 0xff

	authority := authtypes.NewModuleAddress("gov").String()
	msg := &spectre.MsgMigrateForkParameters{
		Authority:      authority,
		ClientId:       clientID,
		ForkParameters: *updated,
	}

	_, err = msgServer.MigrateForkParameters(ctx, msg)
	require.NoError(t, err)

	// The client is active under the new schedule, and nothing else moved.
	state, found := app.IBCKeeper.ClientKeeper.GetClientState(ctx, clientID)
	require.True(t, found)
	spectreCS, ok := state.(*spectre.ClientState)
	require.True(t, ok)
	require.Equal(t, updated.Gloas.Version, spectreCS.ForkParameters.Gloas.Version)
	require.Equal(t, cs.LatestSlot, spectreCS.LatestSlot)
	require.Equal(t, cs.ChainId, spectreCS.ChainId)
	require.Equal(t, exported.Active, app.IBCKeeper.ClientKeeper.GetClientStatus(ctx, clientID))

	t.Run("wrong authority is rejected", func(t *testing.T) {
		_, err := msgServer.MigrateForkParameters(ctx, &spectre.MsgMigrateForkParameters{
			Authority:      "atone1wrongwrongwrongwrongwrongwrongwrongwrong",
			ClientId:       clientID,
			ForkParameters: *updated,
		})
		require.Error(t, err)
	})

	t.Run("an unchanged schedule is rejected", func(t *testing.T) {
		_, err := msgServer.MigrateForkParameters(ctx, &spectre.MsgMigrateForkParameters{
			Authority:      authority,
			ClientId:       clientID,
			ForkParameters: spectreCS.ForkParameters,
		})
		require.ErrorContains(t, err, "do not change")
	})

	t.Run("a changed genesis version is rejected", func(t *testing.T) {
		bad := spectreCS.ForkParameters.Clone()
		bad.GenesisForkVersion[0] ^= 0xff
		_, err := msgServer.MigrateForkParameters(ctx, &spectre.MsgMigrateForkParameters{
			Authority:      authority,
			ClientId:       clientID,
			ForkParameters: *bad,
		})
		require.ErrorContains(t, err, "genesis fork version")
	})

	t.Run("an invalid schedule is rejected", func(t *testing.T) {
		bad := spectreCS.ForkParameters.Clone()
		bad.Electra.Version = nil // wrong length; validation rejects it
		_, err := msgServer.MigrateForkParameters(ctx, &spectre.MsgMigrateForkParameters{
			Authority:      authority,
			ClientId:       clientID,
			ForkParameters: *bad,
		})
		require.ErrorContains(t, err, "validate")
	})

	t.Run("an unknown client is rejected", func(t *testing.T) {
		_, err := msgServer.MigrateForkParameters(ctx, &spectre.MsgMigrateForkParameters{
			Authority:      authority,
			ClientId:       "spectre-999",
			ForkParameters: *updated,
		})
		require.Error(t, err)
	})
}
