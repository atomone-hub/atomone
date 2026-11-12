package migrations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/modules/spectre"
)

// newTestCodec returns a codec with the spectre types registered.
func newTestCodec(t *testing.T) codec.BinaryCodec {
	t.Helper()
	ir := codectypes.NewInterfaceRegistry()
	spectre.RegisterInterfaces(ir)
	return codec.NewProtoCodec(ir)
}

// fakeClientKeeper serves the migrations ClientKeeper interface over a
// single client store.
type fakeClientKeeper struct {
	state *spectre.ClientState
	store storetypes.KVStore
}

func (f fakeClientKeeper) IterateClientStates(_ sdk.Context, _ []byte, cb func(string, exported.ClientState) bool) {
	cb("spectre-0", f.state)
}

func (f fakeClientKeeper) ClientStore(_ sdk.Context, _ string) storetypes.KVStore {
	return f.store
}

func (f fakeClientKeeper) Logger(sdk.Context) log.Logger {
	return log.NewNopLogger()
}

// TestMigrationPruneConsensusStates verifies the migration helper prunes at
// the module window with bounded work per call.
func TestMigrationPruneConsensusStates(t *testing.T) {
	cdc := newTestCodec(t)

	// A client whose 7-day window is 5 slots, with states at slots
	// 100..110 below latest 110.
	cs := &spectre.ClientState{
		SecondsPerSlot: uint64(spectre.PruneKeepDuration / (5 * time.Second)),
		LatestSlot:     110,
	}
	require.Equal(t, uint64(5), cs.PruneKeepSlots())

	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey("spectre-migrations-test")
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	clientStore := ms.GetKVStore(storeKey)

	clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(cdc, cs))
	for slot := uint64(100); slot <= 110; slot++ {
		key := host.ConsensusStateKey(clienttypes.NewHeight(0, slot))
		clientStore.Set(key, clienttypes.MustMarshalConsensusState(cdc, &spectre.ConsensusState{Slot: slot}))
		clientStore.Set(spectre.IterationKey(slot), key)
	}

	keeper := fakeClientKeeper{state: cs, store: clientStore}
	ctx := sdk.NewContext(ms, cmtproto.Header{}, false, log.NewNopLogger())

	// The bound limits a single call; repeating reaches the fixed point.
	pruned, err := PruneConsensusStates(ctx, cdc, keeper, 2)
	require.NoError(t, err)
	require.Equal(t, 2, pruned)

	total := pruned
	for {
		pruned, err = PruneConsensusStates(ctx, cdc, keeper, 2)
		require.NoError(t, err)
		total += pruned
		if pruned == 0 {
			break
		}
	}
	require.Equal(t, 5, total, "slots 100..104 are below 110-5")

	// Slots within the window and the latest state survive.
	for slot := uint64(105); slot <= 110; slot++ {
		require.NotNil(t, clientStore.Get(host.ConsensusStateKey(clienttypes.NewHeight(0, slot))), "slot %d is kept", slot)
	}
}
