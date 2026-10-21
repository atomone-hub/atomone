package spectre

import (
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

// PruneConsensusStates deletes the client's consensus states at finalized
// slots older than keepSlots below the latest finalized slot. The consensus
// state at the latest slot is always kept.
//
// In-flight packets verify against the consensus state at their proof height;
// callers must only prune heights no packet still references.
func PruneConsensusStates(clientStore storetypes.KVStore, cdc codec.BinaryCodec, keepSlots uint64) (int, error) {
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
	defer iterator.Close()

	var staleKeys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		consState, err := clienttypes.UnmarshalConsensusState(cdc, iterator.Value())
		if err != nil {
			return 0, errorsmod.Wrapf(ErrInvalidConsensusState, "unmarshal consensus state at key %X", iterator.Key())
		}
		spectreState, ok := consState.(*ConsensusState)
		if !ok {
			return 0, errorsmod.Wrapf(ErrInvalidConsensusState, "client store holds a %T consensus state", consState)
		}
		if spectreState.Slot < cutoff {
			staleKeys = append(staleKeys, iterator.Key())
		}
	}

	for _, key := range staleKeys {
		clientStore.Delete(key)
	}
	return len(staleKeys), nil
}
