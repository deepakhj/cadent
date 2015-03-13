package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const (
	DEFAULT_INDEX_STATS_FILE = "index.html"
)

// need to up this guy otherwise we quickly run out of sockets
func setSystemStuff(num_procs int) {
	if num_procs <= 0 {
		num_procs = runtime.NumCPU()
	}
	log.Println("[System] Setting GOMAXPROCS to ", num_procs)

	runtime.GOMAXPROCS(num_procs)

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("[System] Error Getting Rlimit: ", err)
	}
	log.Println("[System] Current Rlimit: ", rLimit)

	rLimit.Max = 999999
	rLimit.Cur = 999999
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("[System] Error Setting Rlimit: ", err)
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Println("[System] Error Getting Rlimit:  ", err)
	}
	log.Println("[System] Final Rlimit Final: ", rLimit)
}

// Fire up the http server for stats and healthchecks
func startStatsServer(defaults *Config, servers []*Server) {

	log.Printf("Starting Status server on %s", defaults.HealthServerBind)

	var names []string
	for _, serv := range servers {
		names = append(names, serv.Name)
		serv.AddStatusHandlers()
	}

	fileserve := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		http.ServeFile(w, r, path.Join(defaults.HealthServerPath, DEFAULT_INDEX_STATS_FILE))
	}

	status := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		fmt.Fprintf(w, "ok")
	}
	stats := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		jsonp := r.URL.Query().Get("jsonp")

		stats_map := make(map[string]*ServerStats)
		for idx, serv := range servers {
			// note the server itself populates this ever 5 seconds
			stats_map[serv.Name] = &servers[idx].stats
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

	http.HandleFunc("/", fileserve)
	http.HandleFunc("/ops/status", status)
	http.HandleFunc("/status", status)
	http.HandleFunc("/stats", stats)

	log.Fatal(http.ListenAndServe(defaults.HealthServerBind, nil))

}

func main() {
	configFile := flag.String("config", "config.toml", "Consitent Hash configuration file")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	var config ConfigServers
	var err error

	config, err = parseConfigFile(*configFile)
	if err != nil {
		log.Printf("Error decoding config file: %s", err)
		os.Exit(1)
	}

	config.debugConfig()

	def, err := config.defaultConfig()
	if err != nil {
		log.Printf("Error decoding config file: Could not find default: %s", err)
		os.Exit(1)
	}

	setSystemStuff(def.NumProc)

	// main block as we want to "defer" it's removal at main exit
	var pidFile = def.PIDfile
	if pidFile != "" {
		contents, err := ioutil.ReadFile(pidFile)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(contents)))
			if err != nil {
				log.Fatalf("Error reading proccess id from pidfile '%s': %s", pidFile, err)
			}

			process, err := os.FindProcess(pid)

			// on Windows, err != nil if the process cannot be found
			if runtime.GOOS == "windows" {
				if err == nil {
					log.Fatalf("Process %d is already running.", pid)
				}
			} else if process != nil {
				// err is always nil on POSIX, so we have to send the process
				// a signal to check whether it exists
				if err = process.Signal(syscall.Signal(0)); err == nil {
					log.Fatalf("Process %d is already running.", pid)
				}
			}
		}
		if err = ioutil.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())),
			0644); err != nil {

			log.Fatalf("Unable to write pidfile '%s': %s", pidFile, err)
		}
		log.Printf("Wrote pid to pidfile '%s'", pidFile)
		defer func() {
			if err = os.Remove(pidFile); err != nil {
				log.Printf("Unable to remove pidfile '%s': %s", pidFile, err)
			}
		}()
	}

	if def.Profile {
		log.Println("Starting Profiler on localhost:6060")
		go http.ListenAndServe(":6060", nil)
	}

	//initiallize the statsd singleton
	SetUpStatsdClient(def)

	var servers []*Server
	useconfigs := config.ServableConfigs()

	for _, cfg := range useconfigs {

		var hashers []*ConstHasher

		for _, serverlist := range cfg.ServerLists {
			hasher, err := createConstHasherFromConfig(&cfg, serverlist)

			if err != nil {
				panic(err)
			}
			go hasher.ServerPool.startChecks()
			hashers = append(hashers, hasher)
		}
		server, err := CreateServer(&cfg, hashers)
		if err != nil {
			panic(err)
		}
		servers = append(servers, server)
		go server.StartServer()
	}

	//fire up the http stats if given
	if len(def.HealthServerBind) != 0 {
		startStatsServer(def, servers)
	} else {
		//now we just need to loop
		for {
		}
	}

}
