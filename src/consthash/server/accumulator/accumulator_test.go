package accumulator

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestAccumualtorAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls

	conf_test := `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	flush_time = "5s"
	[[tags]]
	key="foo"
	value="bar"
	[[tags]]
	key="loo"
	value="moo"
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
		Convey("Flush time should be 5 seconds", func() {
			So(acc_c.FlushTime, ShouldEqual, time.Duration(5*time.Second))
		})
		Convey("Should have 2 tags", func() {
			So(len(acc_c.Accumulate.Tags()), ShouldEqual, 2)
		})

	})

	conf_test = `
	backend = "graphite-out"
	moo_input_format = "moo"
	output_format = "graphite"
	flush_time = "5s"
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
	flush_time = "5s"
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml should parse to a fail on input_format", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	conf_test = `
	input_format = "statsd"
	output_format = "moo"
	flush_time = "5s"
	`
	acc_c, err = ParseConfigString(conf_test)
	Convey("Config toml should parse to a fail on output_format", t, func() {
		Convey("Error should be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
	})

	fail_acc, err := NewAccumlator("monkey", "graphite")
	Convey("Bad formatter name `monkey` should faile ", t, func() {
		Convey("Error should not be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
		Convey("Error acc should be nil", func() {
			So(fail_acc, ShouldEqual, nil)
		})
	})

	fail_acc, err = NewAccumlator("graphite", "monkey")
	Convey("Bad accumulator name `monkey` should faile ", t, func() {
		Convey("Error should not be nil", func() {
			So(err, ShouldNotEqual, nil)
		})
		Convey("Error acc should be nil", func() {
			So(fail_acc, ShouldEqual, nil)
		})
	})

	grph_acc, err := NewAccumlator("graphite", "graphite")

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
	statsd_acc, err := NewAccumlator("statsd", "graphite")
	Convey("Graphite to graphite -> statsd accumulator should be ok", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})
		Convey("Error acc should not be nil", func() {
			So(statsd_acc, ShouldNotEqual, nil)
		})
	})
	statsd_acc, err = NewAccumlator("statsd", "statsd")
	Convey("Graphite to statsd -> statsd accumulator should be ok", t, func() {

		Convey("Error should be nil", func() {
			So(err, ShouldEqual, nil)
		})
		Convey("Error acc should not be nil", func() {
			So(statsd_acc, ShouldNotEqual, nil)
		})
	})

	tickC := make(chan string)
	statsd_acc.FlushTime = time.Duration(time.Second)
	statsd_acc.OutputQueue = tickC
	tt := time.NewTimer(time.Duration(2 * time.Second))
	go statsd_acc.Start()
	go statsd_acc.Start()
	Convey("statsd accumluator flush timer", t, func() {

		err = statsd_acc.ProcessLine("moo.goo.poo:12|c")
		err = statsd_acc.ProcessLine("moo.goo.poo:0.1|ms|@0.2")
		err = statsd_acc.ProcessLine("moo.goo.goo:24|c")
		err = statsd_acc.ProcessLine("moo.goo.goo:24|c")
		err = statsd_acc.ProcessLine("moo.goo.loo:36|c")
		err = statsd_acc.ProcessLine("moo.goo.loo||||36|c")
		outs := []string{}
		t_f := func() {
			for {
				select {
				case <-tt.C:
					statsd_acc.Stop()
					return

				case l := <-tickC:
					outs = append(outs, l)
					t.Logf("FlushLine %s", l)
				}
			}
		}
		t_f()
		Convey("should have 9 flushed lines", func() {
			So(len(outs), ShouldEqual, 9)
		})
	})

}
