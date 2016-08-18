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

package main

import (
	"encoding/json"
	"flag"
	"github.com/BurntSushi/toml"
	"log"
	"os"
)

type server struct {
	Servers      string `toml:"servers"`
	CheckServers string `toml:"check_servers"`
}
type config struct {
	Ffervers []server `toml:"servers"`
}
type ConfigServers map[string]config

func parseConfigFile(filename string) (err error) {

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
