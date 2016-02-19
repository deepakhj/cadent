package accumulator

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	//"time"
)

func TestAccumualtorAggLoop(t *testing.T) {
	// Only pass t into top-level Convey calls

	//profiler
	//go http.ListenAndServe(":6065", nil)
	conf_str := `
	backend = "BLACKHOLE"
	input_format = "graphite"
	output_format = "graphite"
	[keeper]
	times = ["1s:5s"]
	[writer]
	driver = "moo"
	dsn = "/tmp/tt"
	`
	_, err := ParseConfigString(conf_str)

	Convey("Aggloop should fail", t, func() {
		Convey("Error should not be nil", func() {
			So(err, ShouldNotEqual, nil)
		})

	})

	conf_str = `
	backend = "BLACKHOLE"
	input_format = "graphite"
	output_format = "graphite"
	[keeper]
	times = ["3s:5s"]
	[writer]
	driver = "file"
	dsn = "/tmp/tt"
	`

	cf, err := DecodeConfigString(conf_str)

	Convey("Aggloop should have been configed properly", t, func() {
		Convey("Error should not be nil", func() {
			So(err, ShouldEqual, nil)
		})
	})

	Convey("Aggloop created properly", t, func() {
		cf.ParseDurations()

		agg, err := NewAggregateLoop(cf.durations, cf.ttls, "tester")
		Convey("Aggregator should init", func() {
			So(agg, ShouldNotEqual, nil)
		})

		err = agg.SetWriter(cf.Writer)
		Convey("Aggregator writer should init", func() {
			So(err, ShouldEqual, nil)
		})

	})
}
