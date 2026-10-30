package spectre

import (
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	storetypes "cosmossdk.io/store/types"
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
	setClientState(clientStore, cdc, cs)
	for slot := firstSlot; slot <= latestSlot; slot++ {
		st := *consState
		st.Slot = slot
		setConsensusState(clientStore, cdc, &st, clienttypes.NewHeight(0, slot))
	}
	return clientStore
}

// TestPruneConsensusStates verifies that pruning removes consensus states
// older than the kept window and always keeps the latest state.
func TestPruneConsensusStates(t *testing.T) {
	cdc := getTestCodec()

	t.Run("keeps the recent window", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		pruned, err := PruneConsensusStates(clientStore, cdc, 5)
		require.NoError(t, err)
		require.Equal(t, 5, pruned, "slots 100..104 are pruned")

		for slot := uint64(100); slot < 105; slot++ {
			_, found := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, slot))
			require.False(t, found, "slot %d should be pruned", slot)
		}
		for slot := uint64(105); slot <= 110; slot++ {
			_, found := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, slot))
			require.True(t, found, "slot %d should be kept", slot)
		}
	})

	t.Run("keepSlots zero keeps only the latest", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		pruned, err := PruneConsensusStates(clientStore, cdc, 0)
		require.NoError(t, err)
		require.Equal(t, 10, pruned)

		_, found := GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, 110))
		require.True(t, found, "the latest state is always kept")
		_, found = GetConsensusState(clientStore, cdc, clienttypes.NewHeight(0, 109))
		require.False(t, found)
	})

	t.Run("window covering genesis keeps everything", func(t *testing.T) {
		clientStore := storeClientWithStates(t, 100, 110)

		pruned, err := PruneConsensusStates(clientStore, cdc, 110)
		require.NoError(t, err)
		require.Zero(t, pruned)
	})

	t.Run("missing client state errors", func(t *testing.T) {
		clientStore := setupClientStore(t)

		_, err := PruneConsensusStates(clientStore, cdc, 5)
		require.Error(t, err)
	})
}
