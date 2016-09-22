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
	"cadent/server/gossip"
	pages "cadent/server/pages"
	prereg "cadent/server/prereg"
	"cadent/server/stats"
	"cadent/server/utils/shared"
	"cadent/server/utils/shutdown"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"io"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// compile passing -ldflags "-X main.ConstHashBuild <build sha1>"
var ConstHashBuild string

var log = logging.MustGetLogger("main")

func logInit(file string, format string) {
	if format == "" {
		format = "%{color}%{time:2006-01-02 15:04:05.000} [%{module}] (%{shortfile}) ▶ %{level:.4s} %{color:reset} %{message}"
	}

	var file_o io.Writer
	var err error
	switch file {
	case "stdout":
		file_o = os.Stdout
	case "":
		file_o = os.Stdout
	case "stderr":
		file_o = os.Stderr
	default:

		file_o, err = os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if os.IsNotExist(err) {
			err = nil
			file_o, err = os.Create(file)
		}
		if err != nil {
			panic(err)
		}
	}
	logBackend := logging.NewLogBackend(file_o, "", 0)
	logging.SetFormatter(logging.MustStringFormatter(format))
	logging.SetBackend(logBackend)
}

// need to up this guy otherwise we quickly run out of sockets
func setSystemStuff(num_procs int) {
	if num_procs <= 0 {
		num_procs = runtime.NumCPU()
	}
	log.Notice("[System] Setting GOMAXPROCS to %d", num_procs)

	runtime.GOMAXPROCS(num_procs)

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Warning("[System] Error Getting Rlimit: %d", err)
	}
	log.Notice("[System] Current Rlimit: Max %d, Cur %d", rLimit.Max, rLimit.Cur)

	rLimit.Max = 999999
	rLimit.Cur = 999999
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Warning("[System] Error Setting Rlimit: %s", err)
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Warning("[System] Error Getting Rlimit: %s ", err)
	}
	log.Notice("[System] Final Rlimit Final: Max %d, Cur %d", rLimit.Max, rLimit.Cur)
}

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

// Fire up the http server for stats and healthchecks
func startStatsServer(defaults *cadent.Config, servers []*cadent.Server) {

	log.Notice("Starting Status server on %s", defaults.HealthServerBind)

	var names []string
	for _, serv := range servers {
		names = append(names, serv.Name)
		serv.AddStatusHandlers()
	}

	fileserve := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		fmt.Fprintf(w, pages.STATS_INDEX_PAGE)
	}

	status := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		fmt.Fprintf(w, "ok")
	}
	stats := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		jsonp := r.URL.Query().Get("jsonp")

		stats_map := make(map[string]*cadent.ServerStats)
		for idx, serv := range servers {
			// note the server itself populates this every 5 seconds
			stats_map[serv.Name] = servers[idx].GetStats()
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

		for idx, serv := range servers {
			// note the server itself populates this ever 5 seconds
			hasher_map[serv.Name] = servers[idx].HasherCheck([]byte(h_key))
		}
		w.Header().Set("Content-Type", "application/json")
		resbytes, _ := json.Marshal(hasher_map)
		fmt.Fprintf(w, string(resbytes))
	}

	listservers := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")

		s_list := make(map[string][]string)

		for _, serv := range servers {

			s_list[serv.Name] = []string{
				fmt.Sprintf("/%s", serv.Name),
				fmt.Sprintf("/%s/ping", serv.Name),
				fmt.Sprintf("/%s/ops/status", serv.Name),
				fmt.Sprintf("/%s/stats", serv.Name),
				fmt.Sprintf("/%s/addserver", serv.Name),
				fmt.Sprintf("/%s/purgeserver", serv.Name),
				fmt.Sprintf("/%s/accumulator", serv.Name),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		resbytes, _ := json.Marshal(s_list)
		fmt.Fprintf(w, string(resbytes))
	}

	http.HandleFunc("/", fileserve)
	http.HandleFunc("/ops/status", status)
	http.HandleFunc("/ping", status)
	http.HandleFunc("/status", status)
	http.HandleFunc("/stats", stats)
	http.HandleFunc("/hashcheck", hashcheck)
	http.HandleFunc("/servers", listservers)

	// stats stuff + profiler live on the same mux
	if len(defaults.HealthServerBind) > 0 && (len(defaults.ProfileBind) > 0 && defaults.ProfileBind != defaults.HealthServerBind) {
		log.Notice("Starting internal Stats server on %s", defaults.HealthServerBind)

		if len(defaults.HealthServerKey) > 0 && len(defaults.HealthServerCert) > 0 {
			cer, err := tls.LoadX509KeyPair(defaults.HealthServerCert, defaults.HealthServerKey)
			if err != nil {
				log.Panicf("Could not start https server: %v", err)
			}
			config := &tls.Config{Certificates: []tls.Certificate{cer}}
			conn, err := tls.Listen("tcp", defaults.HealthServerBind, config)
			if err != nil {
				log.Panicf("Could not make tls http socket: %s", err)
			}
			go http.Serve(conn, nil)

		} else {

			err := http.ListenAndServe(defaults.HealthServerBind, nil)
			if err != nil {
				log.Panicf("Could not start http server %s", defaults.HealthServerBind)
			}
		}
	}
}

func RunApiOnly(file string) error {
	return nil
}

func main() {
	version := flag.Bool("version", false, "Print version and exit")
	configFile := flag.String("config", "config.toml", "Consitent Hash configuration file")
	regConfigFile := flag.String("prereg", "", "File that contains the Regex/Filtering by key to various backends")
	apiOnly := flag.String("api", "", "for instances of Cadent that only do the API parts")
	loglevel := flag.String("loglevel", "DEBUG", "Log Level (debug, info, warning, error, critical)")
	logfile := flag.String("logfile", "stdout", "Log File (stdout, stderr, path/to/file)")

	flag.Parse()

	if *version {
		fmt.Printf("Cadent version %s\n\n", ConstHashBuild)
		os.Exit(0)
	}

	logInit(*logfile, "")

	log.Info("Cadent version %s", ConstHashBuild)

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	switch strings.ToUpper(*loglevel) {
	case "DEBUG":
		logging.SetLevel(logging.DEBUG, "")
	case "INFO":
		logging.SetLevel(logging.INFO, "")
	case "WARNING":
		logging.SetLevel(logging.WARNING, "")
	case "ERROR":
		logging.SetLevel(logging.ERROR, "")
	case "CRITICAL":
		logging.SetLevel(logging.CRITICAL, "")

	}

	if len(*apiOnly) > 0 && len(*configFile) > 0 {
		log.Critical("Cannot have both a Config and an API only config")
		os.Exit(1)
	}
	if len(*apiOnly) > 0 && len(*regConfigFile) > 0 {
		log.Critical("Cannot have both a PreRege and an API only config, API should be configured w/ the prereg config")
		os.Exit(1)
	}

	// we halt here and just run the API
	if len(*apiOnly) > 0 {
		RunApiOnly(*apiOnly)
		return
	}

	var config cadent.ConfigServers
	var err error

	config, err = cadent.ParseConfigFile(*configFile)
	if err != nil {
		log.Info("Error decoding config file: %s", err)
		os.Exit(1)
	}

	def, err := config.DefaultConfig()
	if err != nil {
		log.Critical("Error decoding config file: Could not find default: %s", err)
		os.Exit(1)
	}

	if def.Profile {
		if def.ProfileRate > 0 {
			runtime.SetCPUProfileRate(def.ProfileRate)
			runtime.MemProfileRate = def.ProfileRate
		}
		if len(def.ProfileBind) > 0 {
			log.Notice("Starting Profiler on %s", def.ProfileBind)
			go http.ListenAndServe(def.ProfileBind, nil)
		} else if len(def.HealthServerBind) > 0 {
			log.Notice("Starting Profiler on %s", def.HealthServerBind)
			go http.ListenAndServe(def.HealthServerBind, nil)

		}
	} else {
		// disable
		runtime.SetBlockProfileRate(0)
		runtime.SetCPUProfileRate(0)
		runtime.MemProfileRate = 0
	}

	// this can be "very expensive" so turnon lightly
	if def.BlockProfile {
		runtime.SetBlockProfileRate(1)
	}

	// see if we can join up to the Gossip land
	if def.Gossip.Enabled {
		_, err := gossip.Start(def.Gossip.Mode, def.Gossip.Port, def.Gossip.Name, def.Gossip.Bind)

		if def.Gossip.Seed == "" {
			log.Noticef("Starting Gossip (master node) on port:%d seed: master", def.Gossip.Port)
		} else {
			log.Noticef("Joining gossip on port:%d seed: %s", def.Gossip.Port, def.Gossip.Seed)
			err = gossip.CadentMembers.Join(def.Gossip.Seed)

		}

		if err != nil {
			panic("Failed to join gossip: " + err.Error())
		}

	}

	// set goodies in the shared data
	shared.Set("hashers", config)
	shared.Set("is_writer", false) // these will get overridden later if there are these
	shared.Set("is_reader", false)
	shared.Set("is_hasher", false)

	if len(config) > 1 {
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
		err = config.VerifyAndAssignPreReg(pr)
		if err != nil {
			log.Critical("Error parsing PreReg: %s", err)
			os.Exit(1)
		}
		def.PreRegFilters = pr
	}

	//some printstuff to verify settings
	config.DebugConfig()

	setSystemStuff(def.NumProc)

	// main block as we want to "defer" it's removal at main exit
	var pidFile = def.PIDfile
	if pidFile != "" {
		contents, err := ioutil.ReadFile(pidFile)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(contents)))
			if err != nil {
				log.Critical("Error reading proccess id from pidfile '%s': %s", pidFile, err)
				os.Exit(1)
			}

			process, err := os.FindProcess(pid)

			// on Windows, err != nil if the process cannot be found
			if runtime.GOOS == "windows" {
				if err == nil {
					log.Critical("Process %d is already running.", pid)
				}
			} else if process != nil {
				// err is always nil on POSIX, so we have to send the process
				// a signal to check whether it exists
				if err = process.Signal(syscall.Signal(0)); err == nil {
					log.Critical("Process %d is already running.", pid)
				}
			}
		}
		if err = ioutil.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())),
			0644); err != nil {

			log.Critical("Unable to write pidfile '%s': %s", pidFile, err)
		}
		log.Info("Wrote pid to pidfile '%s'", pidFile)
		defer func() {
			if err = os.Remove(pidFile); err != nil {
				log.Errorf("Unable to remove pidfile '%s': %s", pidFile, err)
			}
		}()
	}

	//initiallize the statsd singleton
	cadent.SetUpStatsdClient(def)

	var servers []*cadent.Server
	useconfigs := config.ServableConfigs()

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
	//fire up the http stats if given
	if len(def.HealthServerBind) > 0 {
		startStatsServer(def, servers) // will stick as the main event loop
	}

	wg.Wait()

}
