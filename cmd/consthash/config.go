package main

import (
	"fmt"
	"github.com/bbangert/toml"
	"log"
	"net"
	"strings"
	"time"
)

type ConstHashConfig struct {
	Servers                string        `toml:"servers"`
	MsgFormat              string        `toml:"msg_format"`
	Protocal               string        `toml:"in_protocol"`
	OutProtocal            string        `toml:"out_protocol"`
	Listen                 string        `toml:"listen"`
	ServerHeartBeat        time.Duration `toml:"heartbeat_time_delay"`
	ServerHeartBeatTimeout time.Duration `toml:"heartbeat_time_timeout"`
	MaxServerHeartBeatFail uint64        `toml:"failed_heartbeat_count"`
	ServerDownPolicy       string        `toml:"server_down_policy"`

	ServerListStr []string
	ServerList    map[string][]string
}

const (
	DEFAULT_CONFIG_SECTION    = "default"
	DEFAULT_PROTOCAL          = "tcp"
	DEFAULT_OUT_PROTOCAL      = "tcp"
	DEFAULT_LISTEN            = "127.0.0.1:6000"
	DEFAULT_HEARTBEAT_COUNT   = uint64(3)
	DEFAULT_HEARTBEAT         = time.Duration(30)
	DEFAULT_HEARTBEAT_TIMEOUT = time.Duration(5)
	DEFAULT_SERVERDOWN_POLICY = "keep"
)

// make our map of servers to hosts
func (self *ConstHashConfig) parseServerList() (s_map map[string][]string, s_list []string, err error) {

	var server_spl = strings.Split(self.Servers, ",")

	s_map = make(map[string][]string)

	for _, s_server := range server_spl {

		//trim white space
		var line = strings.Trim(s_server, " \t\n")

		// skip blank lines
		if len(line) == 0 {
			continue
		}

		//split again to get port #
		host, port, err := net.SplitHostPort(line)
		if err != nil {
			return nil, nil, err
		}
		s_list = append(s_list, line)
		s_map[line] = []string{host, port}
	}
	return s_map, s_list, nil
}

type ConstHashServers map[string]ConstHashConfig

func (self ConstHashServers) parseConfig() (out ConstHashServers, err error) {
	for chunk, cfg := range self {
		s_map, s_list, err := cfg.parseServerList()
		if err != nil {
			return nil, err
		}
		cfg.ServerListStr = s_list
		cfg.ServerList = s_map
		//set some defaults

		if cfg.Protocal != "tcp" && cfg.Protocal != "udp" {
			cfg.Protocal = DEFAULT_PROTOCAL
		}
		if cfg.OutProtocal != "tcp" && cfg.OutProtocal != "udp" {
			cfg.OutProtocal = DEFAULT_OUT_PROTOCAL
		}
		if len(cfg.Listen) == 0 {
			cfg.Listen = DEFAULT_LISTEN
		}
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

		log.Printf("  Protocal: %s", cfg.Protocal)
		log.Printf("  Listen: %v", cfg.Listen)
		log.Printf("  OutProtocal: %v", cfg.OutProtocal)
		log.Printf("  MaxServerHeartBeatFail: %v", cfg.MaxServerHeartBeatFail)
		log.Printf("  ServerHeartBeat: %v", cfg.ServerHeartBeat)
		log.Printf("  MsgFormat: %s ", cfg.MsgFormat)
		log.Printf("  Servers")
		for _, hosts := range cfg.ServerList {
			log.Printf("    %s:%s", hosts[0], hosts[1])
		}
	}
}
