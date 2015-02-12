/*
   The server we listen for our data
*/

package main

import (
	"fmt"
	"github.com/stathat/consistent"
	"log"
	"net/url"
	"strings"
	//"sync/atomic"
	//"time"
)

const (
	DEFAULT_CACHE_ITEMS = 10000
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
	StatsdClient.Incr("hasher.added-server", 1)
	self.Hasher.Add(server.String())
	StatsdClient.Gauge("hasher.up-servers", int64(len(self.Members())))
	log.Printf("Current members %s from hasher", self.Members())
}

func (self ConstHasher) onServerDown(server url.URL) {
	log.Printf("Removing server %s from hasher", server.String())
	StatsdClient.Incr("hasher.removed-server", 1)
	self.Hasher.Remove(server.String())
	StatsdClient.Gauge("hasher.up-servers", int64(len(self.Members())))
	log.Printf("Current members %s from hasher", self.Members())
}

// clean up the server name for statsd
func (self ConstHasher) cleanKey(srv interface{}) string {
	srv_key := strings.Replace(fmt.Sprintf("%s", srv), ":", "-", -1)
	srv_key = strings.Replace(srv_key, "/", "", -1)
	srv_key = strings.Replace(srv_key, ".", "-", -1)
	return srv_key
}

//alias to hasher to allow to use our LRU cache
func (self *ConstHasher) Get(in_key string) (string, error) {
	srv, ok := self.Cache.Get(in_key)

	if !ok {
		StatsdClient.Incr("lrucache.miss", 1)
		srv, err := self.Hasher.Get(in_key)
		self.Cache.Set(in_key, ServerCacheItem(srv))

		StatsdClient.Incr(fmt.Sprintf("hashserver.%s.used", self.cleanKey(srv)), 1)
		return srv, err
	}
	StatsdClient.Incr(fmt.Sprintf("hashserver.%s.used", self.cleanKey(srv)), 1)
	StatsdClient.Incr("lrucache.hit", 1)
	return fmt.Sprintf("%s", srv), nil
}

//alias to hasher
func (self *ConstHasher) Members() []string {
	return self.Hasher.Members()
}

//make from our basic config object
func createConstHasherFromConfig(cfg *Config) (*ConstHasher, error) {
	var hasher ConstHasher
	hasher.Hasher = consistent.New()

	if cfg.CacheItems <= 0 {
		hasher.Cache = NewLRUCache(DEFAULT_CACHE_ITEMS)
	} else {
		hasher.Cache = NewLRUCache(cfg.CacheItems)
	}
	log.Print("Hasher Cache size set to ", hasher.Cache.capacity)
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
