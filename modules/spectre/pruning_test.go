package spectre

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// storeClientWithStates initializes a client at latestSlot with a consensus
// state for every slot in [firstSlot, latestSlot].
func storeClientWithStates(t *testing.T, firstSlot, latestSlot uint64) storetypes.KVStore {
	t.Helper()

	cdc := getTestCodec()
	clientStore := setupClientStore(t)

	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	initial := rawJSON(fixture.Steps[0].Data)
	cs := decodeClientState(initial["client_state"])
	consState := decodeConsensusState(initial["consensus_state"])

	cs.LatestSlot = latestSlot
	setClientState(context.Background(), clientStore, cdc, cs)
	for slot := firstSlot; slot <= latestSlot; slot++ {
		st := *consState
		st.Slot = slot
		setConsensusState(context.Background(), clientStore, cdc, &st, clienttypes.NewHeight(0, slot))
	}
	return clientStore
}

// TestPruneConsensusStates verifies that pruning removes consensus states
// older than the kept window, always keeps the latest state, and does
// bounded work per call.
func TestPruneConsensusStates(t *testing.T) {
	cdc := getTestCodec()

	t.Run("keeps the recent window", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		pruned, err := PruneConsensusStates(context.Background(), clientStore, cdc, 5, 100)
		require.NoError(t, err)
		require.Equal(t, 5, pruned, "slots 100..104 are pruned")

		for slot := uint64(100); slot < 105; slot++ {
			_, found := GetConsensusState(context.Background(), clientStore, cdc, clienttypes.NewHeight(0, slot))
			require.False(t, found, "slot %d should be pruned", slot)
		}
		for slot := uint64(105); slot <= 110; slot++ {
			_, found := GetConsensusState(context.Background(), clientStore, cdc, clienttypes.NewHeight(0, slot))
			require.True(t, found, "slot %d should be kept", slot)
		}
	})

	t.Run("keepSlots zero keeps only the latest", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		pruned, err := PruneConsensusStates(context.Background(), clientStore, cdc, 0, 100)
		require.NoError(t, err)
		require.Equal(t, 10, pruned)

		_, found := GetConsensusState(context.Background(), clientStore, cdc, clienttypes.NewHeight(0, 110))
		require.True(t, found, "the latest state is always kept")
		_, found = GetConsensusState(context.Background(), clientStore, cdc, clienttypes.NewHeight(0, 109))
		require.False(t, found)
	})

	t.Run("window covering genesis keeps everything", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		pruned, err := PruneConsensusStates(context.Background(), clientStore, cdc, 110, 100)
		require.NoError(t, err)
		require.Zero(t, pruned)
	})

	t.Run("maxPrune bounds deletions per call", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		pruned, err := PruneConsensusStates(context.Background(), clientStore, cdc, 0, 3)
		require.NoError(t, err)
		require.Equal(t, 3, pruned)

		// Repeating the call eventually reaches a fixed point: only the
		// latest state survives.
		total := pruned
		for {
			pruned, err = PruneConsensusStates(context.Background(), clientStore, cdc, 0, 3)
			require.NoError(t, err)
			total += pruned
			if pruned == 0 {
				break
			}
		}
		require.Equal(t, 10, total)
		_, found := GetConsensusState(context.Background(), clientStore, cdc, clienttypes.NewHeight(0, 110))
		require.True(t, found, "the latest state is always kept")
	})

	t.Run("non-positive maxPrune errors", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		_, err := PruneConsensusStates(context.Background(), clientStore, cdc, 5, 0)
		require.ErrorIs(t, err, ErrInvalidPruning)
	})

	t.Run("pruned states lose their index entry too", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		pruned, err := PruneConsensusStates(context.Background(), clientStore, cdc, 5, 100)
		require.NoError(t, err)
		require.Equal(t, 5, pruned)

		for slot := uint64(100); slot < 105; slot++ {
			require.Nil(t, clientStore.Get(IterationKey(slot)), "index entry for slot %d is pruned", slot)
		}
		for slot := uint64(105); slot <= 110; slot++ {
			require.NotNil(t, clientStore.Get(IterationKey(slot)), "index entry for slot %d is kept", slot)
		}
	})

	t.Run("an unindexed state is not pruned", func(t *testing.T) {
		// The index is the source of iteration; a state written without one
		// (as by a pre-index binary) is invisible to pruning.
		clientStore := storeClientWithStates(t, 100, 110)
		clientStore.Delete(IterationKey(100))

		pruned, err := PruneConsensusStates(context.Background(), clientStore, cdc, 0, 100)
		require.NoError(t, err)
		require.Equal(t, 9, pruned)
		_, found := GetConsensusState(context.Background(), clientStore, cdc, clienttypes.NewHeight(0, 100))
		require.True(t, found, "the unindexed state is left alone")
	})

	t.Run("missing client state errors", func(t *testing.T) {
		clientStore := setupClientStore(t)

		_, err := PruneConsensusStates(context.Background(), clientStore, cdc, 5, 100)
		require.Error(t, err)
	})
}

// TestPruneOnUpdate verifies pruning runs as part of applying a header
// update, and is skipped in CheckTx, like the tendermint client.
func TestPruneOnUpdate(t *testing.T) {
	cdc := getTestCodec()
	const (
		staleSlot     = 5               // below the window of the advanced latest slot
		latestSlot    = 605000          // window of 604800 slots: cutoff lands at 1020
		finalizedSlot = latestSlot + 10 // same sync period as latestSlot
	)

	// A client whose application path needs no signature verification: the
	// header is hand-crafted, and updateState only applies it.
	newStore := func(t *testing.T) (storetypes.KVStore, *ClientState) {
		t.Helper()
		store := setupClientStore(t)
		cs := &ClientState{
			LatestSlot:                   latestSlot,
			SecondsPerSlot:               12,
			SlotsPerEpoch:                32,
			EpochsPerSyncCommitteePeriod: 8,
			SyncCommitteeSize:            512,
		}
		setClientState(context.Background(), store, cdc, cs)
		for _, slot := range []uint64{staleSlot, latestSlot} {
			setConsensusState(context.Background(), store, cdc, &ConsensusState{
				Slot:      slot,
				StateRoot: make([]byte, 32),
			}, clienttypes.NewHeight(0, slot))
		}
		return store, cs
	}

	header := &Header{
		TrustedSlot: latestSlot,
		ConsensusUpdate: LightClientUpdate{
			FinalizedHeader: LightClientHeader{
				Beacon:    BeaconBlockHeader{Slot: finalizedSlot},
				Execution: &ExecutionPayloadHeader{StateRoot: make([]byte, 32)},
			},
		},
	}

	t.Run("prunes below the window", func(t *testing.T) {
		store, cs := newStore(t)
		ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())

		heights := cs.updateState(ctx, cdc, store, header)
		require.Len(t, heights, 1)
		require.EqualValues(t, finalizedSlot, heights[0].GetRevisionHeight())

		_, found := GetConsensusState(ctx, store, cdc, clienttypes.NewHeight(0, staleSlot))
		require.False(t, found, "the state below the window is pruned with the update")
		_, found = GetConsensusState(ctx, store, cdc, clienttypes.NewHeight(0, finalizedSlot))
		require.True(t, found, "the new state is stored")
	})

	t.Run("skips pruning in CheckTx", func(t *testing.T) {
		store, cs := newStore(t)
		ctx := sdk.NewContext(nil, cmtproto.Header{}, true, log.NewNopLogger())

		heights := cs.updateState(ctx, cdc, store, header)
		require.Len(t, heights, 1)

		_, found := GetConsensusState(ctx, store, cdc, clienttypes.NewHeight(0, staleSlot))
		require.True(t, found, "pruning is skipped in CheckTx")
	})
}

// TestPruneKeepSlots verifies the window derivation from the slot duration.
func TestPruneKeepSlots(t *testing.T) {
	require.Equal(t, uint64(7*24*3600/12), (&ClientState{SecondsPerSlot: 12}).PruneKeepSlots())
	require.Equal(t, uint64(7*24*3600/6), (&ClientState{SecondsPerSlot: 6}).PruneKeepSlots())
	// A corrupt slot duration yields the narrowest safe window instead of
	// dividing by zero.
	require.Equal(t, uint64(0), (&ClientState{}).PruneKeepSlots())
}
