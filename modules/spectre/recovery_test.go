package spectre

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	storetypes "cosmossdk.io/store/types"
)

// consensusStoresEqual reports whether two client stores hold the same
// consensus states.
func consensusStoresEqual(t *testing.T, a, b storetypes.KVStore) bool {
	t.Helper()
	collect := func(store storetypes.KVStore) []byte {
		var parts [][]byte
		iterator := store.Iterator([]byte("consensusStates/"), storetypes.PrefixEndBytes([]byte("consensusStates/")))
		for ; iterator.Valid(); iterator.Next() {
			parts = append(parts, iterator.Key(), iterator.Value())
		}
		iterator.Close()
		return bytes.Join(parts, []byte("\n"))
	}
	return bytes.Equal(collect(a), collect(b))
}

// bootstrapBytes returns the marshaled initial client and consensus states of
// the fixture, optionally mutated, for initializing a second client on the
// same store provider.
func bootstrapBytes(t *testing.T, fixtureName string, mut func(*ClientState)) (clientStateBz, consensusStateBz []byte) {
	t.Helper()
	fixture := loadFixture(t, fixtureName)
	initial := rawJSON(fixture.Steps[0].Data)
	cs := decodeClientState(initial["client_state"])
	cons := decodeConsensusState(initial["consensus_state"])
	if mut != nil {
		mut(cs)
	}
	clientStateBz, err := cs.Marshal()
	require.NoError(t, err)
	consensusStateBz, err = cons.Marshal()
	require.NoError(t, err)
	return clientStateBz, consensusStateBz
}

// freezeClient marks the stored client state of clientID as frozen.
func freezeClient(t *testing.T, c *fixtureClient, clientID string) {
	t.Helper()
	cdc := getTestCodec()
	store := c.provider.ClientStore(c.ctx, clientID)
	cs, found := getClientState(store, cdc)
	require.True(t, found)
	cs.Frozen = true
	setClientState(store, cdc, cs)
}

// TestRecoverClient verifies governance client recovery: a frozen subject is
// unfrozen from an active, further-along substitute with matching chain
// parameters, keeping its identifier and its own in-window consensus states.
func TestRecoverClient(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	cdc := getTestCodec()
	const substituteID = "spectre-1"
	require.GreaterOrEqual(t, len(c.headers), 2)

	csBz, consBz := bootstrapBytes(t, parityFixture, nil)
	require.NoError(t, c.module.Initialize(c.ctx, substituteID, csBz, consBz))

	// The subject advances once and freezes; the substitute keeps going.
	c.apply(0)
	for i, h := range []*Header{c.headers[0], c.headers[1]} {
		require.NoError(t, c.module.VerifyClientMessage(c.ctxAt(h), substituteID, h), "substitute header %d", i)
		c.module.UpdateState(c.ctxAt(h), substituteID, h)
	}
	substituteLatest := c.module.LatestHeight(c.ctx, substituteID).GetRevisionHeight()
	subjectLatest := c.latestSlot()
	require.Greater(t, substituteLatest, subjectLatest)

	freezeClient(t, c, c.clientID)
	require.Equal(t, exported.Frozen, c.module.Status(c.ctx, c.clientID))

	require.NoError(t, c.module.RecoverClient(c.ctx, c.clientID, substituteID))

	// The subject is active again, at the substitute's latest slot, under its
	// own identifier.
	require.Equal(t, exported.Active, c.module.Status(c.ctx, c.clientID))
	require.Equal(t, substituteLatest, c.latestSlot())

	subjectStore := c.provider.ClientStore(c.ctx, c.clientID)
	substituteStore := c.provider.ClientStore(c.ctx, substituteID)
	_, found := GetConsensusState(subjectStore, cdc, clienttypes.NewHeight(0, substituteLatest))
	require.True(t, found, "the substitute's latest consensus state is copied")

	// Every state in the subject's store now comes from the substitute: the
	// subject's own pre-freeze states are dropped, and only the substitute's
	// (replayed through a healthy client) remain, even where slots overlap.
	require.True(t, consensusStoresEqual(t, subjectStore, substituteStore),
		"the subject's consensus states are exactly the substitute's")

	// The recovered client is no longer frozen at verification: a stale
	// anchor is rejected for the anchor, not for the freeze.
	stale := cloneHeader(t, c.headers[1])
	err := c.module.VerifyClientMessage(c.ctxAt(stale), c.clientID, stale)
	require.ErrorIs(t, err, ErrTrustedSlotNotLatest)
}

// TestRecoverClientRejections covers the guard rails of recovery.
func TestRecoverClientRejections(t *testing.T) {
	c := newFixtureClient(t, parityFixture)
	c.apply(0)

	t.Run("substitute of another client type", func(t *testing.T) {
		err := c.module.RecoverClient(c.ctx, c.clientID, "07-tendermint-0")
		require.ErrorIs(t, err, clienttypes.ErrInvalidClientType)
	})

	t.Run("missing substitute client", func(t *testing.T) {
		err := c.module.RecoverClient(c.ctx, c.clientID, "spectre-99")
		require.ErrorIs(t, err, clienttypes.ErrClientNotFound)
	})

	t.Run("missing subject client", func(t *testing.T) {
		err := c.module.RecoverClient(c.ctx, "spectre-98", c.clientID)
		require.ErrorIs(t, err, clienttypes.ErrClientNotFound)
	})

	t.Run("subject is not frozen", func(t *testing.T) {
		// A fresh subject at the bootstrap slot: the frozen check fires before
		// any height comparison, so no advancing is needed.
		csBz, consBz := bootstrapBytes(t, parityFixture, nil)
		require.NoError(t, c.module.Initialize(c.ctx, "spectre-90", csBz, consBz))
		err := c.module.RecoverClient(c.ctx, "spectre-90", "spectre-1")
		require.Error(t, err)
	})
	csBz, consBz := bootstrapBytes(t, parityFixture, nil)
	require.NoError(t, c.module.Initialize(c.ctx, "spectre-1", csBz, consBz))

	freezeClient(t, c, c.clientID)

	t.Run("substitute does not advance the subject", func(t *testing.T) {
		csBz, consBz := bootstrapBytes(t, parityFixture, nil)
		require.NoError(t, c.module.Initialize(c.ctx, "spectre-2", csBz, consBz))
		err := c.module.RecoverClient(c.ctx, c.clientID, "spectre-2")
		require.ErrorIs(t, err, clienttypes.ErrInvalidHeight)
	})

	t.Run("chain parameters differ", func(t *testing.T) {
		csBz, consBz := bootstrapBytes(t, parityFixture, func(cs *ClientState) { cs.ChainId++ })
		require.NoError(t, c.module.Initialize(c.ctx, "spectre-3", csBz, consBz))
		for _, h := range []*Header{c.headers[0], c.headers[1]} {
			require.NoError(t, c.module.VerifyClientMessage(c.ctxAt(h), "spectre-3", h))
			c.module.UpdateState(c.ctxAt(h), "spectre-3", h)
		}
		err := c.module.RecoverClient(c.ctx, c.clientID, "spectre-3")
		require.ErrorIs(t, err, clienttypes.ErrInvalidSubstitute)

		// The subject is untouched by the failed recovery.
		require.Equal(t, exported.Frozen, c.module.Status(c.ctx, c.clientID))
	})
}

// TestIsMatchingClientState verifies which client state fields recovery
// ignores and which it requires to match.
func TestIsMatchingClientState(t *testing.T) {
	csBz, _ := bootstrapBytes(t, parityFixture, nil)
	var base ClientState
	require.NoError(t, base.Unmarshal(csBz))

	mutated := base.Clone()
	mutated.Frozen = true
	mutated.LatestSlot += 1_000_000
	mutated.LatestExecutionBlockNumber += 42
	require.True(t, isMatchingClientState(&base, mutated), "progress fields are ignored")

	for name, mut := range map[string]func(*ClientState){
		"chain id":             func(cs *ClientState) { cs.ChainId++ },
		"genesis slot":         func(cs *ClientState) { cs.GenesisSlot++ },
		"seconds per slot":     func(cs *ClientState) { cs.SecondsPerSlot++ },
		"committee size":       func(cs *ClientState) { cs.SyncCommitteeSize++ },
		"genesis validators":   func(cs *ClientState) { cs.GenesisValidatorsRoot[0] ^= 0xff },
		"fork version":         func(cs *ClientState) { cs.ForkParameters.Electra.Version[0] ^= 0xff },
		"ibc contract address": func(cs *ClientState) { cs.IbcContractAddress[0] ^= 0xff },
		"ibc commitment slot":  func(cs *ClientState) { cs.IbcCommitmentSlot[0] ^= 0xff },
	} {
		t.Run(name, func(t *testing.T) {
			differing := base.Clone()
			mut(differing)
			require.False(t, isMatchingClientState(&base, differing))
		})
	}
}
