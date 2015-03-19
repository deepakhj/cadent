/*
   The server we listen for our data
*/

package main

import (
	"./consistent"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
)

const (
	DEFAULT_CACHE_ITEMS = 10000
	DEFAULT_HASHER      = "crc32"
	DEFAULT_ELTER       = "graphite"
	DEFAULT_REPLICAS    = 4
)

// for LRU cache "value" interface
type ServerCacheItem string

func (s ServerCacheItem) Size() int {
	return len(s)
}

type MultiServerCacheItem []string

func (s MultiServerCacheItem) Size() int {
	return len(s)
}

// match the  ServerPoolRunner interface
type ConstHasher struct {
	Hasher     *consistent.Consistent
	Cache      *LRUCache
	ServerPool *CheckedServerPool

	HashAlgo     string
	HashElter    string
	HashReplicas int

	ServerPutCounts uint64
	ServerGetCounts uint64
}

func (self *ConstHasher) onServerUp(server url.URL) {
	log.Printf("Adding server %s to hasher", server.String())
	StatsdClient.Incr("hasher.added-server", 1)
	self.Hasher.Add(server.String())
	StatsdClient.GaugeAbsolute("hasher.up-servers", int64(len(self.Members())))
	//evil as this is we must clear the cache
	self.Cache.Clear()
	log.Printf("[onServerUp] Current members %s from hasher", self.Members())
}

func (self *ConstHasher) onServerDown(server url.URL) {
	log.Printf("Removing server %s from hasher", server.String())
	StatsdClient.Incr("hasher.removed-server", 1)
	self.Hasher.Remove(server.String())
	StatsdClient.GaugeAbsolute("hasher.up-servers", int64(len(self.Members())))
	//evil as this is we must clear the cache
	self.Cache.Clear()
	log.Printf("[onServerDown] Current members %s from hasher", self.Members())
}

// clean up the server name for statsd
func (self *ConstHasher) cleanKey(srv interface{}) string {
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
	return string(srv.(ServerCacheItem)), nil
}

//alias to hasher to allow to use our LRU cache
func (self *ConstHasher) GetN(in_key string, num int) ([]string, error) {
	cache_key := in_key + ":" + strconv.Itoa(num)
	srv, ok := self.Cache.Get(cache_key)

	if !ok {
		StatsdClient.Incr("lrucache.miss", 1)
		srv, err := self.Hasher.GetN(in_key, num)
		self.Cache.Set(cache_key, MultiServerCacheItem(srv))
		for _, useme := range srv {
			StatsdClient.Incr(fmt.Sprintf("hashserver.%s.used", self.cleanKey(useme)), 1)
		}
		return srv, err
	}
	for _, useme := range srv.(MultiServerCacheItem) {
		StatsdClient.Incr(fmt.Sprintf("hashserver.%s.used", self.cleanKey(useme)), 1)
	}

	StatsdClient.Incr("lrucache.hit", 1)
	return srv.(MultiServerCacheItem), nil
}

//alias to hasher
func (self *ConstHasher) Members() []string {
	if self.Hasher != nil {
		return self.Hasher.Members()
	}
	return []string{}
}

func (self *ConstHasher) DroppedServers() []string {
	var n_str []string
	for _, srv := range self.ServerPool.DroppedServers {
		n_str = append(n_str, srv.Name)

	}
	return n_str
}

func (self *ConstHasher) CheckingServers() []string {
	var n_str []string
	for _, srv := range self.ServerPool.Servers {
		n_str = append(n_str, srv.CheckName)
	}
	return n_str
}

//make from our basic config object
func createConstHasherFromConfig(cfg *Config, serverlist *ParsedServerConfig) (hasher *ConstHasher, err error) {
	hasher = new(ConstHasher)
	hasher.Hasher = consistent.New()

	hasher.HashAlgo = DEFAULT_HASHER
	if len(cfg.HashAlgo) > 0 {
		hasher.HashAlgo = cfg.HashAlgo
	}

	hasher.HashElter = DEFAULT_ELTER
	if len(cfg.HashElter) > 0 {
		hasher.HashElter = cfg.HashElter
	}

	hasher.HashReplicas = DEFAULT_REPLICAS
	if cfg.HashVNodes >= 0 {
		hasher.HashReplicas = cfg.HashVNodes
	}

	hasher.Hasher.SetNumberOfReplicas(hasher.HashReplicas)
	hasher.Hasher.SetHasherByName(hasher.HashAlgo)
	hasher.Hasher.SetElterByName(hasher.HashElter)

	if cfg.CacheItems <= 0 {
		hasher.Cache = NewLRUCache(DEFAULT_CACHE_ITEMS)
	} else {
		hasher.Cache = NewLRUCache(cfg.CacheItems)
	}
	log.Print("Hasher Cache size set to ", hasher.Cache.capacity)

	//s_pool_runner := ServerPoolRunner(hasher)
	s_pool, err := createServerPoolFromConfig(cfg, serverlist, hasher) //&s_pool_runner)

	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	hasher.ServerPool = s_pool
	//log.Print("HASHERRRRR ", &hasher.ServerPool.ServerActions, " ", &hasher, hasher.Hasher.Members())
	return hasher, nil

}

// make from a generic string
func createConstHasher(serverlist []*url.URL, checkerurl []*url.URL) (*ConstHasher, error) {
	var hasher = new(ConstHasher)

	hasher.Hasher = consistent.New()

	// cast it to the interface
	//s_pool_runner := ServerPoolRunner(hasher)

	s_pool, err := createServerPool(serverlist, checkerurl, hasher)
	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	hasher.ServerPool = s_pool
	return hasher, nil
}
