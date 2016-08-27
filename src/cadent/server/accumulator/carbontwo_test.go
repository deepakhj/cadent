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
	. "github.com/smartystreets/goconvey/convey"
	"strings"
	"testing"
)

type devNullWriter struct{}

func (d *devNullWriter) Write(data []byte) (int, error) { return 0, nil }
func (d *devNullWriter) WriteByte(data byte) error      { return nil }

func TestCarbontwoAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls
	grp, err := NewFormatterItem("graphite")
	statter, err := NewAccumulatorItem("carbon2")
	statter.Init(grp)

	Convey("Given an CarbontwoAcc w/ Carbontwo Formatter", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine([]byte("moo.goo.org 1"))
		Convey("`moo.goo.org 1` should fail", func() {
			So(err, ShouldNotEqual, nil)
		})

		err = statter.ProcessLine([]byte("moo.goo.org:1"))
		Convey("`moo.goo.org:1` should  fail", func() {
			So(err, ShouldNotEqual, nil)
		})

		err = statter.ProcessLine([]byte("stat=max mtype=gauge what=house 123 123123"))
		Convey("`type=min mtype=gauge what=house 123 123123` should  fail", func() {
			So(err, ShouldEqual, errCarbonTwoUnitRequired)
		})

		err = statter.ProcessLine([]byte("stat=max unit=B what=house 123 123123"))
		Convey("`type=min unit=B what=house 123 123123` should  fail", func() {
			So(err, ShouldEqual, errCarbonTwoMTypeRequired)
		})

		err = statter.ProcessLine([]byte("stat=max unit=B mtype=gauge what=house 123 123123"))
		Convey("`type=min unit=B mtype=gauge what=house 123 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine([]byte("stat=max unit=B mtype=gauge what=house  moo=goo house=spam 123 123123"))
		Convey("`moo.goo.org 2 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine([]byte("stat=max.unit=B.mtype=gauge.what=house  moo=goo house=spam 123 123123"))
		Convey("type=min.unit=B.mtype=gauge.what=house  moo=goo house=spam 123 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine([]byte("stat=max,unit=B,mtype=gauge,what=house  moo=goo house=spam 123 123123"))
		Convey("type=min,unit=B,mtype=gauge,what=house  moo=goo house=spam 123 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine([]byte("stat_is_max,unit_is_B,mtype_is_gauge,what_is_house  moo_is_goo house_is_spam 123 123123"))
		Convey("type_is_min,unit_is_B,mtype_is_gauge,what_is_house  moo=goo house=spam 123 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		// clear it out
		b := new(devNullWriter)
		statter.Flush(b)

		err = statter.ProcessLine([]byte("stat=max unit=B mtype=gauge what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=max unit=B mtype=gauge what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=max unit=B mtype=gauge what=house  moo=goo house=spam 10 123123"))

		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 10 123123"))

		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=gauge what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=gauge what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=gauge what=house  moo=goo house=spam 10 123123"))

		buf := new(bytes.Buffer)
		b_arr := statter.Flush(buf)

		Convey("Flush should give an array of 3 ", func() {
			So(len(b_arr.Stats), ShouldEqual, 3)
		})

		// taggin support
		for _, item := range b_arr.Stats {
			So(item.Name.MetaTags.Tags(), ShouldResemble, [][]string{
				{"moo", "goo"},
				{"house", "spam"},
			})
		}

		strs := strings.Split(buf.String(), "\n")
		So(strs, ShouldContain, "mtype_is_gauge.stat_is_max.unit_is_B.what_is_house 10.000000 123123 mtype=gauge stat=max unit=B what=house  moo=goo house=spam")
		So(strs, ShouldContain, "mtype_is_gauge.stat_is_min.unit_is_B.what_is_house 2.000000 123123 mtype=gauge stat=min unit=B what=house  moo=goo house=spam")
		So(strs, ShouldContain, "mtype_is_gauge.stat_is_mean.unit_is_B.what_is_house 5.666667 123123 mtype=gauge stat=mean unit=B what=house  moo=goo house=spam")

	})
	stsfmt, err := NewFormatterItem("statsd")
	statter.Init(stsfmt)
	Convey("Set the formatter to Statsd ", t, func() {

		err = statter.ProcessLine([]byte("stat=max unit=B mtype=counter what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=max unit=B mtype=counter what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=max unit=B mtype=counter what=house  moo=goo house=spam 10 123123"))

		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 10 123123"))

		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 10 123123"))

		buf := new(bytes.Buffer)
		b_arr := statter.Flush(buf)
		Convey("statsd out: Flush should give us data", func() {
			So(len(b_arr.Stats), ShouldEqual, 3)
		})
		strs := strings.Split(buf.String(), "\n")
		t.Logf("Stats: %v", strs)
		So(strs, ShouldContain, "mtype_is_counter.stat_is_max.unit_is_B.what_is_house:10.000000|c|#mtype:counter,stat:max,unit:B,what:house,moo:goo,house:spam")
		So(strs, ShouldContain, "mtype_is_gauge.stat_is_min.unit_is_B.what_is_house:2.000000|g|#mtype:gauge,stat:min,unit:B,what:house,moo:goo,house:spam")
		So(strs, ShouldContain, "mtype_is_rate.stat_is_mean.unit_is_B.what_is_house:5.666667|ms|#mtype:rate,stat:mean,unit:B,what:house,moo:goo,house:spam")

	})

	carbfmt, err := NewFormatterItem("carbon2")
	statter.Init(carbfmt)
	Convey("Set the formatter to carbon2 ", t, func() {

		err = statter.ProcessLine([]byte("stat=max unit=B mtype=counter what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=max unit=B mtype=counter what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=max unit=B mtype=counter what=house  moo=goo house=spam 10 123123"))

		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 10 123123"))

		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 2 123123"))
		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 5 123123"))
		err = statter.ProcessLine([]byte("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 10 123123"))

		err = statter.ProcessLine([]byte("stat=count unit=B mtype=rate what=monkey  10 123123"))
		err = statter.ProcessLine([]byte("stat=count unit=B mtype=rate what=monkey  12 123123"))

		buf := new(bytes.Buffer)
		b_arr := statter.Flush(buf)
		Convey("carbon2 out: Flush should give us data", func() {
			So(len(b_arr.Stats), ShouldEqual, 4)
		})
		strs := strings.Split(buf.String(), "\n")

		So(strs, ShouldContain, "mtype=rate stat=count unit=B what=monkey 22.000000 123123")
		So(strs, ShouldContain, "mtype=gauge stat=min unit=B what=house  house=spam moo=goo 2.000000 123123")
		So(strs, ShouldContain, "mtype=rate stat=mean unit=B what=house  house=spam moo=goo 5.666667 123123")
		So(strs, ShouldContain, "mtype=counter stat=max unit=B what=house  house=spam moo=goo 10.000000 123123")

	})
}
