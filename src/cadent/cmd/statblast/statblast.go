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

// a little "blaster of stats" to stress test te consthasher

import (
	"cadent/server/netpool"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

var t_now time.Time

func init() {
	rand.Seed(time.Now().Unix())
	t_now = time.Now()
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var randWords = []string{"test", "house", "here", "badline", "cow", "now"}
var randTagName = []string{"host", "code", "direction", "type", "dummy", "moomoo"}
var randUnit = []string{"B", "Hz", "ft", "J", "jiff"}
var randMtype = []string{"rate", "counter", "gauge"}
var randStat = []string{"min", "max", "mean", "min_90", "max_90"}
var sentLines int64
var numWords int = 3
var startTime = time.Now().Unix()
var ConnectionTimeout, _ = time.ParseDuration("2s")
var Statsdtypes = []string{"c", "g", "ms", "s", "h"}

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
	// use the key string to determine "int scale" of things
	key := sprinter(numWords)
	tn := int(time.Now().Hour())
	offset := 5000*len(key) - 1000*int(math.Sin(float64(tn/24)))
	return fmt.Sprintf(
		"graphitetest.%s %d %d\n",
		key,
		offset+ct,
		time.Now().Unix(),
	)
}

func sprinter(ct int) string {
	r_ws := []string{RandItem(randWords)}

	for i := 0; i < ct-1; i++ {
		r_ws = append(r_ws, RandItem(randWords))
	}
	return strings.Join(r_ws, ".")
}

func sprinterTag(ct int) string {
	// add the "required" tags
	r_ws := []string{"what=" + RandItem(randWords), "unit=" + RandItem(randUnit), "mtype=" + RandItem(randMtype), "stat=" + RandItem(randStat)}

	for i := 0; i < ct-4; i++ {
		r_ws = append(r_ws, RandItem(randTagName)+"="+RandItem(randWords))
	}
	return strings.Join(r_ws, " ")
}

func StatsdStr(ct int) string {
	cc := Statsdtypes[rand.Intn(len(Statsdtypes))]
	sample := float32(rand.Intn(1000)) / float32(1000)
	return fmt.Sprintf(
		"statdtest.%s:%d|%s|@%0.2f\n",
		sprinter(numWords),
		ct,
		cc,
		sample,
	)
}

func Carbon2Str(ct int) string {
	// use the key string to determine "int scale" of things
	key := sprinterTag(numWords)
	tn := int(time.Now().Hour())
	offset := 5000*len(key) - 1000*int(math.Sin(float64(tn/24)))
	return fmt.Sprintf(
		"%s %d %d\n",
		key,
		offset+ct,
		time.Now().Unix(),
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
	var err error
	var i_url *url.URL
	to_stdout := false

	switch server {
	case "stdout":
		to_stdout = true
	default:
		i_url, err = url.Parse(server)
		if err != nil {
			log.Printf("Error in URL: %s", err)
			os.Exit(1)
		}
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
		switch intype {
		case "graphite":
			oneline = GraphiteStr(ct)
		case "statsd":
			oneline = StatsdStr(ct)
		case "carbon2":
			oneline = Carbon2Str(ct)
		default:
			log.Fatalf("Invalid stat type %s", intype)
		}

		switch to_stdout {
		case true:
			fmt.Print(oneline)
			sentLines++
			time.Sleep(sleeper)
		default:
			if len(msg+oneline) >= buffer {
				SendMsg(i_url, msg)
				msg = oneline
				time.Sleep(sleeper)
			} else {
				msg = msg + oneline
				sentLines++
			}
		}

	}
}

func StatTick(print_stats bool) {
	for {
		s_delta := time.Now().UnixNano() - startTime
		t_sec := s_delta / 1000000000
		rate := (float64)(sentLines) / float64(t_sec)
		if print_stats {
			log.Printf("Sent %d lines in %ds - rate: %0.2f lines/s", sentLines, t_sec, rate)
		}
		tick_sleep, _ := time.ParseDuration("5s")
		time.Sleep(tick_sleep)
	}
}

func main() {
	setUlimits()
	rand.Seed(time.Now().UnixNano())
	startTime = time.Now().UnixNano()
	serverList := flag.String("servers", "tcp://127.0.0.1:8125", "list of servers to open (stdout,tcp://127.0.0.1:6002,tcp://127.0.0.1:6003), you can choose tcp://, udp://, http://, unix://")
	intype := flag.String("type", "statsd", "statsd or graphite or carbon2")
	rate := flag.String("rate", "0.1s", "fire rate for stat lines")
	buffer := flag.Int("buffer", 512, "send buffer")
	concur := flag.Int("forks", 2, "number of concurrent senders")
	words := flag.String("words", "test,house,here,there,badline,cow,now", "compose the stat keys from these words")
	words_p_str := flag.Int("words_per_stat", 3, "make stat keys this long (moo.goo.loo)")
	print_stats := flag.Bool("notick", false, "don't print the stats of the number of lines sent (if servers==stdout this is true)")

	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	}
	outCons = make(map[*url.URL]net.Conn)
	server_split := strings.Split(*serverList, ",")
	numWords = *words_p_str
	if len(*words) > 0 {
		randWords = strings.Split(*words, ",")
	}
	for _, serv := range server_split {
		for i := 0; i < *concur; i++ {
			go Runner(serv, *intype, *rate, *buffer)
		}
	}
	to_print := *print_stats
	if *serverList == "stdout" {
		to_print = true
	}
	StatTick(!to_print)
}
