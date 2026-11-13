package spectre

import (
	"context"
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"cosmossdk.io/collections"
	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

// clientKVStoreService adapts a client-prefixed legacy KVStore to the
// core store KVStoreService that collections builds on, so the client and
// consensus states can be managed with collections. The store is already
// scoped to one client, so the context is not needed to resolve it. The
// wrapping mirrors the sdk runtime's unexported coreKVStore adapter.
type clientKVStoreService struct {
	store storetypes.KVStore
}

// OpenKVStore implements the core store KVStoreService interface.
func (s clientKVStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return coreKVStore{kvStore: s.store}
}

type coreKVStore struct {
	kvStore storetypes.KVStore
}

func (store coreKVStore) Get(key []byte) ([]byte, error) {
	return store.kvStore.Get(key), nil
}

func (store coreKVStore) Has(key []byte) (bool, error) {
	return store.kvStore.Has(key), nil
}

func (store coreKVStore) Set(key, value []byte) error {
	store.kvStore.Set(key, value)
	return nil
}

func (store coreKVStore) Delete(key []byte) error {
	store.kvStore.Delete(key)
	return nil
}

func (store coreKVStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	return store.kvStore.Iterator(start, end), nil
}

func (store coreKVStore) ReverseIterator(start, end []byte) (corestore.Iterator, error) {
	return store.kvStore.ReverseIterator(start, end), nil
}

// clientStateCodec encodes a spectre ClientState exactly as core IBC
// expects client state values in a client store: a protobuf Any.
type clientStateCodec struct {
	cdc codec.BinaryCodec
}

func (c clientStateCodec) Encode(value *ClientState) ([]byte, error) {
	return clienttypes.MarshalClientState(c.cdc, value)
}

func (c clientStateCodec) Decode(b []byte) (*ClientState, error) {
	state, err := clienttypes.UnmarshalClientState(c.cdc, b)
	if err != nil {
		return nil, err
	}
	cs, ok := state.(*ClientState)
	if !ok {
		return nil, fmt.Errorf("cannot convert %T into %T", state, &ClientState{})
	}
	return cs, nil
}

func (c clientStateCodec) EncodeJSON(*ClientState) ([]byte, error) {
	return nil, fmt.Errorf("json encoding of the spectre client state is not supported")
}

func (c clientStateCodec) DecodeJSON([]byte) (*ClientState, error) {
	return nil, fmt.Errorf("json decoding of the spectre client state is not supported")
}

func (c clientStateCodec) Stringify(value *ClientState) string {
	return fmt.Sprintf("spectre client state (latest slot %d)", value.LatestSlot)
}

func (c clientStateCodec) ValueType() string {
	return "spectre/ClientState"
}

// consensusStateCodec encodes a spectre ConsensusState exactly as core IBC
// expects consensus state values in a client store: a protobuf Any.
type consensusStateCodec struct {
	cdc codec.BinaryCodec
}

func (c consensusStateCodec) Encode(value *ConsensusState) ([]byte, error) {
	return clienttypes.MarshalConsensusState(c.cdc, value)
}

func (c consensusStateCodec) Decode(b []byte) (*ConsensusState, error) {
	state, err := clienttypes.UnmarshalConsensusState(c.cdc, b)
	if err != nil {
		return nil, err
	}
	cs, ok := state.(*ConsensusState)
	if !ok {
		return nil, fmt.Errorf("cannot convert %T into %T", state, &ConsensusState{})
	}
	return cs, nil
}

func (c consensusStateCodec) EncodeJSON(*ConsensusState) ([]byte, error) {
	return nil, fmt.Errorf("json encoding of the spectre consensus state is not supported")
}

func (c consensusStateCodec) DecodeJSON([]byte) (*ConsensusState, error) {
	return nil, fmt.Errorf("json decoding of the spectre consensus state is not supported")
}

func (c consensusStateCodec) Stringify(value *ConsensusState) string {
	return fmt.Sprintf("spectre consensus state (slot %d)", value.Slot)
}

func (c consensusStateCodec) ValueType() string {
	return "spectre/ConsensusState"
}

// newClientStateItem returns the client state entry of a client store. The
// key is the single byte string "clientState" that core IBC reads when it
// iterates all clients, so it must not change.
func newClientStateItem(clientStore storetypes.KVStore, cdc codec.BinaryCodec) collections.Item[*ClientState] {
	sb := collections.NewSchemaBuilder(clientKVStoreService{store: clientStore})
	return collections.NewItem(sb, collections.NewPrefix(host.KeyClientState), "client_state", clientStateCodec{cdc: cdc})
}

// newConsensusStateMap returns the consensus state map of a client store.
// The keys are "consensusStates/<revision>-<slot>" strings, the format core
// IBC reads when it iterates consensus states, so it must not change.
func newConsensusStateMap(clientStore storetypes.KVStore, cdc codec.BinaryCodec) collections.Map[string, *ConsensusState] {
	sb := collections.NewSchemaBuilder(clientKVStoreService{store: clientStore})
	return collections.NewMap(sb, collections.NewPrefix(host.KeyConsensusStatePrefix+"/"), "consensus_states", collections.StringKey, consensusStateCodec{cdc: cdc})
}

// setClientState stores the client state.
func setClientState(ctx context.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, clientState *ClientState) {
	if err := newClientStateItem(clientStore, cdc).Set(ctx, clientState); err != nil {
		panic(errorsmod.Wrap(err, "set client state"))
	}
}

// getClientState retrieves the client state from the store.
func getClientState(ctx context.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec) (*ClientState, bool) {
	clientState, err := newClientStateItem(clientStore, cdc).Get(ctx)
	if err != nil {
		if errorsmod.IsOf(err, collections.ErrNotFound) {
			return nil, false
		}
		panic(errorsmod.Wrap(err, "get client state"))
	}
	return clientState, true
}

// setConsensusState stores the consensus state at the given height, plus
// its ordered-index entry for pruning.
func setConsensusState(ctx context.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, consensusState *ConsensusState, height exported.Height) {
	if err := newConsensusStateMap(clientStore, cdc).Set(ctx, height.String(), consensusState); err != nil {
		panic(errorsmod.Wrap(err, "set consensus state"))
	}
	clientStore.Set(IterationKey(height.GetRevisionHeight()), host.ConsensusStateKey(height))
}

// GetConsensusState retrieves the consensus state at the given height from
// the client prefixed store.
func GetConsensusState(ctx context.Context, clientStore storetypes.KVStore, cdc codec.BinaryCodec, height exported.Height) (*ConsensusState, bool) {
	consensusState, err := newConsensusStateMap(clientStore, cdc).Get(ctx, height.String())
	if err != nil {
		if errorsmod.IsOf(err, collections.ErrNotFound) {
			return nil, false
		}
		panic(errorsmod.Wrap(err, "get consensus state"))
	}
	return consensusState, true
}
