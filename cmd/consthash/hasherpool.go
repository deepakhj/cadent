/*
   The server we listen for our data
*/

package main

import (
	"fmt"
	"github.com/stathat/consistent"
	"log"
	"net/url"
	//"sync/atomic"
	//"time"
)

const (
	CACHE_ITEMS = 10000
)

// for LRU cache "value" interface
type ServerCacheItem string

func (s ServerCacheItem) Size() int {
	return len(s)
}

// match the  ServerPoolRunner interface
type ConstHasher struct {
	Hasher     *consistent.Consistent
	Cache      *LRUCache
	ServerPool *CheckedServerPool

	ServerPutCounts uint64
	ServerGetCounts uint64
}

func (self ConstHasher) onServerUp(server url.URL) {
	log.Printf("Adding server %s to hasher", server.String())
	self.Hasher.Add(server.String())
	log.Printf("Current members %s from hasher", self.Members())
}

func (self ConstHasher) onServerDown(server url.URL) {
	log.Printf("Removing server %s from hasher", server.String())
	self.Hasher.Remove(server.String())
	log.Printf("Current members %s from hasher", self.Members())
}

//alias to hasher to allow to use our LRU cache
func (self *ConstHasher) Get(in_key string) (string, error) {
	srv, ok := self.Cache.Get(in_key)
	if !ok {
		srv, err := self.Hasher.Get(in_key)
		self.Cache.Set(in_key, ServerCacheItem(srv))
		return srv, err
	}
	return fmt.Sprintf("%s", srv), nil
}

//alias to hasher
func (self *ConstHasher) Members() []string {
	return self.Hasher.Members()
}

//make from our basic config object
func createConstHasherFromConfig(cfg *ConstHashConfig) (*ConstHasher, error) {
	var hasher ConstHasher
	hasher.Hasher = consistent.New()
	hasher.Cache = NewLRUCache(CACHE_ITEMS)

	s_pool_runner := ServerPoolRunner(hasher)
	s_pool, err := createServerPoolFromConfig(cfg, &s_pool_runner)

	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	hasher.ServerPool = s_pool
	return &hasher, nil

}

// make from a generic string
func createConstHasher(serverlist []*url.URL, checkerurl []*url.URL) (*ConstHasher, error) {
	var hasher ConstHasher

	hasher.Hasher = consistent.New()

	// cast it to the interface
	s_pool_runner := ServerPoolRunner(hasher)

	s_pool, err := createServerPool(serverlist, checkerurl, &s_pool_runner)
	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	hasher.ServerPool = &s_pool
	return &hasher, nil
}
