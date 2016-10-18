/*
Copyright 2016 Under Armour, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
	cache data from reads in a LRU cache w/ a TTL

*/

package indexer

import (
	"fmt"
	"sync"
	"time"
)

const (
	// max time to store in tag cache before purge
	TAG_CACHE_MAX_TIME_IN_CACHE = int64(time.Minute * time.Duration(60)) // an hour, just incase
)

// TagCacheItem cached item
type TagCacheItem struct {
	Added  int64
	Name   string
	Value  string
	IsMeta bool
	Id     string
}

func NewTagCacheItem(name string, value string, ismeta bool, id string) *TagCacheItem {

	return &TagCacheItem{
		Name:   name,
		Value:  value,
		IsMeta: ismeta,
		Id:     id,
		Added:  time.Now().Unix(),
	}
}

// TagCache simple map read cache
type TagCache struct {
	mu    *sync.RWMutex
	cache map[string]*TagCacheItem
}

func NewTagCache() *TagCache {
	rc := &TagCache{
		mu:    new(sync.RWMutex),
		cache: make(map[string]*TagCacheItem),
	}
	go rc.periodPurge()
	return rc
}

func (rc *TagCache) periodPurge() {
	tick := time.NewTicker(time.Minute * time.Duration(60))
	for {
		<-tick.C
		maxBack := time.Now().Unix() - TAG_CACHE_MAX_TIME_IN_CACHE
		rc.mu.Lock()
		for k, v := range rc.cache {
			if v.Added < maxBack {
				delete(rc.cache, k)
			}
		}
		rc.mu.Unlock()
	}
}

// Key of a given tag for the intneral map
func (rc *TagCache) Key(name, value string, ismeta bool) string {
	return name + value + fmt.Sprintf("%v", ismeta)
}

// Add a tag to the cache
func (rc *TagCache) Add(name, value string, is_meta bool, id string) bool {

	mKey := rc.Key(name, value, is_meta)
	rc.mu.RLock()
	_, ok := rc.cache[mKey]
	rc.mu.RUnlock()
	if !ok {
		rc_item := NewTagCacheItem(name, value, is_meta, id)
		rc.mu.Lock()
		rc.cache[mKey] = rc_item
		rc.mu.Unlock()
		return true
	}
	return false // already activated
}

// Get a tag id from the cache
func (rc *TagCache) Get(name, value string, is_meta bool) string {
	key := rc.Key(name, value, is_meta)

	rc.mu.RLock()
	gots, ok := rc.cache[key]
	rc.mu.RUnlock()
	if !ok {
		return ""
	}
	maxBack := time.Now().Unix() - TAG_CACHE_MAX_TIME_IN_CACHE
	if gots.Added < maxBack {
		rc.mu.Lock()
		delete(rc.cache, key)
		rc.mu.Unlock()
		return ""
	}
	return gots.Id
}

// Len of the cache
func (rc *TagCache) Len() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return len(rc.cache)
}
