package spectre

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"

	storetypes "cosmossdk.io/store/types"
)

// TestStoreLayoutMatchesCoreIBC verifies the collections-backed store uses
// exactly the keys and values core IBC reads when it iterates clients: the
// client state at the single key "clientState" and consensus states under
// "consensusStates/<revision>-<slot>", both as protobuf Anys. Core IBC's
// GetAllClientStates and GetAllConsensusStates parse these directly, so the
// layout is a compatibility contract, not an implementation detail.
func TestStoreLayoutMatchesCoreIBC(t *testing.T) {
	cdc := getTestCodec()
	ctx := context.Background()
	store := setupClientStore(t)

	csBz, consBz, _ := ExportLoadFixtureBootstrap(t, "Test_TimeoutPacketFromCosmos")
	var cs ClientState
	require.NoError(t, cs.Unmarshal(csBz))
	var cons ConsensusState
	require.NoError(t, cons.Unmarshal(consBz))

	setClientState(ctx, store, cdc, &cs)
	setConsensusState(ctx, store, cdc, &cons, clienttypes.NewHeight(0, cons.Slot))

	// The client state sits at exactly host.ClientStateKey(), as an Any.
	rawClientState := store.Get(host.ClientStateKey())
	require.NotEmpty(t, rawClientState)
	decoded, err := clienttypes.UnmarshalClientState(cdc, rawClientState)
	require.NoError(t, err)
	require.IsType(t, &ClientState{}, decoded)

	// The consensus state sits at exactly host.ConsensusStateKey(height),
	// as an Any.
	height := clienttypes.NewHeight(0, cons.Slot)
	rawConsensus := store.Get(host.ConsensusStateKey(height))
	require.NotEmpty(t, rawConsensus)
	decodedCons, err := clienttypes.UnmarshalConsensusState(cdc, rawConsensus)
	require.NoError(t, err)
	require.IsType(t, &ConsensusState{}, decodedCons)

	// A state written through the raw core IBC layout reads back through
	// the collections helpers, and vice versa.
	height2 := clienttypes.NewHeight(0, cons.Slot+1)
	cons2 := cons
	cons2.Slot++
	store.Set(host.ConsensusStateKey(height2), clienttypes.MustMarshalConsensusState(cdc, &cons2))
	got, found := GetConsensusState(ctx, store, cdc, height2)
	require.True(t, found)
	require.Equal(t, cons2.Slot, got.Slot)

	// The prefix iteration core IBC uses to list consensus states sees the
	// entries.
	count := 0
	prefix := []byte(host.KeyConsensusStatePrefix + "/")
	iterator := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	iterator.Close()
	require.Equal(t, 2, count)
}
