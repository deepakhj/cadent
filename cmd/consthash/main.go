package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// need to up this guy otherwise we quickly run out of sockets
func setSystemStuff() {
	fmt.Println("[System] Setting GOMAXPROCS to ", runtime.NumCPU())

	runtime.GOMAXPROCS(runtime.NumCPU())

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println("[System] Error Getting Rlimit: ", err)
	}
	fmt.Println("[System] Current Rlimit: ", rLimit)

	rLimit.Max = 999999
	rLimit.Cur = 999999
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println("[System] Error Setting Rlimit: ", err)
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println("[System] Error Getting Rlimit:  ", err)
	}
	fmt.Println("[System] Final Rlimit Final: ", rLimit)
}

func main() {
	setSystemStuff()
	configFile := flag.String("config", "config.toml", "Consitent Hash configuration file")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	/// main block as we want to "defer" it's removale at main exit
	var pidFile = "/tmp/consthash.pid"
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
	if def.Profile {
		log.Println("Starting Profiler on localhost:6060")
		go http.ListenAndServe(":6060", nil)
	}

	//initiallize the statsd singleton
	SetUpStatsdClient(def)

	hasher, err := createConstHasherFromConfig(def)
	go hasher.ServerPool.startChecks()
	startServer(def, hasher)

}
