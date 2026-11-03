package spectre

import (
	"container/list"
	"sync"
)

// committeeAggregateCacheCapacity bounds the aggregate cache. A few dozen
// entries cover every committee of every live spectre client (committees
// rotate roughly daily); the bound keeps memory flat under clients created
// to thrash the cache.
const committeeAggregateCacheCapacity = 64

// committeeAggregateCache caches the aggregate public key of a sync committee
// keyed by the committee's pubkeys hash. Aggregating a committee is a full
// BLS pass over its keys, and a client's committee is stable for a sync
// period, so every message after the first reuses the cached aggregate.
//
// The cache is a pure optimization: the aggregate is a pure function of the
// pubkeys and only successfully parsed keys are cached, so a hit and a miss
// produce the same value, and verification gas does not depend on it.
var committeeAggregateCache = newLruCache(committeeAggregateCacheCapacity)

// aggregatedCommitteePubkey returns the aggregate public key of pubkeys,
// through the cache keyed by pubkeysHash. pubkeysHash must bind pubkeys:
// callers validate the key lengths first, so the hash tree root of the
// vector binds the exact keys.
func aggregatedCommitteePubkey(pubkeysHash []byte, pubkeys [][]byte) ([blsPubkeySize]byte, error) {
	var key [32]byte
	copy(key[:], pubkeysHash)
	if agg, ok := committeeAggregateCache.get(key); ok {
		return agg, nil
	}
	agg, err := aggregatePubkeys(pubkeys)
	if err != nil {
		return [blsPubkeySize]byte{}, err
	}
	committeeAggregateCache.put(key, agg)
	return agg, nil
}

// lruCache is a fixed-size LRU map from a 32-byte key to a 48-byte value.
type lruCache struct {
	mu    sync.Mutex
	cap   int
	ll    *list.List // front is the most recently used
	items map[[32]byte]*list.Element
}

type lruEntry struct {
	key   [32]byte
	value [blsPubkeySize]byte
}

func newLruCache(capacity int) *lruCache {
	return &lruCache{
		cap:   capacity,
		ll:    list.New(),
		items: make(map[[32]byte]*list.Element, capacity),
	}
}

func (c *lruCache) get(key [32]byte) ([blsPubkeySize]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return [blsPubkeySize]byte{}, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*lruEntry).value, true
}

func (c *lruCache) put(key [32]byte, value [blsPubkeySize]byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		el.Value.(*lruEntry).value = value
		return
	}
	c.items[key] = c.ll.PushFront(&lruEntry{key: key, value: value})
	for c.ll.Len() > c.cap {
		oldest := c.ll.Back()
		if oldest == nil {
			return
		}
		c.ll.Remove(oldest)
		delete(c.items, oldest.Value.(*lruEntry).key)
	}
}
