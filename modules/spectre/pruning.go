package spectre

import (
	"context"
	"encoding/binary"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

// PruneKeepDuration is how long consensus states are kept below the latest
// finalized slot before pruning. Membership verification rejects proof
// heights below this window, so pruning at the window can never strand an
// in-flight packet: any packet still provable at a height remains provable
// after pruning. The window must stay above the largest packet delay period
// of any application routed over spectre clients.
const PruneKeepDuration = 7 * 24 * time.Hour

// pruneBatchLimit bounds the deletions of one pruning run, so a single
// update transaction does bounded extra work no matter how far behind
// pruning is.
const pruneBatchLimit = 1000

// keyIterateConsensusStatePrefix is the prefix of an ordered index over the
// consensus states, mirroring 07-tendermint's iteration keys. The consensus
// state keys themselves are decimal-height strings, so their lexicographic
// order is not slot order; the index stores one entry per state under a
// big-endian slot, in which byte order is slot order. Pruning iterates the
// index and stops at the first state it keeps, instead of scanning the whole
// window on every update.
const keyIterateConsensusStatePrefix = "iterateConsensusStates"

// IterationKey returns the ordered-index key of a consensus state slot. The
// index is internal to the module but exported for migration tooling that
// prepares pre-index state.
func IterationKey(slot uint64) []byte {
	key := make([]byte, len(keyIterateConsensusStatePrefix)+8)
	copy(key, keyIterateConsensusStatePrefix)
	binary.BigEndian.PutUint64(key[len(keyIterateConsensusStatePrefix):], slot)
	return key
}

// PruneKeepSlots returns the pruning window in slots, derived from the
// client's slot duration. A zero SecondsPerSlot yields a zero window: only
// the latest state is kept.
func (cs *ClientState) PruneKeepSlots() uint64 {
	if cs.SecondsPerSlot == 0 {
		return 0
	}
	return uint64(PruneKeepDuration/time.Second) / cs.SecondsPerSlot
}

// PruneConsensusStates deletes the client's consensus states at finalized
// slots older than keepSlots below the latest finalized slot, at most
// maxPrune per call so a single invocation does bounded work. The consensus
// state at the latest slot is always kept. Slots are read from the ordered
// index keys, so consensus states are never unmarshaled and the scan stops at
// the first kept slot: an in-window client reads one index entry per stale
// state instead of the whole window.
//
// keepSlots should come from PruneKeepSlots: membership verification rejects
// heights below that window, so states below it are unreachable by proofs.
// If more than maxPrune states are stale, callers repeat the call.
func PruneConsensusStates(ctx context.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, keepSlots uint64, maxPrune int) (int, error) {
	if maxPrune <= 0 {
		return 0, errorsmod.Wrap(ErrInvalidPruning, "maxPrune must be positive")
	}

	clientState, found := getClientState(ctx, clientStore, cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, "client store has no client state")
	}

	// Keep the window [latest-keepSlots, latest]; with keepSlots == 0 only
	// the latest state survives.
	cutoff := uint64(0)
	if latest := clientState.LatestSlot; latest > keepSlots {
		cutoff = latest - keepSlots
	}

	prefix := []byte(keyIterateConsensusStatePrefix)
	iterator := clientStore.Iterator(prefix, storetypes.PrefixEndBytes(prefix))

	// Collect first, delete after: mutations during iteration are unsafe on
	// some store implementations.
	type staleEntry struct {
		slot uint64
		key  []byte // consensus state key, stored as the index value
	}
	var stale []staleEntry
	for ; iterator.Valid(); iterator.Next() {
		slot := binary.BigEndian.Uint64(iterator.Key()[len(prefix):])
		if slot >= cutoff {
			break // index order is slot order: everything from here is kept
		}
		if len(stale) == maxPrune {
			break
		}
		stale = append(stale, staleEntry{slot: slot, key: append([]byte(nil), iterator.Value()...)})
	}
	iterator.Close()

	for _, entry := range stale {
		clientStore.Delete(entry.key)
		clientStore.Delete(IterationKey(entry.slot))
	}
	return len(stale), nil
}
