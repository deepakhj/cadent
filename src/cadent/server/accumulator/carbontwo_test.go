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

		err = statter.ProcessLine("moo.goo.org 1")
		Convey("`moo.goo.org 1` should fail", func() {
			So(err, ShouldNotEqual, nil)
		})

		err = statter.ProcessLine("moo.goo.org:1")
		Convey("`moo.goo.org:1` should  fail", func() {
			So(err, ShouldNotEqual, nil)
		})

		err = statter.ProcessLine("stat=max mtype=gauge what=house 123 123123")
		Convey("`type=min mtype=gauge what=house 123 123123` should  fail", func() {
			So(err, ShouldEqual, errCarbonTwoUnitRequired)
		})

		err = statter.ProcessLine("stat=max unit=B what=house 123 123123")
		Convey("`type=min unit=B what=house 123 123123` should  fail", func() {
			So(err, ShouldEqual, errCarbonTwoMTypeRequired)
		})

		err = statter.ProcessLine("stat=max unit=B mtype=gauge what=house 123 123123")
		Convey("`type=min unit=B mtype=gauge what=house 123 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("stat=max unit=B mtype=gauge what=house  moo=goo house=spam 123 123123")
		Convey("`moo.goo.org 2 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("stat=max.unit=B.mtype=gauge.what=house  moo=goo house=spam 123 123123")
		Convey("type=min.unit=B.mtype=gauge.what=house  moo=goo house=spam 123 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		err = statter.ProcessLine("stat=max,unit=B,mtype=gauge,what=house  moo=goo house=spam 123 123123")
		Convey("type=min,unit=B,mtype=gauge,what=house  moo=goo house=spam 123 123123` should not fail", func() {
			So(err, ShouldEqual, nil)
		})

		// clear it out
		b := new(devNullWriter)
		statter.Flush(b)

		err = statter.ProcessLine("stat=max unit=B mtype=gauge what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=max unit=B mtype=gauge what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=max unit=B mtype=gauge what=house  moo=goo house=spam 10 123123")

		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 10 123123")

		err = statter.ProcessLine("stat=mean unit=B mtype=gauge what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=mean unit=B mtype=gauge what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=mean unit=B mtype=gauge what=house  moo=goo house=spam 10 123123")

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
		So(strs, ShouldContain, "mtype=gauge.stat=max.unit=B.what=house 10.000000 123123 moo=goo house=spam")
		So(strs, ShouldContain, "mtype=gauge.stat=min.unit=B.what=house 2.000000 123123 moo=goo house=spam")
		So(strs, ShouldContain, "mtype=gauge.stat=mean.unit=B.what=house 5.666667 123123 moo=goo house=spam")

	})
	stsfmt, err := NewFormatterItem("statsd")
	statter.Init(stsfmt)
	Convey("Set the formatter to Statsd ", t, func() {

		err = statter.ProcessLine("stat=max unit=B mtype=counter what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=max unit=B mtype=counter what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=max unit=B mtype=counter what=house  moo=goo house=spam 10 123123")

		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 10 123123")

		err = statter.ProcessLine("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 10 123123")

		buf := new(bytes.Buffer)
		b_arr := statter.Flush(buf)
		Convey("statsd out: Flush should give us data", func() {
			So(len(b_arr.Stats), ShouldEqual, 3)
		})
		strs := strings.Split(buf.String(), "\n")
		So(strs, ShouldContain, "mtype=counter.stat=max.unit=B.what=house:10.000000|c")
		So(strs, ShouldContain, "mtype=gauge.stat=min.unit=B.what=house:2.000000|g")
		So(strs, ShouldContain, "mtype=rate.stat=mean.unit=B.what=house:5.666667|ms")

	})

	carbfmt, err := NewFormatterItem("carbon2")
	statter.Init(carbfmt)
	Convey("Set the formatter to carbon2 ", t, func() {

		err = statter.ProcessLine("stat=max unit=B mtype=counter what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=max unit=B mtype=counter what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=max unit=B mtype=counter what=house  moo=goo house=spam 10 123123")

		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=min unit=B mtype=gauge what=house  moo=goo house=spam 10 123123")

		err = statter.ProcessLine("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 2 123123")
		err = statter.ProcessLine("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 5 123123")
		err = statter.ProcessLine("stat=mean unit=B mtype=rate what=house  moo=goo house=spam 10 123123")

		err = statter.ProcessLine("stat=mean unit=B mtype=rate what=monkey  10 123123")

		buf := new(bytes.Buffer)
		b_arr := statter.Flush(buf)
		Convey("carbon2 out: Flush should give us data", func() {
			So(len(b_arr.Stats), ShouldEqual, 4)
		})
		strs := strings.Split(buf.String(), "\n")
		So(strs, ShouldContain, "mtype=counter stat=max unit=B what=house  moo=goo house=spam 10.000000 123123")
		So(strs, ShouldContain, "mtype=gauge stat=min unit=B what=house  moo=goo house=spam 2.000000 123123")
		So(strs, ShouldContain, "mtype=rate stat=mean unit=B what=house  moo=goo house=spam 5.666667 123123")
		So(strs, ShouldContain, "mtype=rate stat=mean unit=B what=monkey 10.000000 123123")

	})
}
