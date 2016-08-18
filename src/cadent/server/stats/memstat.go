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

// use the statsd clients to emit GoLang GC bits
package stats

import (
	"fmt"
	"runtime"
	"statsd"
	"time"
)

type MemStats struct {
	running  bool
	statsd   statsd.Statsd
	prefix   string
	tick     time.Duration
	shutdown chan bool
}

func (ms *MemStats) Start() {
	if ms.running {
		return
	}

	ms.running = true
	ms.statsd = StatsdClientSlow
	ms.prefix = "gogc"
	ms.tick = time.Duration(time.Second * 1)
	ms.shutdown = make(chan bool)
	go ms.statsTick()
}

func (ms *MemStats) Stop() {
	ms.running = false
	ms.shutdown <- true
}

func (ms *MemStats) statsTick() {

	memStats := &runtime.MemStats{}
	var lastPauseNs uint64 = 0
	var lastNumGc uint32 = 0

	nsInmS := float64(time.Millisecond)
	ticker := time.NewTicker(ms.tick)
	for {
		select {
		case <-ticker.C:
			runtime.ReadMemStats(memStats)
			ms.statsd.Gauge(fmt.Sprintf("%s.goroutines", ms.prefix), int64(runtime.NumGoroutine()))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.allocated", ms.prefix), int64(memStats.Alloc))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.mallocs", ms.prefix), int64(memStats.Mallocs))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.frees", ms.prefix), int64(memStats.Frees))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.heap.alloc", ms.prefix), int64(memStats.HeapAlloc))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.heap.sys", ms.prefix), int64(memStats.HeapSys))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.heap.released", ms.prefix), int64(memStats.HeapReleased))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.heap.inuse", ms.prefix), int64(memStats.HeapInuse))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.heap.objects", ms.prefix), int64(memStats.HeapObjects))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.heap.idle", ms.prefix), int64(memStats.HeapIdle))
			ms.statsd.Gauge(fmt.Sprintf("%s.memory.stack", ms.prefix), int64(memStats.StackInuse))

			ms.statsd.FGauge(
				fmt.Sprintf("%s.memory.gc.total_pause_ns", ms.prefix),
				float64(memStats.PauseTotalNs),
			)
			ms.statsd.Incr(fmt.Sprintf("%s.memory.gc.gcs", ms.prefix), int64(memStats.NumGC))

			if lastPauseNs > 0 {
				pauseSinceLastSample := memStats.PauseTotalNs - lastPauseNs
				ms.statsd.FGauge(
					fmt.Sprintf("%s.memory.gc.pause_ms_per_second", ms.prefix),
					float64(pauseSinceLastSample)/nsInmS/ms.tick.Seconds(),
				)
			}
			lastPauseNs = memStats.PauseTotalNs

			countGc := int(memStats.NumGC - lastNumGc)
			if lastNumGc > 0 {
				diff := float64(countGc)
				ms.statsd.FGauge(
					fmt.Sprintf("%s.memory.gc.gc_per_second", ms.prefix),
					diff/ms.tick.Seconds(),
				)

			}
			lastNumGc = memStats.NumGC
		case <-ms.shutdown:
			ms.running = false
			ticker.Stop()
			return
		}
	}
}
