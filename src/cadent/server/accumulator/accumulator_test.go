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

package accumulator

import (
	"cadent/server/splitter"
	"testing"
	"time"
	//_ "net/http/pprof"
	//"net/http"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAccumualtorAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls
	//profiler
	//go http.ListenAndServe(":6065", nil)

	fail_acc, err := NewAccumlator("monkey", "graphite", false, "monkeytest")
	Convey("Bad formatter name `monkey` should fail ", t, func() {
		Convey("Error should not be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
		Convey("Error acc should be nil", func() {
			So(fail_acc, ShouldEqual, nil)
		})
	})

	fail_acc, err = NewAccumlator("graphite", "monkey", false, "monkeytest")
	Convey("Bad accumulator name `monkey` should fail ", t, func() {
		Convey("Error should not be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
		Convey("Error acc should be nil", func() {
			So(fail_acc, ShouldEqual, nil)
		})
	})

	grph_acc, err := NewAccumlator("graphite", "graphite", false, "monkeytest")

	Convey("Graphite to graphite -> graphite accumulator should be ok", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})

		err = grph_acc.Accumulate.ProcessLine("moo.goo.max 2 123123")
		err = grph_acc.Accumulate.ProcessLine("moo.goo.max 5 123123")
		err = grph_acc.Accumulate.ProcessLine("moo.goo.max 10 123123")

		err = grph_acc.Accumulate.ProcessLine("stats.counters.goo 2 123123")
		err = grph_acc.Accumulate.ProcessLine("stats.counters.goo 5 123123")
		err = grph_acc.Accumulate.ProcessLine("stats.counters.goo 10 123123")

		err = grph_acc.Accumulate.ProcessLine("stats.gauges.goo 2 123123")
		err = grph_acc.Accumulate.ProcessLine("stats.gauges.goo 5 123123")
		err = grph_acc.Accumulate.ProcessLine("stats.gauges.goo 10 123123")

		b_arr, _ := grph_acc.Flush()
		for _, item := range b_arr {
			t.Logf("Graphite Line: Key: %s, Line: %s, Phase: %v", item.Key(), item.Line(), item.Phase())
		}
		Convey("Flush should give an array of 3 ", func() {
			So(len(b_arr), ShouldEqual, 3)
		})

		b_arr, _ = grph_acc.Flush()
		Convey("Flush should be empty ", func() {
			So(len(b_arr), ShouldEqual, 0)
		})
		Convey("Logger for coverage ", func() {
			grph_acc.LogConfig()
		})
	})
	statsd_acc, err := NewAccumlator("statsd", "statsd", true, "statsdstatsd")
	Convey("Statsd to statsd accumulator should be ok", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})
		Convey("Error acc should not be nil", func() {
			So(statsd_acc, ShouldNotEqual, nil)
		})
	})
	statsd_acc, err = NewAccumlator("statsd", "graphite", false, "statsdgraphite")
	Convey("Statsd to graphite accumulator should be ok", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})
		Convey("Error acc should not be nil", func() {
			So(statsd_acc, ShouldNotEqual, nil)
		})
	})

	tickC := make(chan splitter.SplitItem, 128)
	statsd_acc.Accumulate.SetOptions([][]string{
		{"legacyNamespace", "true"},
		{"prefixGauge", "gauges"},
		{"prefixTimer", "timers"},
		{"prefixCounter", "counters"},
		{"globalPrefix", ""},
		{"globalSuffix", "stats"},
		{"percentThreshold", "0.75,0.90,0.95,0.99"},
	})
	statsd_acc.FlushTimes = []time.Duration{time.Duration(time.Second)}
	statsd_acc.SetOutputQueue(tickC)

	Convey("statsd accumluator flush timer", t, func() {
		go statsd_acc.Start()
		tt := time.NewTimer(time.Duration(2 * time.Second))
		err = statsd_acc.ProcessLine("moo.goo.poo:12|c")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.1|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.1|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.1|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.5|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.3|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.5|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.7|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.7|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.7|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.2|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.goo:24|c")
		err = statsd_acc.ProcessLine("moo.goo.goo:24|c")
		err = statsd_acc.ProcessLine("moo.goo.loo:36|c")
		err = statsd_acc.ProcessLine("moo.goo.loo||||36|c")
		outs := []splitter.SplitItem{}
		t.Logf("LineQueue %d", len(statsd_acc.LineQueue))
		t_f := func() {
			for {
				select {
				case <-tt.C:
					t.Logf("Stopping accumuator after %d", 2*time.Second)
					statsd_acc.Stop()
					return

				case l := <-tickC:
					outs = append(outs, l)
					t.Logf("FlushLine %s", l.Line())
				}
			}
		}
		t_f()
		Convey("should have 30 flushed lines", func() {
			So(len(outs), ShouldEqual, 30)
		})
	})

	// test the keep keys
	statsd_acc, err = NewAccumlator("statsd", "graphite", true, "statsdgraphite")
	statsd_acc.Accumulate.SetOptions([][]string{
		{"legacyNamespace", "true"},
		{"prefixGauge", "gauges"},
		{"prefixTimer", "timers"},
		{"prefixCounter", "counters"},
		{"globalPrefix", ""},
		{"globalSuffix", "stats"},
		{"percentThreshold", "0.75,0.90,0.95,0.99"},
	})
	statsd_acc.FlushTimes = []time.Duration{time.Duration(time.Second)}
	statsd_acc.SetOutputQueue(tickC)

	Convey("statsd accumluator flush timer", t, func() {
		//time.Sleep(2 * time.Second) // wait for things to kick off

		// should "flush" 4 times, the first w/30 lines
		// the next 3 with only 12

		err = statsd_acc.ProcessLine("moo.goo.poo:12|c")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.1|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.1|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.1|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.5|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.3|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.5|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.7|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.7|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.7|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.2|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.goo:24|c")
		err = statsd_acc.ProcessLine("moo.goo.goo:24|c")
		err = statsd_acc.ProcessLine("moo.goo.loo:36|c")
		err = statsd_acc.ProcessLine("moo.goo.loo||||36|c")
		outs := []splitter.SplitItem{}
		t.Logf("Stats -> Graphite :: LineQueue %d", len(statsd_acc.LineQueue))
		go statsd_acc.Start()
		t_f := func() {
			tt := time.NewTimer(time.Duration(5 * time.Second))
			for {
				select {
				case <-tt.C:
					t.Logf("Stopping accumuator after %d", 2*time.Second)
					statsd_acc.Stop()
					stats, _ := json.Marshal(statsd_acc.CurrentStats())
					t.Logf("Current Stats: %s", stats)
					return

				case l := <-tickC:
					outs = append(outs, l)
					t.Logf("FlushLine %s", l.Line())
				}
			}
		}
		t_f()
		Convey("should have 72 flushed lines", func() {
			So(len(outs), ShouldEqual, 72)
		})
	})

	Convey("graphite accumluator flush timer", t, func() {
		// test the keep keys
		tickG := make(chan splitter.SplitItem, 1000)

		graphite_acc, _ := NewAccumlator("graphite", "graphite", true, "graphitegraphite")
		graphite_acc.FlushTimes = []time.Duration{time.Duration(time.Second)}
		graphite_acc.SetOutputQueue(tickG)
		go graphite_acc.Start()
		//time.Sleep(2 * time.Second) // wait for things to kick off
		// should "flush" 4 times, the first w/2 lines
		// the next 3 with only 2

		err = graphite_acc.ProcessLine("moo.goo.poo 12 1465867151")
		err = graphite_acc.ProcessLine("moo.goo.poo 35 1465867152")
		err = graphite_acc.ProcessLine("moo.goo.poo 66 1465867153")
		err = graphite_acc.ProcessLine("moo.goo.loo 100 1465867150")
		err = graphite_acc.ProcessLine("moo.goo.loo 100 1465867150")
		err = graphite_acc.ProcessLine("moo.goo.loo 100 1465867150")

		outs := []splitter.SplitItem{}
		t.Logf("Graphite -> Graphite:: LineQueue %d", len(graphite_acc.LineQueue))

		t_f := func() {
			tt := time.NewTimer(time.Duration(5 * time.Second))
			for {
				select {
				case <-tt.C:
					t.Logf("Stopping accumuator after %d", 2*time.Second)
					graphite_acc.Stop()
					stats, _ := json.Marshal(graphite_acc.CurrentStats())
					t.Logf("Current Stats: %s", stats)
					return

				case l := <-tickG:
					outs = append(outs, l)
					t.Logf("FlushLine %s", l.Line())
				}
			}
		}

		t_f()
		graphite_acc.Stop()
		// note there are 16 becuase
		// we "keep the keys" and a the flush is every second, for 5 seconds (the last one
		// second is canceled above)... of the 6 above stats 4 are "distinct" in the one
		// second flush window, thus 4*4=16
		Convey("should have 16 flushed lines", func() {
			So(len(outs), ShouldEqual, 16)
		})

	})

}
