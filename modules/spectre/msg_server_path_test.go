package spectre_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/app/helpers"
	"github.com/atomone-hub/atomone/modules/spectre"
)

// ctxForHeader returns a context whose block time is shortly after the
// header's attested execution timestamp.
func ctxForHeader(ctx sdk.Context, h *spectre.Header) sdk.Context {
	ts := h.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000
	return sdk.NewContext(ctx.MultiStore(), cmtproto.Header{Time: time.Unix(int64(ts), 0)}, false, log.NewNopLogger())
}

// cloneHeader returns a deep copy of a header.
func cloneHeader(t *testing.T, h *spectre.Header) *spectre.Header {
	t.Helper()
	bz, err := h.Marshal()
	require.NoError(t, err)
	var out spectre.Header
	require.NoError(t, out.Unmarshal(bz))
	return &out
}

// TestMsgUpdateClientThroughKeeper drives client updates and a misbehaviour
// submission through the real 02-client keeper, the path MsgUpdateClient
// takes, plus the direct UpdateState-with-misbehaviour call the
// RedundantRelayDecorator makes during CheckTx. The misbehaviour is built
// from fixture data and tampered (a fully valid one needs the devnet
// committee's keys to sign conflicting roots), so it fails verification
// cleanly; the updateState non-header early return itself is covered by
// TestUpdateStateWithMisbehaviourReturnsNoHeights.
func TestMsgUpdateClientThroughKeeper(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{})

	app.IBCKeeper.ClientKeeper.SetParams(ctx, ibcclienttypes.NewParams(ibcclienttypes.AllowAllClients))

	csBz, consBz, headers := spectre.ExportLoadFixtureBootstrap(t, "Test_TimeoutPacketFromCosmos")
	require.NotEmpty(t, headers)

	clientID, err := app.IBCKeeper.ClientKeeper.CreateClient(ctx, spectre.ModuleName, csBz, consBz)
	require.NoError(t, err)

	// The first fixture header applies through the keeper, like MsgUpdateClient.
	h := headers[0]
	require.NoError(t, app.IBCKeeper.ClientKeeper.UpdateClient(ctxForHeader(ctx, h), clientID, h))
	require.Equal(t, h.ConsensusUpdate.FinalizedHeader.Beacon.Slot,
		app.IBCKeeper.ClientKeeper.GetClientLatestHeight(ctx, clientID).GetRevisionHeight())

	// A misbehaviour whose second update is tampered fails verification
	// cleanly through the same path.
	other := cloneHeader(t, h)
	other.ConsensusUpdate.AttestedHeader.Execution.StateRoot[0] ^= 0xff
	mis := &spectre.Misbehaviour{
		TrustedSlot:         h.TrustedSlot,
		ActiveSyncCommittee: h.ActiveSyncCommittee,
		IsNextCommittee:     h.IsNextCommittee,
		Update_1:            h.ConsensusUpdate,
		Update_2:            other.ConsensusUpdate,
	}
	require.NotPanics(t, func() {
		err = app.IBCKeeper.ClientKeeper.UpdateClient(ctxForHeader(ctx, h), clientID, mis)
	})
	require.Error(t, err, "the tampered update fails signature verification")
	require.Equal(t, exported.Active, app.IBCKeeper.ClientKeeper.GetClientStatus(ctx, clientID),
		"a failed misbehaviour does not freeze the client")

	// The RedundantRelayDecorator reaches UpdateState with the misbehaviour
	// directly in CheckTx, without the misbehaviour checks of the keeper
	// path. That call panicked before the non-header early return; the
	// unit-level coverage is TestUpdateStateWithMisbehaviourReturnsNoHeights.
	clientModule, err := app.IBCKeeper.ClientKeeper.Route(ctx, clientID)
	require.NoError(t, err)
	var heights []exported.Height
	require.NotPanics(t, func() {
		heights = clientModule.UpdateState(ctxForHeader(ctx, h), clientID, mis)
	})
	require.Empty(t, heights, "a misbehaviour applies no state")
	require.Equal(t, exported.Active, app.IBCKeeper.ClientKeeper.GetClientStatus(ctx, clientID))
}
