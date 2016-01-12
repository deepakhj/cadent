package stats

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestStatsAtomic(t *testing.T) {
	// Only pass t into top-level Convey calls
	Convey("Given an AtomicInt", t, func() {
		var atm AtomicInt
		atm.Set(1)
		Convey("Should be 1", func() {
			So(atm.Get(), ShouldEqual, 1)
		})
		atm.Add(2)
		Convey("Adding 2 should be 3", func() {
			So(atm.Get(), ShouldEqual, 3)
		})

		Convey("To String 2 should be 3", func() {
			So(atm.String(), ShouldEqual, "3")
		})
	})
}

func TestStatsCounter(t *testing.T) {
	// Only pass t into top-level Convey calls
	Convey("Given an Counter", t, func() {
		var ctr StatCount
		Convey("initial should be 0", func() {
			So(ctr.TotalCount.Get(), ShouldEqual, 0)
			So(ctr.TickCount.Get(), ShouldEqual, 0)
		})
		ctr.Up(1)

		Convey("Adding 1 should be 1", func() {
			So(ctr.TotalCount.Get(), ShouldEqual, 1)
			So(ctr.TickCount.Get(), ShouldEqual, 1)
		})
		rt, _ := time.ParseDuration("1s")
		Convey("Rate 1 should be about 1", func() {
			So(ctr.TotalRate(rt), ShouldEqual, 1.0)
			So(ctr.Rate(rt), ShouldEqual, 1.0)
		})
		ctr.Reset()
		Convey("Reset should take us to 0", func() {
			So(ctr.TotalCount.Get(), ShouldEqual, 0)
			So(ctr.TickCount.Get(), ShouldEqual, 0)
			So(ctr.TotalRate(rt), ShouldEqual, 0)
			So(ctr.Rate(rt), ShouldEqual, 0)
		})
		ctr.Up(1)
		ctr.ResetTick()
		Convey("ResetTick should take us to 0", func() {

			So(ctr.TotalCount.Get(), ShouldEqual, 1)
			So(ctr.TickCount.Get(), ShouldEqual, 0)
		})

	})
}
func TestStatsStatsd(t *testing.T) {
	// Only pass t into top-level Convey calls
	Convey("Given statsd", t, func() {
		Convey("initial should be nil", func() {
			So(StatsdClient, ShouldEqual, nil)
		})
	})
}
