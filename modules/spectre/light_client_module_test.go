package spectre

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// getTestCodec returns a codec for testing.
func getTestCodec() codec.BinaryCodec {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

// setupTestApp creates a commit multi store and context with a fixed block
// time for light client module tests.
func setupTestApp(t *testing.T, blockTime time.Time) (sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()

	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey("spectre-test")
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, cmtproto.Header{Time: blockTime}, false, log.NewNopLogger())
	return ctx, storeKey
}

// TestLightClientModuleLifecycle drives the full light client module
// interface against the fixture data: initialize, verify, update, and
// membership verification through the store round-trip.
func TestLightClientModuleLifecycle(t *testing.T) {
	cdc := getTestCodec()

	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	initial := rawJSON(fixture.Steps[0].Data)
	cs := decodeClientState(initial["client_state"])
	consState := decodeConsensusState(initial["consensus_state"])

	ctx, storeKey := setupTestApp(t, time.Unix(1, 0))
	module := NewLightClientModule(cdc, clienttypes.NewStoreProvider(runtime.NewKVStoreService(storeKey)))
	const clientID = "spectre-0"

	// Initialize receives the raw marshaled types (the Any value bytes).
	clientStateBz, err := cs.Marshal()
	require.NoError(t, err)
	consensusStateBz, err := consState.Marshal()
	require.NoError(t, err)
	require.NoError(t, module.Initialize(ctx, clientID, clientStateBz, consensusStateBz))

	// Status of a fresh client is active.
	require.Equal(t, exported.Active, module.Status(ctx, clientID))

	// Latest height maps the latest slot.
	height := module.LatestHeight(ctx, clientID)
	require.Equal(t, cs.LatestSlot, height.GetRevisionHeight())

	// Timestamp at height reads the consensus state timestamp in ns.
	ts, err := module.TimestampAtHeight(ctx, clientID, clienttypes.NewHeight(0, consState.Slot))
	require.NoError(t, err)
	require.Equal(t, consState.Timestamp*1_000_000_000, ts)

	// Drive the update pipeline through the module interface.
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)
	require.NotEmpty(t, msgs.headers)

	for i, fh := range msgs.headers {
		header := fh.toProto()
		currentTime := header.ConsensusUpdate.AttestedHeader.Execution.Timestamp + 1000
		updateCtx := sdk.NewContext(ctx.MultiStore(), cmtproto.Header{Time: time.Unix(int64(currentTime), 0)}, false, log.NewNopLogger())

		require.NoError(t, module.VerifyClientMessage(updateCtx, clientID, header), "header %d", i)
		require.False(t, module.CheckForMisbehaviour(updateCtx, clientID, header))
		heights := module.UpdateState(updateCtx, clientID, header)
		require.Len(t, heights, 1)
		require.Equal(t, header.ConsensusUpdate.FinalizedHeader.Beacon.Slot, heights[0].GetRevisionHeight())
	}

	// The timeout proof verifies through the module interface.
	require.NotEmpty(t, msgs.timeout)
	for i, to := range msgs.timeout {
		proofCtx := sdk.NewContext(ctx.MultiStore(), cmtproto.Header{Time: time.Unix(int64(to.ProofHeight.GetRevisionHeight()), 0)}, false, log.NewNopLogger())
		proofBz, err := cdc.Marshal(decodeMembershipProof(to.ProofUnreceived))
		require.NoError(t, err)
		path := commitmenttypesv2.MerklePath{
			KeyPath: [][]byte{hostv2.PacketReceiptKey(to.Packet.DestinationClient, to.Packet.Sequence)},
		}
		require.NoError(t, module.VerifyNonMembership(
			proofCtx, clientID, to.ProofHeight, 0, 0, proofBz, path,
		), "timeout %d", i)
	}
}
