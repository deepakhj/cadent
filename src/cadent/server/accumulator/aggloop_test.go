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
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestAccumualtorAggLoop(t *testing.T) {
	// Only pass t into top-level Convey calls

	//profiler
	//go http.ListenAndServe(":6065", nil)

	Convey("AggLoop Tests", t, func() {
		Convey("Aggloop should fail", func() {
			conf_str := `
	backend = "BLACKHOLE"
	input_format = "graphite"
	output_format = "graphite"
	times = ["1s:5s"]
	[writer.metrics]
	driver = "moo"
	dsn = "/tmp/tt"
	[writer.indexer]
	driver = "noop"
	`
			_, err := ParseConfigString(conf_str)
			Convey("Error should not be nil", func() {
				So(err, ShouldNotEqual, nil)
			})

		})

		conf_str := `
	backend = "BLACKHOLE"
	input_format = "graphite"
	output_format = "graphite"
	times = ["3s:5s"]
	[writer.metrics]
	driver = "file"
	dsn = "/tmp/tt"
	[writer.indexer]
	driver = "noop"

	`

		cf, err := DecodeConfigString(conf_str)
		Convey("Aggloop should have been configed properly", func() {

			Convey("Error should not be nil", func() {
				So(err, ShouldEqual, nil)
			})
		})

		Convey("Aggloop created properly", func() {
			cf.ParseDurations()

			agg, err := NewAggregateLoop(cf.durations, cf.ttls, "tester")
			Convey("Aggregator should init", func() {
				So(agg, ShouldNotEqual, nil)
			})

			err = agg.SetWriter(cf.Writer)
			Convey("Aggregator writer should init", func() {
				So(err, ShouldEqual, nil)
			})
			agg.Stop()
		})
	})
}
