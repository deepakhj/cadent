// a simple atomic stat counter/rate
package main

import (
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"./statsd"
)

//statsd client singleton
var StatsdClient statsd.Statsd = nil

//init a statsd client
func SetUpStatsdClient(cfg *Config) statsd.Statsd {

	if len(cfg.StatsdServer) == 0 {
		log.Printf("Skipping Statsd setup, no server specified")
		StatsdClient = new(statsd.StatsdNoop)
		return StatsdClient
	}
	interval := time.Second * 2 // aggregate stats and flush every 2 seconds
	statsdclient := statsd.NewStatsdClient(cfg.StatsdServer, cfg.StatsdPrefix+".%HOST%.")
	//statsdclient.CreateSocket()
	//StatsdClient = statsdclient
	//return StatsdClient

	// the buffer client seems broken for some reason
	if cfg.StatsdInterval > 0 {
		interval = time.Second * time.Duration(cfg.StatsdInterval)
	}
	stats := statsd.NewStatsdBuffer(interval, statsdclient)
	stats.RetainKeys = true //retain statsd keys to keep emitting 0's
	StatsdClient = stats
	log.Printf("Statsd Client to %s, prefix %s, interval %d", cfg.StatsdServer, cfg.StatsdPrefix, cfg.StatsdInterval)
	return StatsdClient
}

//a handy "defer" function for timers, in Nano seconds
func StatsdNanoTimeFunc(statname string, start time.Time) {
	elapsed := time.Since(start)
	StatsdClient.Timing(statname, int64(elapsed))
}

// An AtomicInt is an int64 to be accessed atomically.
type AtomicInt int64

// Add atomically adds n to i.
func (i *AtomicInt) Add(n int64) int64 {
	return atomic.AddInt64((*int64)(i), n)
}

// Get atomically gets the value of i.
func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

// Set the int to an arb number to a number
func (i *AtomicInt) Set(n int64) {
	atomic.StoreInt64((*int64)(i), n)
}

func (i *AtomicInt) String() string {
	return strconv.FormatInt(i.Get(), 10)
}

type StatCount struct {
	Name       string
	TotalCount AtomicInt
	TickCount  AtomicInt
}

func (stat *StatCount) Up(val uint64) {
	stat.TotalCount.Add(1)
	stat.TickCount.Add(1)
}
func (stat *StatCount) ResetTick() {
	stat.TickCount.Set(0)
}
func (stat *StatCount) Rate(duration time.Duration) float32 {
	if stat.TickCount == 0 {
		return 0
	}
	return float32(stat.TickCount) / float32(duration/time.Second)
}

func (stat *StatCount) TotalRate(duration time.Duration) float32 {
	if stat.TotalCount == 0 {
		return 0
	}
	return float32(stat.TotalCount) / float32(duration/time.Second)
}
func (stat *StatCount) Reset() {
	stat.TickCount.Set(0)
	stat.TotalCount.Set(0)
}
