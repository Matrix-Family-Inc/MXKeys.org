/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Created
 */

package keys

import (
	"container/list"
	"sync"
	"time"
)

// maxResolverCacheEntries bounds each resolver cache. Server names in
// queries come from anonymous clients, so the caches must not grow with
// the number of distinct names they send.
const maxResolverCacheEntries = 20000

// boundedCache is an LRU map capped at max entries. Expiry is decided per
// entry by ttl, measured from the time the entry was stored. Negative
// entries are counted on insert and removal so metrics never scan the map.
type boundedCache[V any] struct {
	mu       sync.Mutex
	max      int
	entries  map[string]*list.Element
	order    *list.List // front = most recently used
	ttl      func(V) time.Duration
	negative func(V) bool
	negCount int
}

type boundedItem[V any] struct {
	key      string
	value    V
	storedAt time.Time
}

func newBoundedCache[V any](max int, ttl func(V) time.Duration, negative func(V) bool) *boundedCache[V] {
	return &boundedCache[V]{
		max:      max,
		entries:  make(map[string]*list.Element),
		order:    list.New(),
		ttl:      ttl,
		negative: negative,
	}
}

func (c *boundedCache[V]) get(key string, now time.Time) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero V
	el, ok := c.entries[key]
	if !ok {
		return zero, false
	}
	item := el.Value.(*boundedItem[V])
	if now.Sub(item.storedAt) > c.ttl(item.value) {
		c.removeLocked(el)
		return zero, false
	}
	c.order.MoveToFront(el)
	return item.value, true
}

func (c *boundedCache[V]) set(key string, value V, storedAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[key]; ok {
		c.removeLocked(el)
	}
	for c.order.Len() >= c.max {
		c.removeLocked(c.order.Back())
	}
	c.entries[key] = c.order.PushFront(&boundedItem[V]{key: key, value: value, storedAt: storedAt})
	if c.negative(value) {
		c.negCount++
	}
}

func (c *boundedCache[V]) removeLocked(el *list.Element) {
	item := el.Value.(*boundedItem[V])
	if c.negative(item.value) {
		c.negCount--
	}
	delete(c.entries, item.key)
	c.order.Remove(el)
}

// stats returns the number of entries and of negative entries.
func (c *boundedCache[V]) stats() (total, negative int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len(), c.negCount
}
