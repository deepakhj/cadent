package main

// a little "blaster of stats" to stress test te consthasher

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var randWords = []string{"test", "house", "here", "there", "cow", "now"}
var sentLines int64
var startTime = time.Now().Unix()
var ConnectionTimeout, _ = time.ParseDuration("2s")

// random char gen
func RandChars(chars []rune, length uint) string {

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)

}
func RandItem(strs []string) string {
	return strs[rand.Intn(len(strs))]
}

func GraphiteStr(ct int) string {
	return fmt.Sprintf(
		"graphitetest.%s.%s.%s %d %d\n",
		RandItem(randWords),
		RandItem(randWords),
		RandItem(randWords),
		ct,
		int32(time.Now().Unix()),
	)
}

func StatsdStr(ct int) string {
	return fmt.Sprintf(
		"statdtest.%s.%s.%s:%d\n",
		RandItem(randWords),
		RandItem(randWords),
		RandItem(randWords),
		ct,
	)
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

// helper to "pool" TCP connections otherwise we can quickly nuke things by just sheer
// port overload
var TCPPOOL map[string][]net.Conn
var POOLLIMIT int = 50

func TcpConn(i_url *url.URL) (net.Conn, error) {
	var str = i_url.String()
	var gots []net.Conn
	if _, ok := TCPPOOL[str]; !ok {
		gots = make([]net.Conn, POOLLIMIT)
		for i := 0; i < POOLLIMIT; i++ {
			conn, err := net.DialTimeout(i_url.Scheme, i_url.Host, ConnectionTimeout)
			if err != nil {
				return nil, fmt.Errorf("Could not connect to %s: %s", str, err)
			}
			gots[i] = conn
		}
		TCPPOOL[str] = gots
	}
	return TCPPOOL[str][rand.Intn(POOLLIMIT)], nil
}

func SendMsg(i_url *url.URL, msg string) {
	conn, err := net.DialTimeout(i_url.Scheme, i_url.Host, ConnectionTimeout)
	if err != nil {
		log.Printf("Error in Rate: %s", err)
		return
	}
	to_send := []byte(msg)
	_, err = conn.Write(to_send)
	conn.Close()
}

func Runner(server string, intype string, rate string, buffer int) {

	rand_bounds := 10000
	i_url, err := url.Parse(server)
	if err != nil {
		log.Printf("Error in URL: %s", err)
		os.Exit(1)
	}
	var msg string = ""
	sleeper, err := time.ParseDuration(rate)
	if err != nil {
		log.Printf("Error in Rate: %s", err)
		os.Exit(1)
	}

	for {
		ct := rand.Intn(rand_bounds)
		var oneline string = ""
		if intype == "graphite" {
			oneline = GraphiteStr(ct)
		} else {
			oneline = StatsdStr(ct)
		}

		if len(msg+oneline) >= buffer {
			SendMsg(i_url, msg)
			msg = oneline
			time.Sleep(sleeper)
		} else {
			msg = msg + oneline
			sentLines = sentLines + 1
		}
	}
}

func StatTick() {
	s_delta := time.Now().UnixNano() - startTime
	t_sec := s_delta / 1000000000
	rate := (float64)(sentLines) / float64(t_sec)
	log.Printf("Sent %d lines in %ds - rate: %0.2f lines/s", sentLines, t_sec, rate)
	tick_sleep, _ := time.ParseDuration("5s")
	time.Sleep(tick_sleep)
	StatTick()
}

func main() {
	setUlimits()
	rand.Seed(time.Now().UnixNano())
	startTime = time.Now().UnixNano()
	serverList := flag.String("servers", "tcp://127.0.0.1:8125", "list of servers to open (tcp://127.0.0.1:6002,tcp://127.0.0.1:6003)")
	intype := flag.String("type", "statsd", "statsd or graphite")
	rate := flag.String("rate", "0.1s", "fire rate for stat lines")
	buffer := flag.Int("buffer", 512, "send buffer")
	concur := flag.Int("forks", 2, "number of concurrent senders")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	server_split := strings.Split(*serverList, ",")

	for _, serv := range server_split {
		for i := 0; i < *concur; i++ {
			go Runner(serv, *intype, *rate, *buffer)
		}
	}
	StatTick()
}
