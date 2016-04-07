package main

// a little "blaster of stats" to stress test te consthasher

import (
	"cadent/server/netpool"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var randWords = []string{"test", "house", "here", "badline", "cow", "now"}
var sentLines int64
var startTime = time.Now().Unix()
var ConnectionTimeout, _ = time.ParseDuration("2s")
var Statsdtypes = []string{"c", "g", "ms"}

var conLock sync.Mutex
var outCons map[*url.URL]net.Conn

// random char gen
func RandChars(length uint) string {

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
	cc := Statsdtypes[rand.Intn(len(Statsdtypes))]
	sample := float32(rand.Intn(1000)) / float32(1000)
	return fmt.Sprintf(
		"statdtest.%s.%s.%s:%d|%s|@%0.2f\n",
		RandItem(randWords),
		RandItem(randWords),
		RandItem(randWords),
		ct,
		cc,
		sample,
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

func Conn(i_url *url.URL) (net.Conn, error) {
	conLock.Lock()
	defer conLock.Unlock()
	c, ok := outCons[i_url]
	if ok {
		return c, nil
	}
	conn, err := netpool.NewWriterConn(i_url.Scheme, i_url.Host+i_url.Path, ConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("Error in Connection: %s", err)
	}
	outCons[i_url] = conn
	return conn, nil
}

func SendMsg(i_url *url.URL, msg string) {
	conn, err := Conn(i_url)
	if err != nil {
		log.Printf("Error in Connection: %s", err)
		return
	}
	to_send := []byte(msg)
	_, err = conn.Write(to_send)
	if err != nil {
		log.Printf("Error in Write: %s", err)
		conLock.Lock()
		defer conLock.Unlock()
		delete(outCons, i_url)
	}
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
	for {
		s_delta := time.Now().UnixNano() - startTime
		t_sec := s_delta / 1000000000
		rate := (float64)(sentLines) / float64(t_sec)
		log.Printf("Sent %d lines in %ds - rate: %0.2f lines/s", sentLines, t_sec, rate)
		tick_sleep, _ := time.ParseDuration("5s")
		time.Sleep(tick_sleep)
	}
}

func main() {
	setUlimits()
	rand.Seed(time.Now().UnixNano())
	startTime = time.Now().UnixNano()
	serverList := flag.String("servers", "tcp://127.0.0.1:8125", "list of servers to open (tcp://127.0.0.1:6002,tcp://127.0.0.1:6003), you can choose tcp://, udp://, http://, unix://")
	intype := flag.String("type", "statsd", "statsd or graphite")
	rate := flag.String("rate", "0.1s", "fire rate for stat lines")
	buffer := flag.Int("buffer", 512, "send buffer")
	concur := flag.Int("forks", 2, "number of concurrent senders")
	words := flag.String("words", "test,house,here,there,badline,cow,now", "compose the stat keys from these words")

	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}
	outCons = make(map[*url.URL]net.Conn)
	server_split := strings.Split(*serverList, ",")
	if len(*words) > 0 {
		randWords = strings.Split(*words, ",")
	}
	for _, serv := range server_split {
		for i := 0; i < *concur; i++ {
			go Runner(serv, *intype, *rate, *buffer)
		}
	}
	StatTick()
}
