package spectre

import (
	"bytes"
	"strconv"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"

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
// state at the latest slot is always kept. Slots are read from the store
// keys, so consensus states are never unmarshaled.
//
// The keys are decimal-height strings, so iteration is lexicographic rather
// than by slot and the scan cannot stop at the cutoff: it visits every state
// the client still holds, which the pruning window bounds, and deletes at
// most maxPrune of them.
//
// keepSlots should come from PruneKeepSlots: membership verification rejects
// heights below that window, so states below it are unreachable by proofs.
// If more than maxPrune states are stale, callers repeat the call.
func PruneConsensusStates(clientStore storetypes.KVStore, cdc codec.BinaryCodec, keepSlots uint64, maxPrune int) (int, error) {
	if maxPrune <= 0 {
		return 0, errorsmod.Wrap(ErrInvalidPruning, "maxPrune must be positive")
	}

	clientState, found := getClientState(clientStore, cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, "client store has no client state")
	}

	// Keep the window [latest-keepSlots, latest]; with keepSlots == 0 only
	// the latest state survives.
	cutoff := uint64(0)
	if latest := clientState.LatestSlot; latest > keepSlots {
		cutoff = latest - keepSlots
	}

	prefix := []byte(host.KeyConsensusStatePrefix + "/")
	iterator := clientStore.Iterator(prefix, storetypes.PrefixEndBytes(prefix))

	// Collect first, delete after: mutations during iteration are unsafe on
	// some store implementations.
	var staleKeys [][]byte
	var iterErr error
	for ; iterator.Valid(); iterator.Next() {
		if len(staleKeys) == maxPrune {
			break
		}
		slot, err := slotFromConsensusStateKey(iterator.Key())
		if err != nil {
			iterErr = err
			break
		}
		if slot < cutoff {
			staleKeys = append(staleKeys, iterator.Key())
		}
	}
	iterator.Close()
	if iterErr != nil {
		return 0, iterErr
	}

	for _, key := range staleKeys {
		clientStore.Delete(key)
	}
	return len(staleKeys), nil
}

// slotFromConsensusStateKey parses the finalized slot from a consensus state
// store key, whose suffix is the height string "<revision>-<slot>"
// (host.ConsensusStateKey). Keys that do not encode a height are corrupt.
func slotFromConsensusStateKey(key []byte) (uint64, error) {
	dash := bytes.LastIndexByte(key, '-')
	if dash < 0 {
		return 0, errorsmod.Wrapf(ErrInvalidConsensusState, "consensus state key %X does not encode a height", key)
	}
	slot, err := strconv.ParseUint(string(key[dash+1:]), 10, 64)
	if err != nil {
		return 0, errorsmod.Wrapf(ErrInvalidConsensusState, "consensus state key %X does not encode a height", key)
	}
	return slot, nil
}
