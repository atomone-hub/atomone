package spectre

import (
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

// setClientState stores the client state.
func setClientState(clientStore storetypes.KVStore, cdc codec.BinaryCodec, clientState *ClientState) {
	clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(cdc, clientState))
}

// getClientState retrieves the client state from the store.
func getClientState(store storetypes.KVStore, cdc codec.BinaryCodec) (*ClientState, bool) {
	bz := store.Get(host.ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}

	clientStateI := clienttypes.MustUnmarshalClientState(cdc, bz)
	clientState, ok := clientStateI.(*ClientState)
	if !ok {
		panic(fmt.Errorf("cannot convert %T into %T", clientStateI, clientState))
	}
	return clientState, true
}

// setConsensusState stores the consensus state at the given height.
func setConsensusState(clientStore storetypes.KVStore, cdc codec.BinaryCodec, consensusState *ConsensusState, height exported.Height) {
	clientStore.Set(host.ConsensusStateKey(height), clienttypes.MustMarshalConsensusState(cdc, consensusState))
}

// GetConsensusState retrieves the consensus state at the given height from
// the client prefixed store.
func GetConsensusState(store storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height) (*ConsensusState, bool) {
	bz := store.Get(host.ConsensusStateKey(height))
	if len(bz) == 0 {
		return nil, false
	}

	consensusStateI := clienttypes.MustUnmarshalConsensusState(cdc, bz)
	consensusState, ok := consensusStateI.(*ConsensusState)
	if !ok {
		panic(fmt.Errorf("cannot convert %T into %T", consensusStateI, consensusState))
	}

	return consensusState, true
}
