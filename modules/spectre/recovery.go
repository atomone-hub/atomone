package spectre

import (
	"reflect"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

// recoverClient copies the verified state of an active substitute client into
// the frozen subject client's store, unfreezing it under its own identifier.
// Governance authorizes the recovery through MsgRecoverClient; this function
// enforces that the substitute tracks the same Ethereum bridge, so only the
// progress (latest slot, execution block number, consensus states) is taken
// over, never the chain parameters.
func (cs *ClientState) recoverClient(
	cdc codec.BinaryCodec,
	subjectStore, substituteStore storetypes.KVStore,
	substitute *ClientState,
) error {
	if !cs.Frozen {
		return errorsmod.Wrap(clienttypes.ErrInvalidClient, "client is not frozen: nothing to recover")
	}
	if substitute.LatestSlot <= cs.LatestSlot {
		return errorsmod.Wrapf(clienttypes.ErrInvalidHeight,
			"substitute latest slot %d does not advance subject latest slot %d", substitute.LatestSlot, cs.LatestSlot)
	}
	if !isMatchingClientState(cs, substitute) {
		return errorsmod.Wrap(clienttypes.ErrInvalidSubstitute, "subject and substitute client states differ in chain parameters")
	}

	// The substitute's latest state must exist; recovery without it would
	// leave the client unable to verify its next update.
	if _, found := GetConsensusState(substituteStore, cdc, substitute.GetLatestHeight()); !found {
		return errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "substitute has no consensus state at its latest slot")
	}

	// Copy the substitute's consensus states over the subject's. The
	// consensus state at a given finalized slot is a function of Ethereum
	// history, so overlapping slots carry identical values; the subject's own
	// states are kept, which preserves in-window proof heights from before
	// the freeze.
	prefix := []byte(host.KeyConsensusStatePrefix + "/")
	iterator := substituteStore.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	for ; iterator.Valid(); iterator.Next() {
		subjectStore.Set(iterator.Key(), iterator.Value())
	}
	iterator.Close()

	recovered := cs.Clone()
	recovered.Frozen = false
	recovered.LatestSlot = substitute.LatestSlot
	recovered.LatestExecutionBlockNumber = substitute.LatestExecutionBlockNumber
	setClientState(subjectStore, cdc, recovered)
	return nil
}

// isMatchingClientState reports whether subject and substitute track the same
// chain and bridge: every field except the mutable progress (frozen flag,
// latest slot, latest execution block number) must be equal.
func isMatchingClientState(subject, substitute *ClientState) bool {
	normalize := func(cs *ClientState) ClientState {
		out := *cs
		out.Frozen = false
		out.LatestSlot = 0
		out.LatestExecutionBlockNumber = 0
		return out
	}
	return reflect.DeepEqual(normalize(subject), normalize(substitute))
}
