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

// A simply "echo" server to run on a few ports for testing
import (
	"bufio"
	"flag"
	"fmt"
	"github.com/davecheney/profile"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var QUIET bool = false

func echoMe(line string) {
	if len(line) > 0 && !QUIET {
		log.Println(line)
	}
}

var TotalLines uint64

type EchoStat struct {
	NumConnections       uint64
	NumOpenConnections   uint64
	NumLines             uint64
	NumClosedConnections uint64
	NumCurLines          uint64
}

type EchoServer interface {
	ReadMessages() (err error)
	String() string
	EchoStats()
}

/** HTTP **/

type EchoServerHTTP struct {
	StatCt   *EchoStat
	Listen   *url.URL
	Listener *http.Server
}

func CreateServerHTTP(listen *url.URL) (EchoServerHTTP, error) {

	log.Printf("Binding server to %s", listen.String())
	serv := new(EchoServerHTTP)
	serv.Listen = listen

	serv.StatCt = new(EchoStat)
	return *serv, nil
}

func (echo EchoServerHTTP) ReadMessages() (err error) {
	// no op for HTTP
	Handler := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		go func() {
			atomic.AddUint64(&echo.StatCt.NumConnections, 1)
			atomic.AddUint64(&echo.StatCt.NumOpenConnections, 1)
			str := strings.Split(string(body), "\n")
			for _, line := range str {
				atomic.AddUint64(&echo.StatCt.NumLines, 1)
				atomic.AddUint64(&echo.StatCt.NumCurLines, 1)
				atomic.AddUint64(&TotalLines, 1)
				echoMe("[ECHO from " + echo.Listen.String() + "] " + strings.Trim(line, " \t\n"))
			}
			atomic.AddUint64(&echo.StatCt.NumClosedConnections, 1)
		}()
	}
	pth := echo.Listen.Path
	if len(pth) == 0 {
		pth = "/"
	}
	// multi http servers needs new muxers
	serverMux := http.NewServeMux()
	serverMux.HandleFunc(pth, Handler)
	err = http.ListenAndServe(echo.Listen.Host, serverMux)
	if err != nil {
		return fmt.Errorf("Error binding: %s", err)
	}
	return nil
}

func (echo EchoServerHTTP) EchoStats() {
	for {
		log.Printf("Server %s: ValidConnections: %d, OpenConnections: %d, Lines: %d, Lines/s: %0.2f Total: %d",
			echo.String(),
			echo.StatCt.NumConnections,
			echo.StatCt.NumOpenConnections-echo.StatCt.NumClosedConnections,
			echo.StatCt.NumLines,
			float32(echo.StatCt.NumCurLines)/float32(5.0),
			TotalLines,
		)
		atomic.StoreUint64(&echo.StatCt.NumCurLines, 0)
		time.Sleep(time.Duration(5 * time.Second))
	}
}

func (echo EchoServerHTTP) String() string {
	return echo.Listen.String()
}

/** UDP **/

type EchoServerUDP struct {
	StatCt   *EchoStat
	Listen   *url.URL
	Listener *net.UDPConn
}

func CreateServerUDP(listen *url.URL) (EchoServerUDP, error) {

	log.Printf("Binding server to %s", listen.String())
	serv := new(EchoServerUDP)
	serv.Listen = listen

	udpArr, err := net.ResolveUDPAddr(listen.Scheme, listen.Host)
	if err != nil {
		return *serv, fmt.Errorf("Error binding: %s", err)
	}
	listener, err := net.ListenUDP(listen.Scheme, udpArr)
	if err != nil {
		return *serv, fmt.Errorf("Error binding: %s", err)
	}
	serv.Listener = listener
	serv.StatCt = new(EchoStat)
	return *serv, nil
}

func (echo EchoServerUDP) String() string {
	return echo.Listen.String()
}
func (echo EchoServerUDP) EchoStats() {
	for {
		log.Printf("Server %s: ValidConnections: %d, OpenConnections: %d, Lines: %d, Lines/s: %0.2f Total: %d",
			echo.String(),
			echo.StatCt.NumConnections,
			echo.StatCt.NumOpenConnections-echo.StatCt.NumClosedConnections,
			echo.StatCt.NumLines,
			float32(echo.StatCt.NumCurLines)/float32(5.0),
			TotalLines,
		)
		atomic.StoreUint64(&echo.StatCt.NumCurLines, 0)
		time.Sleep(time.Duration(5 * time.Second))
	}
}
func (echo EchoServerUDP) ReadMessages() (err error) {
	for {
		var buf []byte = make([]byte, 1024)
		atomic.AddUint64(&echo.StatCt.NumConnections, 1)
		atomic.AddUint64(&echo.StatCt.NumOpenConnections, 1)

		n, address, err := echo.Listener.ReadFromUDP(buf)

		if err != nil {
			fmt.Println("error reading data from connection")
			fmt.Println(err)
			return err
		}
		if address != nil {
			if n > 0 {
				str := strings.Split(string(buf), "\n")
				for _, line := range str {
					atomic.AddUint64(&echo.StatCt.NumLines, 1)
					atomic.AddUint64(&echo.StatCt.NumCurLines, 1)
					atomic.AddUint64(&TotalLines, 1)
					echoMe("[ECHO from " + echo.String() + "] " + strings.Trim(line, " \t\n"))
				}
			}
			atomic.AddUint64(&echo.StatCt.NumClosedConnections, 1)

		}

	}
}

type EchoServerTCP struct {
	StatCt   *EchoStat
	Listen   *url.URL
	Listener net.Listener
}

func CreateServerTCP(listen *url.URL) (EchoServerTCP, error) {
	log.Printf("Binding server to %s", listen.String())
	serv := new(EchoServerTCP)
	serv.Listen = listen
	listener, err := net.Listen(listen.Scheme, listen.Host+listen.Path)
	if err != nil {
		return *serv, fmt.Errorf("Error binding: %s", err)
	}
	serv.Listener = listener
	serv.StatCt = new(EchoStat)
	return *serv, nil

}
func (echo EchoServerTCP) String() string {
	return echo.Listen.String()
}

func (echo EchoServerTCP) EchoStats() {
	for {
		log.Printf("Server %s: ValidConnections: %d, OpenConnections: %d, Lines: %d, Lines/s: %0.2f Total: %d",
			echo.String(),
			echo.StatCt.NumConnections,
			echo.StatCt.NumOpenConnections-echo.StatCt.NumClosedConnections,
			echo.StatCt.NumLines,
			float32(echo.StatCt.NumCurLines)/float32(5.0),
			TotalLines,
		)
		atomic.StoreUint64(&echo.StatCt.NumCurLines, 0)
		time.Sleep(time.Duration(5 * time.Second))
	}
}

func (echo EchoServerTCP) ReadMessages() (err error) {

	//drop the socket if closed
	if echo.Listen.Scheme == "unix" {
		defer os.Remove(echo.Listen.Host + echo.Listen.Path)
	}
	for {
		conn, err := echo.Listener.Accept()

		if err != nil {
			log.Printf("Error connection from %s", err)
			return err
		}
		if conn != nil {
			atomic.AddUint64(&echo.StatCt.NumConnections, 1)
			atomic.AddUint64(&echo.StatCt.NumOpenConnections, 1)
		}

		log.Printf("[%s] Accepted connection from %s", echo.String(), conn.RemoteAddr())
		buf := bufio.NewScanner(conn)
		for buf.Scan() {
			line := buf.Text()
			line = strings.Trim(line, " \t\n")
			if len(line) == 0 {
				continue
			}
			atomic.AddUint64(&TotalLines, 1)
			atomic.AddUint64(&echo.StatCt.NumLines, 1)
			atomic.AddUint64(&echo.StatCt.NumCurLines, 1)

			echoMe("[ECHO from " + echo.String() + "] " + line)
		}

		if conn != nil {
			conn.Close()
		}
		log.Print("Close")
		atomic.AddUint64(&echo.StatCt.NumClosedConnections, 1)
	}
}

func createServer(listen *url.URL) (EchoServer, error) {

	if listen.Scheme == "udp" {
		return CreateServerUDP(listen)
	} else if listen.Scheme == "http" {
		return CreateServerHTTP(listen)
	} else {
		return CreateServerTCP(listen)
	}
}

func startServer(serv EchoServer) {
	go serv.EchoStats()

	serv.ReadMessages()
}

// need to up this guy otherwise we quickly run out of sockets
func setUlimits() {

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
	setUlimits()
	serverList := flag.String("servers", "tcp://127.0.0.1:6002", "list of servers to open (tcp://127.0.0.1:6002,tcp://127.0.0.1:6003), you can choose tcp://, udp://, unix:///, http://")
	cpuProfile := flag.String("profile", "", "CPU profile? to which file")
	quiet := flag.Bool("quiet", false, "Quiet, just report number lines in")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}
	QUIET = *quiet
	if *cpuProfile != "" {
		cfg := profile.Config{
			CPUProfile:     true,
			MemProfile:     true,
			ProfilePath:    ".",  // store profiles in current directory
			NoShutdownHook: true, // do not hook SIGINT
		}

		// p.Stop() must be called before the program exits to
		// ensure profiling information is written to disk.
		p := profile.Start(&cfg)
		defer p.Stop()
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		go func() {
			<-sigc
			fmt.Printf("Error: user interrupt.")
			p.Stop()
			os.Exit(-1)
		}()
	}

	server_split := strings.Split(*serverList, ",")
	var srv_list []EchoServer

	for _, serv := range server_split {
		i_url, err := url.Parse(serv)
		if err != nil {
			log.Printf("Error in URL: %s", err)
			os.Exit(1)
		}
		server, err := createServer(i_url)
		srv_list = append(srv_list, server)

		if err != nil {
			log.Printf("Error Server Creation: %s", err)
			os.Exit(1)
		}
	}
	// background all but the last one
	num_s := len(srv_list)
	if num_s-2 >= 0 {
		for idx := range srv_list[0 : num_s-1] {
			go startServer(srv_list[idx])
		}
	}
	startServer(srv_list[num_s-1])
}
