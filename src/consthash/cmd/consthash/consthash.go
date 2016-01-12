package main

import (
	consthash "consthash/server"
	prereg "consthash/server/prereg"
	"encoding/json"
	"flag"
	"fmt"
	logging "github.com/op/go-logging"
	"io/ioutil"
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

// compile passing -ldflags "-X main.Build <build sha1>"
var ConstHashBuild string

var log = logging.MustGetLogger("main")

func logInit() {
	var format = "%{color}%{time:2006-01-02 15:04:05.000} [%{module}] ▶ %{level:.4s} %{color:reset} %{message}"
	logBackend := logging.NewLogBackend(os.Stderr, "", 0)
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

// Fire up the http server for stats and healthchecks
func startStatsServer(defaults *consthash.Config, servers []*consthash.Server) {

	log.Notice("Starting Status server on %s", defaults.HealthServerBind)

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

		stats_map := make(map[string]*consthash.ServerStats)
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
		hasher_map := make(map[string]consthash.ServerHashCheck)

		for idx, serv := range servers {
			// note the server itself populates this ever 5 seconds
			hasher_map[serv.Name] = servers[idx].HasherCheck(h_key)
		}
		w.Header().Set("Content-Type", "application/json")
		resbytes, _ := json.Marshal(hasher_map)
		fmt.Fprintf(w, string(resbytes))

	}

	http.HandleFunc("/", fileserve)
	http.HandleFunc("/ops/status", status)
	http.HandleFunc("/status", status)
	http.HandleFunc("/stats", stats)
	http.HandleFunc("/hashcheck", hashcheck)

	err := http.ListenAndServe(defaults.HealthServerBind, nil)
	if err != nil {
		log.Critical("Could not start http server %s", defaults.HealthServerBind)
		os.Exit(1)
	}

}

func main() {
	version := flag.Bool("version", false, "Print version and exit")
	configFile := flag.String("config", "config.toml", "Consitent Hash configuration file")
	regConfigFile := flag.String("prereg", "", "File that contains the Regex/Filtering by key to various backends")
	flag.Parse()

	if *version {
		fmt.Printf("ConstHash version %s\n\n", ConstHashBuild)
		os.Exit(0)
	}

	logInit()

	log.Info("ConstHash version %s", ConstHashBuild)

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	var config consthash.ConfigServers
	var err error

	config, err = consthash.ParseConfigFile(*configFile)
	if err != nil {
		log.Info("Error decoding config file: %s", err)
		os.Exit(1)
	}

	def, err := config.DefaultConfig()
	if err != nil {
		log.Critical("Error decoding config file: Could not find default: %s", err)
		os.Exit(1)
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
				log.Notice("Unable to remove pidfile '%s': %s", pidFile, err)
			}
		}()
	}

	if def.Profile {
		log.Notice("Starting Profiler on localhost:6060")
		go http.ListenAndServe(":6060", nil)
	}

	//initiallize the statsd singleton
	consthash.SetUpStatsdClient(def)

	var servers []*consthash.Server
	useconfigs := config.ServableConfigs()

	for _, cfg := range useconfigs {

		var hashers []*consthash.ConstHasher

		for _, serverlist := range cfg.ServerLists {
			hasher, err := consthash.CreateConstHasherFromConfig(cfg, serverlist)

			if err != nil {
				panic(err)
			}
			go hasher.ServerPool.StartChecks()
			hashers = append(hashers, hasher)
		}
		server, err := consthash.CreateServer(cfg, hashers)
		if err != nil {
			panic(err)
		}
		servers = append(servers, server)
	}
	// finally we need to set the accumulator backends
	for _, srv := range consthash.SERVER_BACKENDS {

		//set the accumulator to this servers input queue
		if srv.Queue.PreRegFilter != nil && srv.Queue.PreRegFilter.Accumulator != nil {
			to_srv := consthash.SERVER_BACKENDS[srv.Queue.PreRegFilter.Accumulator.ToBackend]
			srv.Queue.PreRegFilter.Accumulator.OutputQueue = to_srv.Queue.InputQueue
		}
		// start them up
		go srv.Queue.StartServer()
	}

	//fire up the http stats if given
	if len(def.HealthServerBind) != 0 {
		startStatsServer(def, servers) // will stick as the main event loop
	} else {
		//now we just need to loop forever!
		for {
		}
	}

}
