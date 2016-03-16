package accumulator

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestConfigAccumualtorAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls

	//profiler
	//go http.ListenAndServe(":6065", nil)

	conf_test := `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	accumulate_flush = "1s"
	times = ["5s", "1m", "10m"]
	[[tags]]
	key="foo"
	value="bar"
	[[tags]]
	key="loo"
	value="moo"

	[writer]
	driver="file"
	dsn="/tmp/none"

	`

	acc_c, err := ParseConfigString(conf_test)
	Convey("Config toml should parse to a good accumulator", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})
		Convey("Error acc should not be nil", func() {
			So(acc_c, ShouldNotEqual, nil)
		})

		t.Logf("%v", acc_c)

		Convey("Should have 3 timers", func() {
			So(len(acc_c.FlushTimes), ShouldEqual, 3)
		})

		Convey("First Flush time should be 5 seconds", func() {
			So(acc_c.FlushTimes[0], ShouldEqual, time.Duration(5*time.Second))
		})
		Convey("Should have 2 tags", func() {
			So(len(acc_c.Accumulate.Tags()), ShouldEqual, 2)
		})

	})

	conf_test = `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	times = ["5s", "31s", "10m"]
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml keeper should not have proper mutiples", t, func() {
		Convey("Error should NOT be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	times = ["1m", "5s", "10m"]
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml keeper should not have proper order", t, func() {
		Convey("Error should NOT be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	times = ["1m:7asd", "5s:30d", "10m:1y"]
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml keeper should fail on TTL", t, func() {
		Convey("Error should NOT be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	times = ["5s:168h", "1m:720h", "10m:17520h"]
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml keeper should parse correctly", t, func() {
		Convey("Error should NOT be nil", func() {
			So(err, ShouldEqual, nil)
		})
	})

	conf_test = `
	kasdjasd = {};
	backend = 8
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml should parse", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	backend = "graphite-out"
	moo_input_format = "moo"
	output_format = "graphite"
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml should parse to a fail on bad toml", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	backend = "graphite-out"
	input_format = "moo"
	output_format = "graphite"
	keep_keys = true
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml should parse to a fail on input_format", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	keep_keys = true
	times = ["5ii"]
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml should parse to a fail on flush time", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	input_format = "statsd"
	output_format = "graphite"
	keep_keys = true
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml should parse to a fail on No Backend", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	input_format = "statsd"
	output_format = "moo"
	options = [
            ["legacyNamespace", "true"],
            ["prefixGauge", "gauges"],
            ["prefixTimer", "timers"],
            ["prefixCounter", "counters"],
            ["globalPrefix", ""],
            ["globalSuffix", "stats"],
            ["percentThreshold", "0.75,0.90,0.95,0.99"]
    ]
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml should parse to a fail on output_format", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})
}
