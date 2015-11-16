package main

import (
	"flag"
	"log"
	"os"
	"github.com/BurntSushi/toml"
	"encoding/json"

)
type server struct{
	Servers string  `toml:"servers"`
	CheckServers string `toml:"check_servers"`
}
type config struct{

	Ffervers []server `toml:"servers"`
}
type ConfigServers map[string]config

func parseConfigFile(filename string) ( err error) {

	var cfg ConfigServers

	if _, err := toml.DecodeFile(filename, &cfg); err != nil {
		log.Printf("Error decoding config file: %s", err)
		return err
	}
	b, _ := json.Marshal(cfg)
	log.Printf("%s", b)
	log.Printf("%s", cfg)
	return nil
}

func main() {
	configFile := flag.String("config", "config.toml", "Consitent Hash configuration file")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	var err error

	err = parseConfigFile(*configFile)
	if err != nil {
		log.Printf("Error decoding config file: %s", err)
		os.Exit(1)
	}
}
