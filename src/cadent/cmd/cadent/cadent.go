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
	cadent "cadent/server"
	"cadent/server/config"
	"cadent/server/prereg"
	"cadent/server/stats"
	"cadent/server/utils/shared"
	"cadent/server/utils/shutdown"
	"encoding/json"
	"flag"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
)

// compile passing -ldflags "-X main.ConstHashBuild <build sha1>"
var ConstHashBuild string

var log = logging.MustGetLogger("main")

// due to the API render sometimes needing "LOTS" of ram to do it's stuff
// we need to force this issue
func freeOsMem() {
	t := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-t.C:
			debug.FreeOSMemory()
		}
	}
}

func RunApiOnly(file string, logfile string, loglevel string) error {

	conf, err := config.ParseApiConfigFile(file)
	if err != nil {
		panic(err)
	}

	//overrides
	if len(logfile) > 0 {
		conf.Logger.File = logfile
	}
	if len(loglevel) > 0 {
		conf.Logger.Level = loglevel
	}
	// set goodies in the shared data
	shared.Set("hashers", conf)
	shared.Set("is_writer", false) // these will get overridden later if there are these
	shared.Set("is_reader", true)
	shared.Set("is_hasher", false)
	shared.Set("is_api", true)

	err = conf.Start()
	if err != nil {
		panic(err)
	}

	return nil
}

// these are handlers global to all servers so need to add them separately
func AddGlobalHttpHandlers(h *config.HealthConfig, servers []*cadent.Server) {

	stat_func := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		jsonp := r.URL.Query().Get("jsonp")

		stats_map := make(map[string]*stats.HashServerStats)
		for _, serv := range servers {
			// note the server itself populates this every 5 seconds
			stats_map[serv.Name] = serv.GetStats()
		}
		resbytes, _ := json.Marshal(stats_map)
		if len(jsonp) > 0 {
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprintf(w, fmt.Sprintf("%s(", jsonp))
		} else {
			w.Header().Set("Content-Type", "application/json")

		}
		fmt.Fprintf(w, string(resbytes))
		if len(jsonp) > 0 {
			fmt.Fprintf(w, ")")
		}
	}

	hashcheck := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		h_key := r.URL.Query().Get("key")
		if len(h_key) == 0 {
			http.Error(w, "Please provide a 'key' to process", 404)
			return
		}
		hasher_map := make(map[string]cadent.ServerHashCheck)

		for _, serv := range servers {
			// note the server itself populates this ever 5 seconds
			hasher_map[serv.Name] = serv.HasherCheck([]byte(h_key))
		}
		w.Header().Set("Content-Type", "application/json")
		resbytes, _ := json.Marshal(hasher_map)
		fmt.Fprintf(w, string(resbytes))
	}

	h.GetMux().HandleFunc("/stats", stat_func)
	h.GetMux().HandleFunc("/hashcheck", hashcheck)
}

func main() {
	version := flag.Bool("version", false, "Print version and exit")
	configFile := flag.String("config", "", "Consitent Hash configuration file")
	regConfigFile := flag.String("prereg", "", "File that contains the Regex/Filtering by key to various backends")
	apiOnly := flag.String("api", "", "for instances of Cadent that ONLY do the API parts")
	loglevel := flag.String("loglevel", "", "Log Level (debug, info, warning, error, critical)")
	logfile := flag.String("logfile", "", "Log File (stdout, stderr, path/to/file)")

	flag.Parse()

	if *version {
		fmt.Printf("Cadent version %s\n\n", ConstHashBuild)
		os.Exit(0)
	}

	log.Info("Cadent version %s", ConstHashBuild)

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(*apiOnly) > 0 && len(*configFile) > 0 {
		log.Critical("Cannot have both a Config and an API only config")
		os.Exit(1)
	}
	if len(*apiOnly) > 0 && len(*regConfigFile) > 0 {
		log.Critical("Cannot have both a PreReg and an API only config, API should be configured w/ the prereg config")
		os.Exit(1)
	}

	// we halt here and just run the API
	if len(*apiOnly) > 0 {
		RunApiOnly(*apiOnly, *logfile, *loglevel)
		return
	}

	var conf *config.ConstHashConfig
	var err error

	conf, err = config.ParseHasherConfigFile(*configFile)
	if err != nil {
		log.Panicf("Error decoding config file: %s", err)
		os.Exit(1)
	}

	def, err := conf.Servers.DefaultConfig()
	if err != nil {
		log.Panicf("Error decoding config file: Could not find default: %s", err)
		os.Exit(1)
	}

	//overrides
	if len(*logfile) > 0 {
		conf.Logger.File = *logfile
	}
	if len(*loglevel) > 0 {
		conf.Logger.Level = *loglevel
	}

	conf.Logger.Start()

	// fire up procs, etc
	conf.System.Start()

	// set goodies in the shared data
	shared.Set("hashers", conf)
	shared.Set("is_writer", false) // these will get overridden later if there are these
	shared.Set("is_reader", false)
	shared.Set("is_hasher", false)
	shared.Set("is_api", false)

	if len(conf.Servers) > 1 {
		shared.Set("is_hasher", true)
	}

	// deal with the pre-reg file
	if len(*regConfigFile) != 0 {
		pr, err := prereg.ParseConfigFile(*regConfigFile)
		if err != nil {
			log.Critical("Error parsing PreReg: %s", err)
			os.Exit(1)
		}
		// make sure that we have all the backends
		err = conf.Servers.VerifyAndAssignPreReg(pr)
		if err != nil {
			log.Critical("Error parsing PreReg: %s", err)
			os.Exit(1)
		}
		def.PreRegFilters = pr
	}

	//some printstuff to verify settings
	conf.Servers.DebugConfig()

	//gossip
	conf.Gossip.Start()

	//profiling
	conf.Profile.Start()

	//initialize the statsd singleton
	conf.Statsd.Start()

	var servers []*cadent.Server
	useconfigs := conf.Servers.ServableConfigs()

	for _, cfg := range useconfigs {

		var hashers []*cadent.ConstHasher

		for _, serverlist := range cfg.ServerLists {
			hasher, err := cadent.CreateConstHasherFromConfig(cfg, serverlist)

			if err != nil {
				panic(err)
			}
			//go hasher.ServerPool.StartChecks() //started in the startserver
			hashers = append(hashers, hasher)
		}
		server, err := cadent.CreateServer(cfg, hashers)
		if err != nil {
			panic(err)
		}
		servers = append(servers, server)
	}
	// finally we need to set the accumulator backends
	// need to create all the servers first before we can start them as the PreReg mappings
	// need to know of all the things
	for _, srv := range servers {

		// start them up
		log.Notice("Staring Server `%s`", srv.Name)
		go srv.StartServer()
	}

	// traps some signals

	TrapExit := func() {
		//trap kills to flush queues and close connections
		sc := make(chan os.Signal, 1)
		signal.Notify(sc,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT)

		go func() {
			s := <-sc

			for _, srv := range servers {
				log.Warning("Caught %s: Closing Server `%s` out before quit ", s, srv.Name)
				srv.StopServer()
			}

			// need to stop the statsd collection as well
			if stats.StatsdClient != nil {
				stats.StatsdClient.Close()
			}
			if stats.StatsdClientSlow != nil {
				stats.StatsdClientSlow.Close()
			}

			signal.Stop(sc)
			//close(sc)
			// re-raise it
			//process, _ := os.FindProcess(os.Getpid())
			//process.Signal(s)

			shutdown.WaitOnShutdown()
			os.Exit(0)

			return
		}()
	}
	go freeOsMem()
	go TrapExit()

	wg := sync.WaitGroup{}
	wg.Add(1)

	// add server status handlers
	for _, serv := range servers {
		serv.AddStatusHandlers(conf.Health.GetMux())
	}

	AddGlobalHttpHandlers(&conf.Health, servers)
	//fire up the http stats if given
	conf.Health.Start(conf)

	wg.Wait()

}
