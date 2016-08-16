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

package stats

import (
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"testing"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestStatsAtomic(t *testing.T) {
	// Only pass t into top-level Convey calls
	Convey("Given an AtomicInt", t, func() {
		atm := NewAtomic("tester")
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
		// need a random string as tests can be called a few times
		nm := RandString(10)
		ctr := NewStatCount(nm)
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
		Convey("initial should be StatsdNoop", func() {
			So(StatsdClient, ShouldNotEqual, nil)
		})
	})
}
