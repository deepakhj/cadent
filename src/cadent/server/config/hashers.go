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

package config

import (
	"cadent/server/accumulator"
	"cadent/server/prereg"
	"cadent/server/utils/tomlenv"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ParsedServerConfig struct {
	// the server name will default to the hashkey if none is given
	ServerMap       map[string]*url.URL
	HashkeyToServer map[string]string
	ServerList      []string
	HashkeyList     []string
	ServerUrls      []*url.URL
	CheckMap        map[string]*url.URL
	CheckList       []string
	CheckUrls       []*url.URL
}

type HasherServerList struct {
	CheckServers []string `toml:"check_servers" json:"check_servers,omitempty"`
	Servers      []string `toml:"servers" json:"servers,omitempty"`
	HashKeys     []string `toml:"hashkeys" json:"hashkeys,omitempty"`
}

type HasherConfig struct {
	Name string `toml:"name" json:"name,omitempty"`

	StatsTick bool `toml:"stats_tick" json:"stats_tick,omitempty"`

	// can be set in both defaults or overrides from the children

	MaxPoolConnections      int    `toml:"max_pool_connections" json:"max_pool_connections,omitempty"`
	MaxWritePoolBufferSize  int    `toml:"pool_buffersize" json:"pool_buffersize,omitempty"`
	SendingConnectionMethod string `toml:"sending_method" json:"sending_method,omitempty"`

	MsgType                string        `toml:"msg_type" json:"msg_type,omitempty"`
	MsgFormatRegEx         string        `toml:"msg_regex" json:"msg_regex,omitempty"`
	ListenStr              string        `toml:"listen" json:"listen,omitempty"`
	ServerHeartBeat        time.Duration `toml:"heartbeat_time_delay" json:"heartbeat_time_delay,omitempty"`
	ServerHeartBeatTimeout time.Duration `toml:"heartbeat_time_timeout" json:"heartbeat_time_timeout,omitempty"`
	MaxServerHeartBeatFail uint64        `toml:"failed_heartbeat_count" json:"failed_heartbeat_count,omitempty"`
	ServerDownPolicy       string        `toml:"server_down_policy" json:"server_down_policy,omitempty"`
	CacheItems             uint64        `toml:"cache_items" json:"cache_items,omitempty"`
	Replicas               int           `toml:"num_dupe_replicas" json:"num_dupe_replicas,omitempty"`
	HashAlgo               string        `toml:"hasher_algo" json:"hasher_algo,omitempty"`
	HashElter              string        `toml:"hasher_elter" json:"hasher_elter,omitempty"`
	HashVNodes             int           `toml:"hasher_vnodes" json:"hasher_vnodes,omitempty"`

	// TLS (if possible tcp/http)
	TLSkey  string `toml:"tls_key" json:"tls_key,omitempty"`
	TLScert string `toml:"tls_cert" json:"tls_cert,omitempty"`

	ClientReadBufferSize int64 `toml:"read_buffer_size" json:"read_buffer_size,omitempty"`
	MaxReadBufferSize    int64 `toml:"max_read_buffer_size" json:"max_read_buffer_size,omitempty"`

	//Timeouts
	WriteTimeout    time.Duration `toml:"write_timeout" json:"write_timeout,omitempty"`
	SplitterTimeout time.Duration `toml:"runner_timeout" json:"runner_timeout,omitempty"`

	//the array of potential servers to send stuff to (yes we can dupe data out)
	ConfServerList []HasherServerList `toml:"servers" json:"servers,omitempty"`

	// number of workers to handle message sending queue
	Workers    int64 `toml:"workers" json:"workers,omitempty"`
	OutWorkers int64 `toml:"out_workers" json:"out_workers,omitempty"`

	ListenURL  *url.URL `toml:"-" json:"-"`
	DevNullOut bool     `toml:"out_dev_null" json:"out_dev_null,omitempty"` // if set will NOT go to any outputs

	ServerLists []*ParsedServerConfig `toml:"-" json:"-"` //parsing up the ConfServerList after read

	// the pre-reg object this is used only in the Default section
	PreRegFilters prereg.PreRegMap `toml:"-" json:"-"`

	//Listener Specific pinned PreReg set
	PreRegFilter *prereg.PreReg `toml:"-" json:"-"`

	//compiled Regex
	ComRegEx      *regexp.Regexp `toml:"-" json:"-"`
	ComRegExNames []string       `toml:"-" json:"-"`

	// some runners in the hasher need extra config bits to
	// operate this construct that from the config args
	MsgConfig map[string]interface{} `toml:"-" json:"-"`

	//this config can be used as a server list
	OkToUse bool `toml:"-" json:"-"`
}

func (c *HasherConfig) ToJson() ([]byte, error) {
	return json.Marshal(c)
}

func (c *HasherConfig) String() string {
	j, _ := json.Marshal(c)
	return string(j)
}

const (
	DEFAULT_CONFIG_SECTION       = "default"
	DEFAULT_BACKEND_ONLY         = "backend_only"
	DEFAULT_LISTEN               = "tcp://127.0.0.1:6000"
	DEFAULT_HEARTBEAT_COUNT      = uint64(3)
	DEFAULT_HEARTBEAT            = time.Duration(30)
	DEFAULT_HEARTBEAT_TIMEOUT    = time.Duration(5)
	DEFAULT_SERVERDOWN_POLICY    = "keep"
	DEFAULT_HASHER_ALGO          = "mmh3"
	DEFAULT_HASHER_ELTER         = "simple"
	DEFAULT_HASHER_VNODES        = 4
	DEFAULT_DUPE_REPLICAS        = 1
	DEFAULT_MAX_POOL_CONNECTIONS = 10
)

// full const hash config
type HasherServers map[string]*HasherConfig

type ConstHashConfig struct {
	Gossip  GossipConfig  `toml:"gossip" json:"gossip,omitempty"`
	Statsd  StatsdConfig  `toml:"statsd" json:"statsd,omitempty"`
	Health  HealthConfig  `toml:"health" json:"health,omitempty"`
	Profile ProfileConfig `toml:"profile" json:"profile,omitempty"`
	System  SystemConfig  `toml:"system" json:"system,omitempty"`
	Servers HasherServers `toml:"servers" json:"servers,omitempty"`
}

// make our map of servers to hosts
func (self *HasherConfig) parseServerList(servers []string, checkservers []string, hashkeys []string) (*ParsedServerConfig, error) {

	parsed := new(ParsedServerConfig)

	if len(servers) == 0 {
		return nil, fmt.Errorf("No 'servers' in config section, skipping")
	}

	// need to set a defaults
	if len(checkservers) == 0 {
		checkservers = servers[:]
	} else {
		if len(checkservers) != len(servers) {
			return nil, fmt.Errorf("need to have a servers count be the same as check_servers")
		}
	}

	// need to set a defaults
	if len(hashkeys) == 0 {
		hashkeys = servers[:]
	} else {
		if len(hashkeys) != len(servers) {
			return nil, fmt.Errorf("need to have a servers count be the same as hashkeys")
		}
	}

	parsed.CheckMap = make(map[string]*url.URL)
	parsed.ServerMap = make(map[string]*url.URL)
	parsed.HashkeyToServer = make(map[string]string)
	for idx, s_server := range servers {

		//trim white space
		line := strings.Trim(s_server, " \t\n")

		// skip blank lines
		if len(line) == 0 {
			continue
		}

		check_line := strings.Trim(checkservers[idx], " \t\n")
		hashkey_line := hashkeys[idx] // leave this "as is"

		parsed.HashkeyList = append(parsed.HashkeyList, hashkey_line)
		parsed.ServerList = append(parsed.ServerList, line)
		server_url, err := url.Parse(line)
		if err != nil {
			return nil, err
		}
		parsed.ServerUrls = append(parsed.ServerUrls, server_url)

		if _, ok := parsed.ServerMap[line]; ok {
			return nil, fmt.Errorf("Servers need to be unique")
		}

		parsed.ServerMap[line] = server_url

		if _, ok := parsed.HashkeyToServer[hashkey_line]; ok {
			return nil, fmt.Errorf("Hashkeys need to be unique")
		}

		parsed.HashkeyToServer[hashkey_line] = line

		//parse up the check URLs
		parsed.CheckList = append(parsed.CheckList, check_line)

		check_url, err := url.Parse(check_line)

		if err != nil {
			return nil, err
		}
		parsed.CheckUrls = append(parsed.CheckUrls, check_url)
		parsed.CheckMap[check_line] = check_url
	}
	return parsed, nil
}

func (self HasherServers) ParseHasherConfig(defaults *HasherConfig) (out HasherServers, err error) {

	have_listener := false
	out = make(HasherServers, 0)

	for chunk, cfg := range self {
		cfg.Name = chunk

		if chunk == DEFAULT_CONFIG_SECTION {
			cfg.OkToUse = false
			out[chunk] = cfg
			continue
		}

		//stats on or off
		cfg.StatsTick = defaults.StatsTick

		//set some defaults
		if len(cfg.ListenStr) == 0 {
			cfg.ListenStr = DEFAULT_LISTEN
		}
		if cfg.ListenStr == DEFAULT_BACKEND_ONLY {
			//log.Info("Configuring %s as a BACKEND only (no listeners)", chunk)
			cfg.ListenURL = nil
		} else {
			l_url, err := url.Parse(cfg.ListenStr)
			if err != nil {
				return nil, err
			}
			cfg.ListenURL = l_url
			have_listener = true
		}

		//parse our list of servers
		minServerCount := 0
		if !cfg.DevNullOut {
			if len(cfg.ConfServerList) == 0 {
				return nil, fmt.Errorf("No 'servers' in config section, skipping")
			}
			cfg.ServerLists = make([]*ParsedServerConfig, 0)
			for _, confSection := range cfg.ConfServerList {

				parsed, err := cfg.parseServerList(confSection.Servers, confSection.CheckServers, confSection.HashKeys)
				// no servers is ok for the defaults section
				if err != nil {
					return nil, err
				}

				if minServerCount == 0 {
					minServerCount = len(parsed.ServerList)
				} else {
					minServerCount = int(math.Min(float64(len(parsed.ServerList)), float64(minServerCount)))
				}
				cfg.ServerLists = append(cfg.ServerLists, parsed)
			}
		}
		if cfg.MaxServerHeartBeatFail == 0 {
			cfg.MaxServerHeartBeatFail = DEFAULT_HEARTBEAT_COUNT
			if defaults.MaxServerHeartBeatFail > 0 {
				cfg.MaxServerHeartBeatFail = defaults.MaxServerHeartBeatFail
			}
		}
		if cfg.ServerHeartBeatTimeout == 0 {
			cfg.ServerHeartBeatTimeout = DEFAULT_HEARTBEAT_TIMEOUT
			if defaults.ServerHeartBeatTimeout > 0 {
				cfg.ServerHeartBeatTimeout = defaults.ServerHeartBeatTimeout
			}
		}
		if cfg.ServerHeartBeat == 0 {
			cfg.ServerHeartBeat = DEFAULT_HEARTBEAT
			if defaults.ServerHeartBeat > 0 {
				cfg.ServerHeartBeat = defaults.ServerHeartBeat
			}
		}
		if cfg.ServerDownPolicy == "" {
			cfg.ServerDownPolicy = DEFAULT_SERVERDOWN_POLICY
			if defaults.ServerDownPolicy != "" {
				cfg.ServerDownPolicy = defaults.ServerDownPolicy
			}
		}
		if cfg.MaxPoolConnections <= 0 {
			cfg.MaxPoolConnections = DEFAULT_MAX_POOL_CONNECTIONS
			if defaults.MaxPoolConnections > 0 {
				cfg.MaxPoolConnections = defaults.MaxPoolConnections
			}
		}
		if cfg.MaxWritePoolBufferSize <= 0 {
			cfg.MaxWritePoolBufferSize = 0
			if defaults.MaxWritePoolBufferSize > 0 {
				cfg.MaxWritePoolBufferSize = defaults.MaxWritePoolBufferSize
			}
		}

		if cfg.ClientReadBufferSize <= 0 {
			cfg.ClientReadBufferSize = 1024 * 1024
			if defaults.ClientReadBufferSize > 0 {
				cfg.ClientReadBufferSize = defaults.ClientReadBufferSize
			}
		}

		if cfg.MaxReadBufferSize <= 0 {
			cfg.MaxReadBufferSize = 1000 * cfg.ClientReadBufferSize
			if defaults.MaxReadBufferSize > 0 {
				cfg.MaxReadBufferSize = defaults.MaxReadBufferSize
			}
		}

		if cfg.HashAlgo == "" {
			cfg.HashAlgo = DEFAULT_HASHER_ALGO
			if defaults.HashAlgo != "" {
				cfg.HashAlgo = defaults.HashAlgo
			}
		}
		if cfg.HashElter == "" {
			cfg.HashElter = DEFAULT_HASHER_ELTER
			if defaults.HashElter != "" {
				cfg.HashElter = defaults.HashElter
			}
		}
		if cfg.SendingConnectionMethod == "" {
			if defaults.SendingConnectionMethod != "" {
				cfg.SendingConnectionMethod = defaults.SendingConnectionMethod
			}
		}
		if cfg.HashVNodes == 0 {
			cfg.HashVNodes = DEFAULT_HASHER_VNODES
			if defaults.HashVNodes > 0 {
				cfg.HashVNodes = defaults.HashVNodes
			}
		}

		if cfg.Workers == 0 {
			if defaults.Workers > 0 {
				cfg.Workers = defaults.Workers
			}
		}
		if cfg.OutWorkers == 0 {
			if defaults.OutWorkers > 0 {
				cfg.OutWorkers = defaults.OutWorkers
			}
		}

		if cfg.CacheItems == 0 {
			if defaults.CacheItems > 0 {
				cfg.CacheItems = defaults.CacheItems
			}
		}
		if cfg.Replicas == 0 {
			cfg.Replicas = DEFAULT_DUPE_REPLICAS
			if defaults.Replicas > 0 {
				cfg.Replicas = defaults.Replicas
			}
		}

		if cfg.Replicas > minServerCount {
			log.Warning("Changing replica count to %d to match the # servers", minServerCount)
			cfg.Replicas = minServerCount
		}

		if cfg.WriteTimeout == 0 {
			cfg.WriteTimeout = 0
			if defaults.WriteTimeout > 0 {
				cfg.WriteTimeout = defaults.WriteTimeout
			}
		}

		if cfg.SplitterTimeout == 0 {
			cfg.SplitterTimeout = 0
			if defaults.SplitterTimeout > 0 {
				cfg.SplitterTimeout = defaults.SplitterTimeout
			}
		}

		//need to make things seconds
		//NOTE: write and runner time are in millisecond

		cfg.ServerHeartBeat = cfg.ServerHeartBeat * time.Second
		cfg.ServerHeartBeatTimeout = cfg.ServerHeartBeatTimeout * time.Second
		if cfg.WriteTimeout != 0 {
			cfg.WriteTimeout = cfg.WriteTimeout * time.Millisecond
		}
		if cfg.SplitterTimeout != 0 {
			cfg.SplitterTimeout = cfg.SplitterTimeout * time.Millisecond
		}

		cfg.MsgConfig = make(map[string]interface{})

		//check the message type/regex
		if cfg.MsgType != "statsd" && cfg.MsgType != "graphite" && cfg.MsgType != "regex" && cfg.MsgType != "carbon2" && cfg.MsgType != "json" && cfg.MsgType != "opentsdb" {
			panic("`msg_type` must be 'statsd', 'graphite', 'carbon2', 'opentsdb', 'json' or 'regex'")
		}
		if cfg.MsgType == "regex" && len(cfg.MsgFormatRegEx) == 0 {
			panic("`msg_type` of 'regex' needs to have `msg_regex` defined")
		}

		cfg.MsgConfig["type"] = cfg.MsgType

		if cfg.MsgType == "regex" {
			//check the regex itself
			cfg.ComRegEx = regexp.MustCompile(cfg.MsgFormatRegEx)
			cfg.ComRegExNames = cfg.ComRegEx.SubexpNames()
			got_name := false
			for _, nm := range cfg.ComRegExNames {
				if nm == "Key" {
					got_name = true
				}
			}
			if !got_name {
				panic("`msg_regex` MUST have a `(?P<Key>...)` group")
			}
			cfg.MsgConfig["regexp"] = cfg.ComRegEx
			cfg.MsgConfig["regexpNames"] = cfg.ComRegExNames

		}
		cfg.OkToUse = true
		out[chunk] = cfg
	}
	//need to have at least ONE listener
	if !have_listener {
		panic("No bind/listeners defined, need at least one")
	}
	return out, nil
}

func ParseHasherConfigFile(filename string) (cfg *ConstHashConfig, err error) {
	cfg = new(ConstHashConfig)
	if _, err := tomlenv.DecodeFile(filename, cfg); err != nil {
		log.Critical("Error decoding config file: %s", err)
		return nil, err
	}
	var defaults *HasherConfig
	var ok bool

	if len(cfg.Servers) == 0 {
		panic("Need some [server]s in the config")
	}

	servers := cfg.Servers

	if defaults, ok = servers[DEFAULT_CONFIG_SECTION]; !ok {
		panic("Need to have a [default] section in the config.")
	}
	cfg.Servers, err = servers.ParseHasherConfig(defaults)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}


func ParseHasherConfigString(bits string) (cfg *ConstHashConfig, err error) {
	cfg = new(ConstHashConfig)
	if _, err := tomlenv.Decode(bits, cfg); err != nil {
		log.Critical("Error decoding config file: %s", err)
		return nil, err
	}
	var defaults *HasherConfig
	var ok bool

	if len(cfg.Servers) == 0 {
		panic("Need some [server]s in the config")
	}

	servers := cfg.Servers

	if defaults, ok = servers[DEFAULT_CONFIG_SECTION]; !ok {
		panic("Need to have a [default] section in the config.")
	}
	cfg.Servers, err = servers.ParseHasherConfig(defaults)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (self HasherServers) DefaultConfig() (def_cfg *HasherConfig, err error) {

	if val, ok := self[DEFAULT_CONFIG_SECTION]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("Could not find default in config file")
}

func (self HasherServers) VerifyAndAssignPreReg(prm prereg.PreRegMap) (err error) {

	// validate that all the backends in the server conf acctually match something
	// in the regex filtering

	for _, pr := range prm {
		// check that the listern server is really a listen server and exists
		var listen_s *HasherConfig
		var ok bool
		if listen_s, ok = self[pr.ListenServer]; !ok {
			return fmt.Errorf("ListenServer `%s` is not in the Config servers", pr.ListenServer)
		}
		if listen_s.ListenStr == "backend_only" {
			return fmt.Errorf("ListenServer `%s` cannot be a `backend_only` listener", pr.ListenServer)
		}

		// assign the filters to this server so it can use it
		// (we can do this as listen_s is a ptr .. said here in case i forget)
		listen_s.PreRegFilter = pr
		for _, filter := range pr.FilterList {

			if _, ok := self[filter.Backend()]; !ok {
				return fmt.Errorf("Backend `%s` is not in the Config servers", filter.Backend())
			}
		}

		//verify that if there is an Accumulator that the backend for it does in fact live
		if pr.Accumulator != nil {
			// special BLACKHOLE backend
			if pr.Accumulator.ToBackend == accumulator.BLACK_HOLE_BACKEND {
				log.Notice("NOTE: using BlackHole for Accumulator `%s`", pr.Accumulator.Name)
				continue
			} else {
				_, ok := self[pr.Accumulator.ToBackend]
				if !ok {
					return fmt.Errorf("Backend `%s` for accumulator is not in the Config servers", pr.Accumulator.ToBackend)
				}
			}
		}
	}
	return nil

}

func (self HasherServers) ServableConfigs() (configs []*HasherConfig) {

	for _, cfg := range self {
		if !cfg.OkToUse {
			continue
		}
		configs = append(configs, cfg)
	}
	return configs
}

func (self *HasherServers) DebugConfig() {
	log.Debug("== Consthash backends ===")
	for chunk, cfg := range *self {
		log.Debug("Section '%s'", chunk)
		if cfg.OkToUse {
			if cfg.ListenURL != nil {
				log.Debug("  Listen: %s", cfg.ListenURL.String())
			} else {
				log.Debug("  BACKEND ONLY: %s", chunk)
			}
			log.Debug("  MaxServerHeartBeatFail: %v", cfg.MaxServerHeartBeatFail)
			log.Debug("  ServerHeartBeat: %v", cfg.ServerHeartBeat)
			log.Debug("  MsgType: %s ", cfg.MsgType)
			log.Debug("  Hashing Algo: %s ", cfg.HashAlgo)
			log.Debug("  Dupe Replicas: %d ", cfg.Replicas)
			log.Debug("  Write Timeout: %v ", cfg.WriteTimeout)
			log.Debug("  Splitter Timeout: %v ", cfg.SplitterTimeout)
			log.Debug("  Write Buffer Size: %v ", cfg.MaxWritePoolBufferSize)
			log.Debug("  Read Buffer Size: %v ", cfg.ClientReadBufferSize)
			log.Debug("  Max Read Buffer Size: %v ", cfg.MaxReadBufferSize)
			log.Debug("  Write Pool Method: %v ", cfg.SendingConnectionMethod)
			log.Debug("  Write Pool Connections: %v ", cfg.MaxPoolConnections)
			log.Debug("  Workers-Input Queue Size: %v ", cfg.Workers)
			log.Debug("  Output workers: %v ", cfg.OutWorkers)

			if cfg.DevNullOut {
				log.Debug("  out -> dev/null")
			} else {
				log.Debug("  Servers")
				for idx, slist := range cfg.ServerLists {
					log.Debug("   Replica Set %d", idx)
					for idx, hosts := range slist.ServerList {
						log.Debug("     %s Checked via %s", hosts, slist.CheckList[idx])
					}
				}
			}
		}
		if cfg.PreRegFilters != nil {
			cfg.PreRegFilters.LogConfig()
		}
	}
}
