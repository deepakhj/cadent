/*
   The server we listen for our data
*/

package cadent

import (
	"cadent/server/lrucache"
	"cadent/server/stats"
	"consistent"
	"fmt"
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

func (s ServerCacheItem) ToString() string {
	return string(s)
}

type MultiServerCacheItem []string

func (s MultiServerCacheItem) Size() int {
	return len(s)
}
func (s MultiServerCacheItem) ToString() string {
	return strings.Join(s, ",")
}

// match the  ServerPoolRunner interface
type ConstHasher struct {
	Hasher          *consistent.Consistent
	HashKeyToServer map[string]string
	ServerToHashkey map[string]string
	Cache           *lrucache.LRUCache
	ServerPool      *CheckedServerPool

	HashAlgo     string
	HashElter    string
	HashReplicas int

	ServerPutCounts uint64
	ServerGetCounts uint64
}

func (self *ConstHasher) memberString(members []string) string {
	member_str := ""
	for _, m := range members {
		if m != self.HashKeyToServer[m] {
			member_str += self.HashKeyToServer[m] + " (key:" + m + ")"
		} else {
			member_str += self.HashKeyToServer[m] + " "
		}
	}
	return member_str
}

func (self *ConstHasher) onServerUp(server url.URL) {
	log.Notice("Adding server %s to hasher", server.String())
	stats.StatsdClient.Incr("hasher.added-server", 1)

	//add the hash key, not the server string
	hash_key := self.ServerToHashkey[server.String()]
	self.Hasher.Add(hash_key)

	stats.StatsdClient.GaugeAbsolute("hasher.up-servers", int64(len(self.Members())))
	//evil as this is we must clear the cache
	self.Cache.Clear()

	log.Info("[onServerUp] Current members %s from hasher", self.memberString(self.Members()))
}

func (self *ConstHasher) onServerDown(server url.URL) {
	log.Notice("Removing server %s from hasher", server.String())
	stats.StatsdClient.Incr("hasher.removed-server", 1)

	//remove the hash key, not the server string
	hash_key := self.ServerToHashkey[server.String()]
	self.Hasher.Remove(hash_key)

	stats.StatsdClient.GaugeAbsolute("hasher.up-servers", int64(len(self.Members())))
	//evil as this is we must clear the cache
	self.Cache.Clear()
	log.Info("[onServerDown] Current members %s from hasher", self.memberString(self.Members()))
}

func (self *ConstHasher) PurgeServer(serverUrl *url.URL) bool {
	//not there, move on
	if _, ok := self.ServerToHashkey[serverUrl.String()]; !ok {
		return false
	}
	// purge from server lis first then remove the xrefs
	self.ServerPool.PurgeServer(serverUrl)

	hashkey := self.ServerToHashkey[serverUrl.String()]
	delete(self.HashKeyToServer, hashkey)
	delete(self.ServerToHashkey, serverUrl.String())
	return true
}

func (self *ConstHasher) AddServer(serverUrl *url.URL, checkUrl *url.URL, hashkey string) {

	//only add it once
	if _, ok := self.ServerToHashkey[serverUrl.String()]; ok {
		return
	}
	self.ServerToHashkey[serverUrl.String()] = hashkey
	self.HashKeyToServer[hashkey] = serverUrl.String()

	self.ServerPool.AddServer(serverUrl, checkUrl)
}

// clean up the server name for statsd
func (self *ConstHasher) cleanKey(srv string) string {
	srv_key := strings.Replace(srv, ":", "-", -1)
	srv_key = strings.Replace(srv_key, "/", "", -1)
	return strings.Replace(srv_key, ".", "-", -1)
}

//alias to hasher to allow to use our LRU cache
func (self *ConstHasher) Get(in_key string) (string, error) {
	srv, ok := self.Cache.Get(in_key)

	if !ok {
		stats.StatsdClient.Incr("lrucache.miss", 1)
		r_srv, err := self.Hasher.Get(in_key)

		//find out real server string
		real_server := self.HashKeyToServer[r_srv]

		self.Cache.Set(in_key, ServerCacheItem(real_server))

		stats.StatsdClient.Incr(fmt.Sprintf("hashserver.%s.used", self.cleanKey(real_server)), 1)

		return real_server, err
	}
	stats.StatsdClient.Incr(fmt.Sprintf("hashserver.%s.used", self.cleanKey(srv.ToString())), 1)
	stats.StatsdClient.Incr("lrucache.hit", 1)
	return string(srv.(ServerCacheItem)), nil
}

//alias to hasher to allow to use our LRU cache
func (self *ConstHasher) GetN(in_key string, num int) ([]string, error) {
	cache_key := in_key + ":" + strconv.Itoa(num)
	srv, ok := self.Cache.Get(cache_key)

	upStatsd := func(items []string) {
		for _, useme := range items {
			stats.StatsdClient.Incr(fmt.Sprintf("hashserver.%s.used", self.cleanKey(useme)), 1)
		}
	}

	if !ok {
		stats.StatsdClient.Incr("lrucache.miss", 1)
		srv, err := self.Hasher.GetN(in_key, num)
		//find out real server string(s)
		var real_servers []string
		for _, s := range srv {
			real_servers = append(real_servers, self.HashKeyToServer[s])
		}

		//log.Println("For: ", in_key, " Got: ", srv, " ->", real_servers)

		self.Cache.Set(cache_key, MultiServerCacheItem(real_servers))
		upStatsd(real_servers)
		return real_servers, err
	}
	upStatsd(srv.(MultiServerCacheItem))
	stats.StatsdClient.Incr("lrucache.hit", 1)

	return srv.(MultiServerCacheItem), nil
}

// Original Server list
func (self *ConstHasher) OriginalMembers() []string {
	if self.Hasher != nil {
		return self.ServerPool.ServerNames()
	}
	return []string{}
}

// List Memebers in the hashpool
func (self *ConstHasher) Members() []string {
	if self.Hasher != nil {
		return self.Hasher.Members()
	}
	return []string{}
}

// the list of dropped servers
func (self *ConstHasher) DroppedServers() []string {
	var n_str []string
	self.ServerPool.mu.Lock()
	defer self.ServerPool.mu.Unlock()
	for _, srv := range self.ServerPool.DroppedServers {
		n_str = append(n_str, srv.Name)
	}
	return n_str
}

// the list of servers that are the "checker" sockets
func (self *ConstHasher) CheckingServers() []string {
	var n_str []string
	self.ServerPool.mu.Lock()
	defer self.ServerPool.mu.Unlock()
	for _, srv := range self.ServerPool.Servers {
		n_str = append(n_str, srv.CheckName)
	}
	return n_str
}

//make HasherPool from our basic config object
func CreateConstHasherFromConfig(cfg *Config, serverlist *ParsedServerConfig) (hasher *ConstHasher, err error) {
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
		hasher.Cache = lrucache.NewLRUCache(DEFAULT_CACHE_ITEMS)
	} else {
		hasher.Cache = lrucache.NewLRUCache(cfg.CacheItems)
	}
	log.Notice("Hasher Cache size set to %d ", hasher.Cache.GetCapacity())

	//compose our maps and remaps for the hashkey server list maps
	hasher.HashKeyToServer = serverlist.HashkeyToServer
	hasher.ServerToHashkey = make(map[string]string)
	//now flip it
	for key, value := range hasher.HashKeyToServer {
		hasher.ServerToHashkey[value] = key
	}

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
