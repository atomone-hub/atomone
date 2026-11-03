package spectre

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// fakeClientKeeper serves the spectre module's ClientKeeper interface over a
// single client store.
type fakeClientKeeper struct {
	store storetypes.KVStore
}

func (f fakeClientKeeper) IterateClientStates(_ sdk.Context, _ []byte, cb func(string, exported.ClientState) bool) {
	clientState, found := getClientState(f.store, getTestCodec())
	if !found {
		return
	}
	cb("spectre-0", clientState)
}

func (f fakeClientKeeper) ClientStore(_ sdk.Context, _ string) storetypes.KVStore {
	return f.store
}

// TestEndBlockPrunesBelowWindow verifies the endblocker prunes only on its
// interval and only below the module window.
func TestEndBlockPrunesBelowWindow(t *testing.T) {
	cdc := getTestCodec()

	newModule := func(t *testing.T) (AppModule, storetypes.KVStore) {
		t.Helper()
		// States at slots 100..110. An inflated slot duration shrinks the
		// 7-day window to 7 slots, so everything below 110-7=103 is stale.
		store := storeClientWithStates(t, 100, 110)
		cs, found := getClientState(store, cdc)
		require.True(t, found)
		cs.SecondsPerSlot = uint64(PruneKeepDuration / (7 * time.Second))
		setClientState(store, cdc, cs)
		return NewAppModule(cdc, NewLightClientModule(cdc, clienttypes.StoreProvider{}), fakeClientKeeper{store: store}), store
	}

	ctxAt := func(height int64) context.Context {
		return sdk.NewContext(nil, cmtproto.Header{Height: height}, false, log.NewNopLogger())
	}

	t.Run("off-interval blocks do not prune", func(t *testing.T) {
		module, store := newModule(t)
		require.NoError(t, module.EndBlock(ctxAt(101)))
		_, found := GetConsensusState(store, cdc, clienttypes.NewHeight(0, 100))
		require.True(t, found)
	})

	t.Run("on-interval blocks prune below the window", func(t *testing.T) {
		module, store := newModule(t)
		require.NoError(t, module.EndBlock(ctxAt(pruneIntervalBlocks)))
		for slot := uint64(100); slot < 103; slot++ {
			_, found := GetConsensusState(store, cdc, clienttypes.NewHeight(0, slot))
			require.False(t, found, "slot %d is below the window", slot)
		}
		for slot := uint64(103); slot <= 110; slot++ {
			_, found := GetConsensusState(store, cdc, clienttypes.NewHeight(0, slot))
			require.True(t, found, "slot %d is within the window", slot)
		}
	})
}
