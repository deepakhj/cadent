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

package config

import (
	"cadent/server/stats"
	"statsd"
	"time"
)

var memStats *stats.MemStats

type StatsdConfig struct {
	// send some stats to the land
	StatsdServer          string  `toml:"server" json:"server,omitempty"`
	StatsdPrefix          string  `toml:"prefix" json:"prefix,omitempty"`
	StatsdInterval        uint    `toml:"interval" json:"interval,omitempty"`
	StatsdSampleRate      float32 `toml:"sample_rate" json:"sample_rate,omitempty"`
	StatsdTimerSampleRate float32 `toml:"timer_sample_rate" json:"timer_sample_rate,omitempty"`
}

//init a statsd client from our config object
func (cfg *StatsdConfig) Start() {

	if len(cfg.StatsdServer) == 0 {
		log.Notice("Skipping Statsd setup, no server specified")
		stats.StatsdClient = new(statsd.StatsdNoop)
		stats.StatsdClientSlow = new(statsd.StatsdNoop)
		return
	}
	interval := time.Second * 2 // aggregate stats and flush every 2 seconds
	statsdclient := statsd.NewStatsdClient(cfg.StatsdServer, cfg.StatsdPrefix+".%HOST%.")
	statsdclientslow := statsd.NewStatsdClient(cfg.StatsdServer, cfg.StatsdPrefix+".%HOST%.")

	if cfg.StatsdTimerSampleRate > 0 {
		statsdclient.TimerSampleRate = cfg.StatsdTimerSampleRate
	}
	if cfg.StatsdSampleRate > 0 {
		statsdclient.SampleRate = cfg.StatsdSampleRate
	}
	//statsdclient.CreateSocket()
	//StatsdClient = statsdclient
	//return StatsdClient

	// the buffer client seems broken for some reason
	if cfg.StatsdInterval > 0 {
		interval = time.Second * time.Duration(cfg.StatsdInterval)
	}
	statsder := statsd.NewStatsdBuffer("fast", interval, statsdclient)
	statsderslow := statsd.NewStatsdBuffer("slow", interval, statsdclientslow)
	statsder.RetainKeys = true //retain statsd keys to keep emitting 0's
	if cfg.StatsdTimerSampleRate > 0 {
		statsder.TimerSampleRate = cfg.StatsdTimerSampleRate
	}
	if cfg.StatsdSampleRate > 0 {
		statsder.SampleRate = cfg.StatsdSampleRate
	}

	stats.StatsdClient = statsder
	stats.StatsdClientSlow = statsderslow // slow does not have sample rates enabled
	log.Notice("Statsd Fast Client to %s, prefix %s, interval %d", cfg.StatsdServer, cfg.StatsdPrefix, cfg.StatsdInterval)
	log.Notice("Statsd Slow Client to %s, prefix %s, interval %d", cfg.StatsdServer, cfg.StatsdPrefix, cfg.StatsdInterval)

	// start the MemTicker
	memStats = new(stats.MemStats)
	memStats.Start()
	return
}
