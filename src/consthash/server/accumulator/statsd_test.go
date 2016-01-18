package accumulator

import (
	. "github.com/smartystreets/goconvey/convey"
	"strings"
	"testing"
	"time"
)

func TestStatsdAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls
	grp, err := NewFormatterItem("graphite")
	statter, err := NewAccumulatorItem("statsd")
	statter.Init(grp)

	Convey("Given an StatsdAccumulate w/ Graphite Formatter", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org 1")
		Convey("`moo.goo.org 1` should fail", func() {
			So(err, ShouldNotEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:1")
		Convey("`moo.goo.org:1` should not fail", func() {
			So(err, ShouldEqual, nil)
		})
		err = statter.ProcessLine("moo.goo.org:1|c")
		Convey("`moo.goo.org:1|c` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:1|ms")
		Convey("`moo.goo.org:1|ms` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:3|ms|@0.01")
		Convey("`moo.goo.org:1|ms|@0.01` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:3|ms|@gg")
		Convey("`moo.goo.org:1|ms|@gg` should fail", func() {
			So(err, ShouldNotEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:1|h")
		Convey("`moo.goo.org:1|h` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:1|g")
		err = statter.ProcessLine("moo.goo.org:+1|g")
		err = statter.ProcessLine("moo.goo.org:-5|g")
		Convey("`moo.goo.org.gauge:1|g` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		b_arr := statter.Flush()
		Convey("Flush should give an array bigger then 2 ", func() {
			So(len(b_arr), ShouldBeGreaterThanOrEqualTo, 2)
		})
		got_gauge := ""
		got_counter := ""
		got_timer := ""
		for _, item := range b_arr {
			t.Logf("Graphite Line: %s", item)
			if strings.Contains(item, ".gauges") {
				got_gauge = item
			}
			if strings.Contains(item, ".timers") {
				got_timer = item
			}
			if strings.Contains(item, ".counters") {
				got_counter = item
			}
		}
		Convey("Have the Gauge item ", func() {
			So(len(got_gauge), ShouldBeGreaterThan, 0)
		})
		Convey("Have the Timer item ", func() {
			So(len(got_timer), ShouldBeGreaterThan, 0)
		})
		Convey("Have the Counter item ", func() {
			So(len(got_counter), ShouldBeGreaterThan, 0)
		})

	})
	stsfmt, err := NewFormatterItem("statsd")
	statter.Init(stsfmt)
	Convey("Set the formatter to Statsd ", t, func() {

		err = statter.ProcessLine("moo.goo.org:1|c")
		Convey("statsd out: `moo.goo.org:1|c` should not fail", func() {
			So(err, ShouldEqual, nil)
		})
		err = statter.ProcessLine("moo.goo.org:1")
		Convey("statsd out: `moo.goo.org:1` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:1|g")
		Convey("statsd out: `moo.goo.org:1|g` should not fail", func() {
			So(err, ShouldEqual, nil)
		})
		Convey("statsd out: type ", func() {
			So(stsfmt.Type(), ShouldEqual, "statsd_formater")
		})
		err = statter.ProcessLine("moo.goo.org:1|ms|@0.1")
		Convey("statsd out: `moo.goo.org:1|ms|@0.1` should not fail", func() {
			So(err, ShouldEqual, nil)
		})
		// let some time pass
		time.Sleep(time.Second)
		err = statter.ProcessLine("moo.goo.org:2|ms|@0.1")

		b_arr := statter.Flush()
		Convey("statsd out: Flush should give us data", func() {
			So(len(b_arr), ShouldBeGreaterThan, 3)
		})
		got_timer := ""
		have_upper := ""
		for _, item := range b_arr {
			t.Logf("Statsd Line: %s", item)
			if strings.Contains(item, "stats.timers") {
				got_timer = item
			}
			if strings.Contains(item, "moo.goo.org.upper_95:20.000000|g") {
				have_upper = item
			}

		}
		Convey("Statsd should not have stats.timers ", func() {
			So(len(got_timer), ShouldEqual, 0)
		})
		Convey("Statsd should have proper upper_95 ", func() {
			So(len(have_upper), ShouldNotEqual, 0)
		})
	})
}
