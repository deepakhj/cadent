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

/** "system" config elements **/

package config

import (
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

type SystemConfig struct {
	PIDfile string `toml:"pid_file" json:"pid_file,omitempty"`
	NumProc int    `toml:"num_procs" json:"num_procs,omitempty"`
}

func (c *SystemConfig) Start() {
	// need to up this guy otherwise we quickly run out of sockets
	if c.NumProc <= 0 {
		c.NumProc = runtime.NumCPU()
	}
	log.Notice("[System] Setting GOMAXPROCS to %d", c.NumProc)

	runtime.GOMAXPROCS(c.NumProc)

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

	c.PidFile()
}

func (c *SystemConfig) PidFile() {

	// main block as we want to "defer" it's removal at main exit
	var pidFile = c.PIDfile
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
		} else {
			log.Info("Wrote pid to pidfile '%s'", pidFile)
		}
	}
}
