package main

// A simply "echo" server to run on a few ports for testing
import (
	"bufio"
	"flag"
	"fmt"
	"github.com/davecheney/profile"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func echoMe(line string) {
	if len(line) > 0 {
		log.Println(line)

	}
}

var TotalLines uint64

type EchoStat struct {
	NumConnections       uint64
	NumOpenConnections   uint64
	NumLines             uint64
	NumClosedConnections uint64
}

type EchoServer interface {
	ReadMessages() (err error)
	String() string
	EchoStats()
}

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
	log.Printf("Server %s: {ValidConnections: %d, OpenConnections: %d, Lines: %d, Total: %d}",
		echo.String(),
		echo.StatCt.NumOpenConnections-echo.StatCt.NumClosedConnections,
		echo.StatCt.NumLines,
		echo.StatCt.NumConnections,
		TotalLines)

	time.Sleep(time.Duration(5 * time.Second))
	go echo.EchoStats()
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
					atomic.AddUint64(&TotalLines, 1)
					echoMe("[ECHO from " + echo.String() + "] " + strings.Trim(line, " \t\n"))
				}
			}
			atomic.AddUint64(&echo.StatCt.NumClosedConnections, 1)

		}

	}

	return nil
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
	listener, err := net.Listen(listen.Scheme, listen.Host)
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
	log.Printf("Server %s: {ValidConnections: %d, OpenConnections: %d, Lines: %d, Total: %d}",
		echo.String(),
		echo.StatCt.NumOpenConnections-echo.StatCt.NumClosedConnections,
		echo.StatCt.NumLines,
		echo.StatCt.NumConnections,
		TotalLines)

	time.Sleep(time.Duration(5 * time.Second))
	go echo.EchoStats()
}

func (echo EchoServerTCP) ReadMessages() (err error) {
	for {
		conn, err := echo.Listener.Accept()
		defer func() {

		}()
		if err != nil {
			log.Printf("Error connection from %s", err)
			return err
		}
		if conn != nil {
			atomic.AddUint64(&echo.StatCt.NumConnections, 1)
			atomic.AddUint64(&echo.StatCt.NumOpenConnections, 1)
		}

		log.Printf("Accepted connection from %s", conn.RemoteAddr())
		buf := bufio.NewReader(conn)
		for {
			line, err := buf.ReadString('\n')

			if err != nil || len(line) == 0 {
				break
			}
			atomic.AddUint64(&TotalLines, 1)
			atomic.AddUint64(&echo.StatCt.NumLines, 1)
			echoMe("[ECHO from " + echo.String() + "] " + strings.Trim(line, " \t\n"))

		}
		if conn != nil {
			conn.Close()
		}
		log.Print("Close")
		atomic.AddUint64(&echo.StatCt.NumClosedConnections, 1)
	}

	return nil
}

func createServer(listen *url.URL) (EchoServer, error) {

	if listen.Scheme == "udp" {
		return CreateServerUDP(listen)
	}
	return CreateServerTCP(listen)

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
	serverList := flag.String("servers", "tcp://127.0.0.1:6002", "list of servers to open (tcp://127.0.0.1:6002,tcp://127.0.0.1:6003)")
	cpuProfile := flag.String("profile", "", "CPU profile? to which file")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}

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
		for idx, _ := range srv_list[0 : num_s-1] {
			go startServer(srv_list[idx])
		}
	}
	startServer(srv_list[num_s-1])
}
