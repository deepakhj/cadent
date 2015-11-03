package consthash

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
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

type ConfigServerList struct {
	CheckServers []string `toml:"check_servers"`
	Servers      []string `toml:"servers"`
	HashKeys     []string `toml:"hashkeys"`
}

type Config struct {
	Name                    string
	PIDfile                 string        `toml:"pid_file"`
	NumProc                 int           `toml:"num_procs"`
	MaxPoolConnections      int           `toml:"max_pool_connections"`
	MaxPoolBufferSize       int           `toml:"pool_buffersize"`
	SendingConnectionMethod string        `toml:"sending_method"`
	MsgType                 string        `toml:"msg_type"`
	MsgFormatRegEx          string        `toml:"msg_regex"`
	ListenStr               string        `toml:"listen"`
	ServerHeartBeat         time.Duration `toml:"heartbeat_time_delay"`
	ServerHeartBeatTimeout  time.Duration `toml:"heartbeat_time_timeout"`
	MaxServerHeartBeatFail  uint64        `toml:"failed_heartbeat_count"`
	ServerDownPolicy        string        `toml:"server_down_policy"`
	CacheItems              uint64        `toml:"cache_items"`
	Profile                 bool          `toml:"cpu_profile"`
	Replicas                int           `toml:"num_dupe_replicas"`
	HashAlgo                string        `toml:"hasher_algo"`
	HashElter               string        `toml:"hasher_elter"`
	HashVNodes              int           `toml:"hasher_vnodes"`
	ReadBufferSize          int           `toml:"read_buffer_size"`

	//Timeouts
	WriteTimeout  time.Duration `toml:"write_timeout"`
	RunnerTimeout time.Duration `toml:"runner_timeout"`

	//the array of potential servers to send stuff to (yes we can dupe data out)
	ConfServerList []ConfigServerList `toml:"servers"`

	// send some stats to the land
	StatsdServer          string  `toml:"statsd_server"`
	StatsdPrefix          string  `toml:"statsd_prefix"`
	StatsdInterval        uint    `toml:"statsd_interval"`
	StatsdSampleRate      float32 `toml:"statsd_sample_rate"`
	StatsdTimerSampleRate float32 `toml:"statsd_timer_sample_rate"`

	// number of workers to handle message sending queue
	Workers int64 `toml:"workers"`

	// start a little http server for external health checks and stats probes
	HealthServerBind   string `toml:"internal_health_server_listen"`
	HealthServerPoints uint   `toml:"internal_health_server_points"`
	HealthServerPath   string `toml:"internal_health_server_path"`

	ListenURL   *url.URL
	ServerLists []*ParsedServerConfig //parsing up the ConfServerList after read

	//compiled Regex
	ComRegEx      *regexp.Regexp
	ComRegExNames []string

	// some runners in the hasher need extra config bits to
	// operate this construct that from the config args
	MsgConfig map[string]interface{}

	//this config can be used as a server list
	OkToUse bool
}

const (
	DEFAULT_CONFIG_SECTION       = "default"
	DEFAULT_LISTEN               = "tcp://127.0.0.1:6000"
	DEFAULT_HEARTBEAT_COUNT      = uint64(3)
	DEFAULT_HEARTBEAT            = time.Duration(30)
	DEFAULT_HEARTBEAT_TIMEOUT    = time.Duration(5)
	DEFAULT_SERVERDOWN_POLICY    = "keep"
	DEFAULT_HASHER_ALGO          = "crc32"
	DEFAULT_HASHER_ELTER         = "graphite"
	DEFAULT_HASHER_VNODES        = 4
	DEFAULT_DUPE_REPLICAS        = 1
	DEFAULT_MAX_POOL_CONNECTIONS = 10
)

type ConfigServers map[string]Config

// make our map of servers to hosts
func (self *Config) parseServerList(servers []string, checkservers []string, hashkeys []string) (*ParsedServerConfig, error) {

	if len(servers) == 0 {
		return nil, fmt.Errorf("No 'servers' in config section, skipping")
	}
	parsed := new(ParsedServerConfig)

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

func (self ConfigServers) ParseConfig(defaults Config) (out ConfigServers, err error) {
	for chunk, cfg := range self {

		if chunk == DEFAULT_CONFIG_SECTION {
			cfg.OkToUse = false
			self[chunk] = cfg
			continue
		}

		cfg.Name = chunk

		//set some defaults
		if len(cfg.ListenStr) == 0 {
			cfg.ListenStr = DEFAULT_LISTEN
		}
		l_url, err := url.Parse(cfg.ListenStr)
		if err != nil {
			return nil, err
		}
		cfg.ListenURL = l_url

		//parse our list of servers
		if len(cfg.ConfServerList) == 0 {
			return nil, fmt.Errorf("No 'servers' in config section, skipping")
		}
		cfg.ServerLists = make([]*ParsedServerConfig, 0)
		minServerCount := 0
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
		if cfg.MaxPoolBufferSize <= 0 {
			cfg.MaxPoolBufferSize = 0
			if defaults.MaxPoolBufferSize > 0 {
				cfg.MaxPoolBufferSize = defaults.MaxPoolBufferSize
			}
		}

		if cfg.ReadBufferSize <= 0 {
			cfg.ReadBufferSize = 0
			if defaults.ReadBufferSize > 0 {
				cfg.ReadBufferSize = defaults.ReadBufferSize
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
			log.Printf("[WARN] Changing replica count to %d to match the # servers", minServerCount)
			cfg.Replicas = minServerCount
		}

		if cfg.WriteTimeout == 0 {
			cfg.WriteTimeout = 0
			if defaults.WriteTimeout > 0 {
				cfg.WriteTimeout = defaults.WriteTimeout
			}
		}

		if cfg.RunnerTimeout == 0 {
			cfg.RunnerTimeout = 0
			if defaults.RunnerTimeout > 0 {
				cfg.RunnerTimeout = defaults.RunnerTimeout
			}
		}

		//need to make things seconds
		//NOTE: write and runner time are in millisecond

		cfg.ServerHeartBeat = cfg.ServerHeartBeat * time.Second
		cfg.ServerHeartBeatTimeout = cfg.ServerHeartBeatTimeout * time.Second
		if cfg.WriteTimeout != 0 {
			cfg.WriteTimeout = cfg.WriteTimeout * time.Millisecond
		}
		if cfg.RunnerTimeout != 0 {
			cfg.RunnerTimeout = cfg.RunnerTimeout * time.Millisecond
		}

		cfg.MsgConfig = make(map[string]interface{})

		//check the message type/regex
		if cfg.MsgType != "statsd" && cfg.MsgType != "graphite" && cfg.MsgType != "regex" {
			panic("`msg_type` must be 'statsd', 'graphite', or 'regex'")
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
		self[chunk] = cfg
	}

	return self, nil
}

func ParseConfigFile(filename string) (cfg ConfigServers, err error) {

	if _, err := toml.DecodeFile(filename, &cfg); err != nil {
		log.Printf("Error decoding config file: %s", err)
		return nil, err
	}
	var defaults Config
	var ok bool
	if defaults, ok = cfg[DEFAULT_CONFIG_SECTION]; !ok {
		panic("Need to have a [default] section in the config.")
	}
	return cfg.ParseConfig(defaults)
}

func (self ConfigServers) DefaultConfig() (def_cfg *Config, err error) {

	if val, ok := self[DEFAULT_CONFIG_SECTION]; ok {
		return &val, nil
	}

	return nil, fmt.Errorf("Could not find default in config file")
}

func (self ConfigServers) ServableConfigs() (configs []Config) {

	for _, cfg := range self {
		if !cfg.OkToUse {
			continue
		}
		configs = append(configs, cfg)
	}
	return configs
}

func (self *ConfigServers) DebugConfig() {

	for chunk, cfg := range *self {
		log.Printf("Chunk '%s'", chunk)
		if cfg.OkToUse {
			log.Printf("  Listen: %s", cfg.ListenURL.String())
			log.Printf("  MaxServerHeartBeatFail: %v", cfg.MaxServerHeartBeatFail)
			log.Printf("  ServerHeartBeat: %v", cfg.ServerHeartBeat)
			log.Printf("  MsgType: %s ", cfg.MsgType)
			log.Printf("  Hashing Algo: %s ", cfg.HashAlgo)
			log.Printf("  Dupe Replicas: %d ", cfg.Replicas)
			log.Printf("  Write Timeout: %v ", cfg.WriteTimeout)
			log.Printf("  Runner Timeout: %v ", cfg.RunnerTimeout)
			log.Printf("  Out Buffer Size: %v ", cfg.MaxPoolBufferSize)
			log.Printf("  Out Pool Connections: %v ", cfg.MaxPoolConnections)
			log.Printf("  Workers: %v ", cfg.Workers)

			log.Printf("  Servers")
			for _, slist := range cfg.ServerLists {
				for idx, hosts := range slist.ServerList {
					log.Printf("    %s Checked via %s", hosts, slist.CheckList[idx])
				}
			}
		}
	}
}