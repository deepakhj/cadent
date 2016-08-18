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

	carbfmt, err := NewFormatterItem("carbon2")
	statter.Init(carbfmt)
	Convey("Set the formatter to carbon2 ", t, func() {

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
		Convey("carbon2 out: Flush should give us data", func() {
			So(len(b_arr.Stats), ShouldEqual, 4)
		})
		strs := strings.Split(buf.String(), "\n")
		t.Logf(strings.Join(strs, "\n"))
		So(strs, ShouldContain, "mtype=count unit=jiff what=stats.counters.goo 17.000000 123123")
		So(strs, ShouldContain, "mtype=count unit=jiff what=moo.goo.avg 5.666667 123123")
		So(strs, ShouldContain, "mtype=count unit=jiff what=moo.goo.min 2.000000 123123")
		So(strs, ShouldContain, "mtype=count unit=jiff what=moo.goo.max 10.000000 123123")

		statter.SetTags(repr.SortingTags{[]string{"moo", "goo"}, []string{"foo", "bar"}})

		err = statter.ProcessLine("stats.counters.goo 2 123123")
		err = statter.ProcessLine("stats.counters.goo 5 123123")
		err = statter.ProcessLine("stats.counters.goo 10 123123")

		buf = new(bytes.Buffer)
		b_arr = statter.Flush(buf)
		strs = strings.Split(buf.String(), "\n")
		t.Logf(strings.Join(strs, "\n"))
		So(strs, ShouldContain, "mtype=count unit=jiff what=stats.counters.goo  foo=bar moo=goo 17.000000 123123")

	})
}
