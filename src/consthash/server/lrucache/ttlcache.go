// a little TTL LRU cache
package lrucache

import (
	"sync"
	"time"
)

type TTLLRUCache struct {
	mu          sync.Mutex
	lc          *LRUCache
	default_ttl time.Duration
}

// Values that go into TTLLRUCache need to satisfy this interface.
type TTLValue interface {
	Size() int
	ToString() string
	IsExpired() bool
	Touch(time.Duration)
}

func NewTTLLRUCache(capacity uint64, ttl time.Duration) *TTLLRUCache {
	return &TTLLRUCache{
		lc:          NewLRUCache(capacity),
		default_ttl: ttl,
	}
}

// proxy some calls
func (lru *TTLLRUCache) GetCapacity() uint64 {
	return lru.lc.GetCapacity()
}

func (lru *TTLLRUCache) StatsJSON() string {
	return lru.lc.StatsJSON()
}

func (lru *TTLLRUCache) Keys() []string {
	return lru.lc.Keys()
}

func (lru *TTLLRUCache) Items() []Item {
	return lru.lc.Items()
}

func (lru *TTLLRUCache) Delete(key string) (rmkey string, rmelement Value) {
	return lru.lc.Delete(key)
}

func (lru *TTLLRUCache) Clear() {
	lru.lc.Clear()
}

func (lru *TTLLRUCache) SetCapacity(capacity uint64) {
	lru.lc.SetCapacity(capacity)
}

func (lru *TTLLRUCache) Stats() (length, size, capacity uint64, oldest time.Time) {
	return lru.lc.Stats()
}

// override some methods
func (lru *TTLLRUCache) Get(key string) (v TTLValue, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.lc.table[key]
	if element == nil {
		return nil, false
	}
	v = element.Value.(*entry).value.(TTLValue)
	if v.IsExpired() {
		lru.lc.Delete(key)
		return nil, false
	} else {
		v.Touch(lru.default_ttl)
	}
	lru.lc.moveToFront(element)
	return v, true
}

func (lru *TTLLRUCache) Set(key string, value TTLValue) (rmkey string, rmelement Value) {
	value.Touch(lru.default_ttl)
	return lru.lc.Set(key, value)
}

func (lru *TTLLRUCache) SetIfAbsent(key string, value TTLValue) {
	value.Touch(lru.default_ttl)
	lru.lc.SetIfAbsent(key, value.(Value))
}
