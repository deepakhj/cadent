package main

import (
	"fmt"
	"github.com/bbangert/toml"
	"log"
	"net/url"
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

type ConstHashConfig struct {
	Servers                string        `toml:"servers"`
	CheckServers           string        `toml:"check_servers"`
	MsgFormat              string        `toml:"msg_format"`
	ListenStr              string        `toml:"listen"`
	ServerHeartBeat        time.Duration `toml:"heartbeat_time_delay"`
	ServerHeartBeatTimeout time.Duration `toml:"heartbeat_time_timeout"`
	MaxServerHeartBeatFail uint64        `toml:"failed_heartbeat_count"`
	ServerDownPolicy       string        `toml:"server_down_policy"`
	Profile                bool          `toml:"cpu_profile"`

	ListenURL   *url.URL
	ServerLists *ParsedServerConfig
}

const (
	DEFAULT_CONFIG_SECTION    = "default"
	DEFAULT_LISTEN            = "tcp://127.0.0.1:6000"
	DEFAULT_HEARTBEAT_COUNT   = uint64(3)
	DEFAULT_HEARTBEAT         = time.Duration(30)
	DEFAULT_HEARTBEAT_TIMEOUT = time.Duration(5)
	DEFAULT_SERVERDOWN_POLICY = "keep"
)

type ConstHashServers map[string]ConstHashConfig

// make our map of servers to hosts
func (self *ConstHashConfig) parseServerList() (*ParsedServerConfig, error) {

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

func (self ConstHashServers) parseConfig() (out ConstHashServers, err error) {
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

		//need to make things seconds
		cfg.ServerHeartBeat = cfg.ServerHeartBeat * time.Second
		cfg.ServerHeartBeatTimeout = cfg.ServerHeartBeatTimeout * time.Second
		self[chunk] = cfg

	}
	return self, nil
}

func parseConfigFile(filename string) (cfg ConstHashServers, err error) {

	if _, err := toml.DecodeFile(filename, &cfg); err != nil {
		log.Printf("Error decoding config file: %s", err)
		return nil, err
	}
	return cfg.parseConfig()
}

func (self ConstHashServers) defaultConfig() (def_cfg *ConstHashConfig, err error) {

	if val, ok := self[DEFAULT_CONFIG_SECTION]; ok {
		return &val, nil
	}

	for _, cfg := range self {
		//just yank the first one
		return &cfg, nil
	}

	return nil, fmt.Errorf("Could not find default in config file")
}

func (self *ConstHashServers) debugConfig() {

	for chunk, cfg := range *self {
		log.Printf("Chunk '%s'", chunk)

		log.Printf("  Listen: %s", cfg.ListenURL.String())
		log.Printf("  MaxServerHeartBeatFail: %v", cfg.MaxServerHeartBeatFail)
		log.Printf("  ServerHeartBeat: %v", cfg.ServerHeartBeat)
		log.Printf("  MsgFormat: %s ", cfg.MsgFormat)
		log.Printf("  Servers")
		for idx, hosts := range cfg.ServerLists.ServerList {
			log.Printf("    %s Checked via %s", hosts, cfg.ServerLists.CheckList[idx])
		}
	}
}
