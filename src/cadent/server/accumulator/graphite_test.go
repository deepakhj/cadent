package accumulator

import (
	"bytes"
	"cadent/server/repr"
	. "github.com/smartystreets/goconvey/convey"
	"strings"
	"testing"
)

func TestGraphiteAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls
	statter, err := NewAccumulatorItem("graphite")
	grp, err := NewFormatterItem("graphite")
	statter.Init(grp)

	Convey("Given an GraphiteAcc w/ Graphite Formatter", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org 1")
		Convey("`moo.goo.org 1` should fail", func() {
			So(err, ShouldNotEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:1")
		Convey("`moo.goo.org:1` should  fail", func() {
			So(err, ShouldNotEqual, nil)
		})
		err = statter.ProcessLine("moo.goo.org 1 123123")
		Convey("`moo.goo.org 1 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org 2 123123")
		Convey("`moo.goo.org 2 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})
		err = statter.ProcessLine("moo.goo.max 2 123123")
		err = statter.ProcessLine("moo.goo.max 5 123123")
		err = statter.ProcessLine("moo.goo.max 10 123123")

		err = statter.ProcessLine("stats.counters.goo 2 123123")
		err = statter.ProcessLine("stats.counters.goo 5 123123")
		err = statter.ProcessLine("stats.counters.goo 10 123123")

		err = statter.ProcessLine("stats.gauges.goo 2 123123")
		err = statter.ProcessLine("stats.gauges.goo 5 123123")
		err = statter.ProcessLine("stats.gauges.goo 10 123123")

		buf := new(bytes.Buffer)
		b_arr := statter.Flush(buf)
		for _, item := range b_arr.Stats {
			t.Logf("Graphite Line: %s", item.Name.Key)
		}
		Convey("Flush should give an array of 4 ", func() {
			So(len(b_arr.Stats), ShouldEqual, 4)
		})

		// taggin support
		err = statter.ProcessLine("stats.gauges.goo 10 123123 moo=house host=me")
		buf = new(bytes.Buffer)
		b_arr = statter.Flush(buf)
		for _, item := range b_arr.Stats {
			So(item.Name.MetaTags, ShouldResemble, repr.SortingTags{
				[]string{"moo", "house"},
				[]string{"host", "me"},
			})
		}

		So(strings.Split(buf.String(), "\n")[0], ShouldEqual, "stats.gauges.goo 10.000000 123123 moo=house host=me")

	})
	stsfmt, err := NewFormatterItem("statsd")
	statter.Init(stsfmt)
	Convey("Set the formatter to Statsd ", t, func() {

		err = statter.ProcessLine("moo.goo.max 2 123123")
		err = statter.ProcessLine("moo.goo.max 5 123123")
		err = statter.ProcessLine("moo.goo.max 10 123123")

		err = statter.ProcessLine("moo.goo.min 2 123123")
		err = statter.ProcessLine("moo.goo.min 5 123123")
		err = statter.ProcessLine("moo.goo.min 10 123123")

		err = statter.ProcessLine("moo.goo.avg 2 123123")
		err = statter.ProcessLine("moo.goo.avg 5 123123")
		err = statter.ProcessLine("moo.goo.avg 10 123123")

		err = statter.ProcessLine("stats.counters.goo 2 123123")
		err = statter.ProcessLine("stats.counters.goo 5 123123")
		err = statter.ProcessLine("stats.counters.goo 10 123123")

		buf := new(bytes.Buffer)
		b_arr := statter.Flush(buf)
		Convey("statsd out: Flush should give us data", func() {
			So(len(b_arr.Stats), ShouldEqual, 4)
		})
		for _, item := range b_arr.Stats {
			t.Logf("Statsd Line: %s", item.Name.Key)

		}
	})
}
