package accumulator

import (
	"cadent/server/splitter"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
	//_ "net/http/pprof"
	//"net/http"
	"encoding/json"
)

func TestAccumualtorAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls

	//profiler
	//go http.ListenAndServe(":6065", nil)

	fail_acc, err := NewAccumlator("monkey", "graphite", false)
	Convey("Bad formatter name `monkey` should faile ", t, func() {
		Convey("Error should not be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
		Convey("Error acc should be nil", func() {
			So(fail_acc, ShouldEqual, nil)
		})
	})

	fail_acc, err = NewAccumlator("graphite", "monkey", false)
	Convey("Bad accumulator name `monkey` should faile ", t, func() {
		Convey("Error should not be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
		Convey("Error acc should be nil", func() {
			So(fail_acc, ShouldEqual, nil)
		})
	})

	grph_acc, err := NewAccumlator("graphite", "graphite", false)

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
	statsd_acc, err := NewAccumlator("statsd", "statsd", true)
	Convey("Statsd to statsd accumulator should be ok", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})
		Convey("Error acc should not be nil", func() {
			So(statsd_acc, ShouldNotEqual, nil)
		})
	})
	statsd_acc, err = NewAccumlator("statsd", "graphite", false)
	Convey("Statsd to graphite accumulator should be ok", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})
		Convey("Error acc should not be nil", func() {
			So(statsd_acc, ShouldNotEqual, nil)
		})
	})

	tickC := make(chan splitter.SplitItem)
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
			return
		}
		t_f()
		Convey("should have 30 flushed lines", func() {
			So(len(outs), ShouldEqual, 30)
		})
	})

	// test the keep keys
	statsd_acc, err = NewAccumlator("statsd", "graphite", true)
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
			return
		}
		t_f()
		Convey("should have 72 flushed lines", func() {
			So(len(outs), ShouldEqual, 72)
		})
	})

	// test the keep keys
	tickG := make(chan splitter.SplitItem, 1000)

	graphite_acc, err := NewAccumlator("graphite", "graphite", true)
	graphite_acc.FlushTimes = []time.Duration{time.Duration(time.Second)}
	graphite_acc.SetOutputQueue(tickG)

	Convey("graphite accumluator flush timer", t, func() {
		go graphite_acc.Start()
		//time.Sleep(2 * time.Second) // wait for things to kick off
		// should "flush" 4 times, the first w/2 lines
		// the next 3 with only 2
		err = graphite_acc.ProcessLine("moo.goo.poo 12 123123")
		err = graphite_acc.ProcessLine("moo.goo.poo 35 123124")
		err = graphite_acc.ProcessLine("moo.goo.poo 66 123125")
		err = graphite_acc.ProcessLine("moo.goo.loo 100 123123")
		err = graphite_acc.ProcessLine("moo.goo.loo 100 123123")
		err = graphite_acc.ProcessLine("moo.goo.loo 100 123123")

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
			return
		}

		t_f()
		Convey("should have 8 flushed lines", func() {
			So(len(outs), ShouldEqual, 8)
		})

	})

}
