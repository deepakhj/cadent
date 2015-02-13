package main

import (
	"fmt"
	"github.com/bbangert/toml"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ParsedServerConfig struct {
	ServerMap  map[string]*url.URL
	ServerList []string
	ServerUrls []*url.URL
	CheckMap   map[string]*url.URL
	CheckList  []string
	CheckUrls  []*url.URL
}

type Config struct {
	Servers                string        `toml:"servers"`
	CheckServers           string        `toml:"check_servers"`
	MsgType                string        `toml:"msg_type"`
	MsgFormatRegEx         string        `toml:"msg_regex"`
	ListenStr              string        `toml:"listen"`
	ServerHeartBeat        time.Duration `toml:"heartbeat_time_delay"`
	ServerHeartBeatTimeout time.Duration `toml:"heartbeat_time_timeout"`
	MaxServerHeartBeatFail uint64        `toml:"failed_heartbeat_count"`
	ServerDownPolicy       string        `toml:"server_down_policy"`
	CacheItems             uint64        `toml:"cache_items"`
	Profile                bool          `toml:"cpu_profile"`
	HashAlgo               string        `toml:"hasher_algo"`

	// send some stats to the land
	StatsdServer   string `toml:"statsd_server"`
	StatsdPrefix   string `toml:"statsd_prefix"`
	StatsdInterval uint   `toml:"statsd_interval"`

	// number of workers to handle message sending queue
	Workers int64 `toml:"workers"`

	// start a little http server for external health checks and stats probes
	HealthServerBind string `toml:"internal_health_server_listen"`

	ListenURL   *url.URL
	ServerLists *ParsedServerConfig

	//compiled Regex
	ComRegEx      *regexp.Regexp
	ComRegExNames []string

	// some runners in the hasher need extra config bits to
	// operate this construct that from the config args
	MsgConfig map[string]interface{}
}

const (
	DEFAULT_CONFIG_SECTION    = "default"
	DEFAULT_LISTEN            = "tcp://127.0.0.1:6000"
	DEFAULT_HEARTBEAT_COUNT   = uint64(3)
	DEFAULT_HEARTBEAT         = time.Duration(30)
	DEFAULT_HEARTBEAT_TIMEOUT = time.Duration(5)
	DEFAULT_SERVERDOWN_POLICY = "keep"
	DEFAULT_HASHER_ALGO       = "crc32"
)

type ConfigServers map[string]Config

// make our map of servers to hosts
func (self *Config) parseServerList() (*ParsedServerConfig, error) {

	parsed := new(ParsedServerConfig)

	var server_spl = strings.Split(self.Servers, ",")
	var checked_spl []string

	// need to set a defaults
	if len(self.CheckServers) == 0 {
		for _, s_server := range server_spl {
			checked_spl = append(checked_spl, strings.Trim(s_server, " \t\n"))
		}
	} else {
		checked_spl = strings.Split(self.CheckServers, ",")
		if len(checked_spl) != len(server_spl) {
			return nil, fmt.Errorf("need to have a servers count be the same as check_servers")
		}
	}
	parsed.CheckMap = make(map[string]*url.URL)
	parsed.ServerMap = make(map[string]*url.URL)
	for idx, s_server := range server_spl {

		//trim white space
		var line = strings.Trim(s_server, " \t\n")

		// skip blank lines
		if len(line) == 0 {
			continue
		}

		var check_line = strings.Trim(checked_spl[idx], " \t\n")

		parsed.ServerList = append(parsed.ServerList, line)
		server_url, err := url.Parse(line)
		if err != nil {
			return nil, err
		}
		parsed.ServerUrls = append(parsed.ServerUrls, server_url)
		parsed.ServerMap[line] = server_url

		//parse up the check URLs
		parsed.CheckList = append(parsed.CheckList, check_line)

		check_url, err := url.Parse(check_line)

		if err != nil {
			return nil, err
		}
		parsed.CheckUrls = append(parsed.CheckUrls, check_url)
		parsed.CheckMap[checked_spl[idx]] = check_url
	}
	return parsed, nil
}

func (self ConfigServers) parseConfig() (out ConfigServers, err error) {
	for chunk, cfg := range self {
		parsedServer, err := cfg.parseServerList()

		if err != nil {
			return nil, err
		}
		cfg.ServerLists = parsedServer

		//set some defaults

		if len(cfg.ListenStr) == 0 {
			cfg.ListenStr = DEFAULT_LISTEN
		}
		l_url, err := url.Parse(cfg.ListenStr)
		if err != nil {
			return nil, err
		}
		cfg.ListenURL = l_url
		if cfg.MaxServerHeartBeatFail == 0 {
			cfg.MaxServerHeartBeatFail = DEFAULT_HEARTBEAT_COUNT
		}
		if cfg.ServerHeartBeatTimeout == 0 {
			cfg.ServerHeartBeatTimeout = DEFAULT_HEARTBEAT_TIMEOUT
		}
		if cfg.ServerHeartBeat == 0 {
			cfg.ServerHeartBeat = DEFAULT_HEARTBEAT
		}
		if cfg.ServerDownPolicy == "" {
			cfg.ServerDownPolicy = DEFAULT_SERVERDOWN_POLICY
		}
		if cfg.HashAlgo == "" {
			cfg.HashAlgo = DEFAULT_HASHER_ALGO
		}

		//need to make things seconds
		cfg.ServerHeartBeat = cfg.ServerHeartBeat * time.Second
		cfg.ServerHeartBeatTimeout = cfg.ServerHeartBeatTimeout * time.Second

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

		self[chunk] = cfg

	}
	return self, nil
}

func parseConfigFile(filename string) (cfg ConfigServers, err error) {

	if _, err := toml.DecodeFile(filename, &cfg); err != nil {
		log.Printf("Error decoding config file: %s", err)
		return nil, err
	}
	return cfg.parseConfig()
}

func (self ConfigServers) defaultConfig() (def_cfg *Config, err error) {

	if val, ok := self[DEFAULT_CONFIG_SECTION]; ok {
		return &val, nil
	}

	for _, cfg := range self {
		//just yank the first one
		return &cfg, nil
	}

	return nil, fmt.Errorf("Could not find default in config file")
}

func (self *ConfigServers) debugConfig() {

	for chunk, cfg := range *self {
		log.Printf("Chunk '%s'", chunk)

		log.Printf("  Listen: %s", cfg.ListenURL.String())
		log.Printf("  MaxServerHeartBeatFail: %v", cfg.MaxServerHeartBeatFail)
		log.Printf("  ServerHeartBeat: %v", cfg.ServerHeartBeat)
		log.Printf("  MsgType: %s ", cfg.MsgType)
		log.Printf("  Hashing Algo: %s ", cfg.HashAlgo)
		log.Printf("  Servers")
		for idx, hosts := range cfg.ServerLists.ServerList {
			log.Printf("    %s Checked via %s", hosts, cfg.ServerLists.CheckList[idx])
		}
	}
}
