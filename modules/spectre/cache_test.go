package spectre

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLruCacheBehavior covers get/put, LRU ordering and eviction.
func TestLruCacheBehavior(t *testing.T) {
	c := newLruCache(2)

	put := func(b byte, v byte) { c.put([32]byte{b}, [blsPubkeySize]byte{v}) }
	get := func(b byte) ([blsPubkeySize]byte, bool) { return c.get([32]byte{b}) }

	put(1, 0x11)
	put(2, 0x22)
	v, ok := get(1)
	require.True(t, ok)
	require.Equal(t, [blsPubkeySize]byte{0x11}, v)

	// Key 1 was just used, so inserting a third key evicts key 2.
	put(3, 0x33)
	_, ok = get(2)
	require.False(t, ok, "the least recently used key is evicted")
	_, ok = get(1)
	require.True(t, ok)
	_, ok = get(3)
	require.True(t, ok)

	// Re-putting an existing key updates its value.
	put(1, 0x99)
	v, ok = get(1)
	require.True(t, ok)
	require.Equal(t, [blsPubkeySize]byte{0x99}, v)

	// A miss returns the zero value and false.
	v, ok = get(0xAA)
	require.False(t, ok)
	require.Equal(t, [blsPubkeySize]byte{}, v)
}

// TestLruCacheConcurrentAccess hammers the cache from concurrent goroutines;
// run with -race to validate the locking.
func TestLruCacheConcurrentAccess(t *testing.T) {
	c := newLruCache(8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				c.put([32]byte{byte(i)}, [blsPubkeySize]byte{byte(j)})
				c.get([32]byte{byte(i)})
				c.get([32]byte{byte(i + 100)})
			}
		}(i)
	}
	wg.Wait()
}

// TestAggregatedCommitteePubkeyMatchesDirectAggregation verifies the cache
// returns exactly what a direct aggregation computes, on both the miss and
// the hit path.
func TestAggregatedCommitteePubkeyMatchesDirectAggregation(t *testing.T) {
	fixture := loadFixture(t, "Test_TimeoutPacketFromCosmos")
	msgs := decodeRelayerMessages(t, fixture.Steps[1].Data)
	require.NotEmpty(t, msgs.headers)
	committee := msgs.headers[0].ActiveSyncCommittee
	require.NotNil(t, committee)

	direct, err := aggregatePubkeys(committee.Pubkeys)
	require.NoError(t, err)

	// First call computes and caches.
	summary := committee.summarize()
	cached, err := aggregatedCommitteePubkey(summary.PubkeysHash, committee.Pubkeys)
	require.NoError(t, err)
	require.Equal(t, direct, cached)

	// Second call is served from the cache with the same value.
	cached, err = aggregatedCommitteePubkey(summary.PubkeysHash, committee.Pubkeys)
	require.NoError(t, err)
	require.Equal(t, direct, cached)

	// The claimed aggregate of the fixture committee matches its keys.
	var claimed [blsPubkeySize]byte
	copy(claimed[:], committee.AggregatePubkey)
	require.Equal(t, claimed, direct)
}
