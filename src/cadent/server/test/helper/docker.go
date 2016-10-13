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

// some helpers for testing various components that are hard to "mock"
// like databases and what not, here we have a docker compose "fire-upper"

package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"
)

const BASE_COMPOSE = "../../../../../docker-compose.yml"
const BASE_NAME = "cadent_"

func doPath() {
	oldpth := os.Getenv("PATH")
	if !strings.Contains(oldpth, "/usr/local/bin") {
		os.Setenv("PATH", "/usr/local/bin:"+oldpth)
	}
	oldpth = os.Getenv("PATH")
	if !strings.Contains(oldpth, "/usr/bin") {
		os.Setenv("PATH", "/usr/bin:"+oldpth)
	}
	oldpth = os.Getenv("PATH")
	if !strings.Contains(oldpth, "/bin") {
		os.Setenv("PATH", "/bin:"+oldpth)
	}

}

func myPath() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

func isRunning(which string) bool {
	doPath()
	cmd := exec.Command("docker", "ps")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return strings.Contains(out.String(), fmt.Sprintf("%s%s", BASE_NAME, which))
}

func DockerUp(which string) bool {
	doPath()
	pth := myPath()
	compose_pth := path.Join(pth, BASE_COMPOSE)
	log.Print(compose_pth)
	if isRunning(which) {
		return true
	}
	cmd := exec.Command("docker-compose", "-f", compose_pth, "up", "-d", which)
	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		log.Fatalf("%s :: %v", errb.String(), err)
		return false
	}
	log.Printf(out.String())
	return true
}

func DockerDown(which string) bool {
	doPath()
	pth := myPath()
	compose_pth := path.Join(pth, BASE_COMPOSE)
	if !isRunning(which) {
		return true
	}
	cmd := exec.Command("docker-compose", "-f", compose_pth, "stop", which)
	var out, oerr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &oerr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
		return false
	}
	log.Printf(out.String())
	log.Printf(oerr.String())

	cmd = exec.Command("docker-compose", "-f", compose_pth, "rm", "-f", which)
	cmd.Stdout = &out
	cmd.Stderr = &oerr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
		return false
	}
	log.Printf(out.String())
	log.Printf(oerr.String())

	return true
}

func DockerIp() string {
	var out, oerr bytes.Buffer
	cmd := exec.Command("docker-machine", "ip")
	cmd.Stdout = &out
	cmd.Stderr = &oerr
	err := cmd.Run()
	if err != nil {
		// best guess for non-docker machine stuff
		return "127.0.0.1"
	}
	return strings.TrimSpace(out.String())
}

func DockerExposedPorts(which string) ([]string, error) {
	doPath()
	if !isRunning(which) {
		return nil, fmt.Errorf("container not running")
	}
	var out, oerr bytes.Buffer
	cmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("name=%s%s_1", BASE_NAME, which))
	cmd.Stdout = &out
	cmd.Stderr = &oerr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	image := strings.TrimSpace(out.String())
	cmd = exec.Command("docker", "inspect", "-f", `"{{json .NetworkSettings.Ports }}"`, image)
	var jout, joerr bytes.Buffer
	cmd.Stdout = &jout
	cmd.Stderr = &joerr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	str := jout.String()
	str = str[1 : len(str)-2]
	item := make(map[string][]map[string]string, 0)
	err = json.Unmarshal([]byte(str), &item)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	out_ps := make([]string, 0)
	for _, v := range item {
		for _, bits := range v {
			if bits["HostIp"] == "0.0.0.0" {
				p := bits["HostPort"]
				out_ps = append(out_ps, p)
			}
		}
	}
	return out_ps, nil
}

func DockerWaitUntilReady(which string) bool {
	doPath()

	ports, err := DockerExposedPorts(which)
	if err != nil {
		return false
	}

	if len(ports) == 0 {
		return true
	}

	tick := time.NewTimer(10 * time.Second)
	got_c := make(chan bool, len(ports))
	ok_ct := 0

	try_conn := func(port string) {
		tick_c := time.NewTimer(10 * time.Second)
		log.Printf("Testing connection on port %s", port)
		for {
			select {
			case <-tick_c.C:
				log.Printf("Failed to connection Timeout")
				return
			default:
				log.Printf("Trying :%s ...", port)
				conn, err := net.DialTimeout("tcp", fmt.Sprintf(":%s", port), time.Second)
				if err != nil {
					log.Printf("Failed to connection: %v", err)
					time.Sleep(time.Second)
				} else {
					log.Printf("Got connection: %v", conn.RemoteAddr())
					conn.Close()
					got_c <- true
					return
				}
			}
		}
	}

	for _, p := range ports {
		go try_conn(p)
	}
	for {
		select {
		case <-tick.C:
			return false
		case <-got_c:
			ok_ct++
			if ok_ct == len(ports) {
				return true
			}
		}
	}
}
