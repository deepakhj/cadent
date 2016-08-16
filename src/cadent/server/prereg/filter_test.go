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

package prereg

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestPreRegFilters(t *testing.T) {

	//some tester strings
	statstring := "tester.i.am.a.stat 1234 1234"
	backend_str := []string{"graphite-prefix", "graphite-substring", "graphite-regex", "defaultbackend"}

	// Only pass t into top-level Convey calls
	Convey("Given a prereg Map", t, func() {
		var prere = PreReg{
			DefaultBackEnd: "defaultbackend",
			Name:           "matcher",
			ListenServer:   "graphite-test",
		}

		prere.FilterList = make([]FilterItem, 4)

		prefix := new(PrefixFilter)
		prefix.Prefix = "tester.i"
		prefix.IsReject = false
		prefix.SetBackend("graphite-prefix")

		prere.FilterList[0] = prefix

		substr := new(SubStringFilter)
		substr.SubString = ".am."
		substr.IsReject = true
		substr.SetBackend("graphite-substring")

		prere.FilterList[1] = substr

		reg := new(RegexFilter)
		reg.RegexString = ".i.*stat"
		reg.IsReject = false
		reg.SetBackend("graphite-regex")

		prere.FilterList[2] = reg

		nop := new(NoOpFilter)
		nop.SetBackend("defaultbackend")

		prere.FilterList[3] = nop

		maper := make(PreRegMap)
		maper["match_section"] = &prere

		Convey("When we init the filters", func() {
			for idx, f := range prere.FilterList {
				f.Init()
				So(f.Backend(), ShouldEqual, backend_str[idx])
			}
			So(prere.DefaultBackEnd, ShouldEqual, "defaultbackend")
		})

		// need to reinit as the damned COnvey hates loops
		for _, f := range prere.FilterList {
			f.Init()
		}

		Convey("When we attempt to match `"+statstring+"`", func() {
			Convey("Prefix should match ", func() {
				f := prere.FilterList[0]
				matches, rejected, err := f.Match(statstring)
				So(matches, ShouldEqual, true)
				So(rejected, ShouldEqual, false)
				So(err, ShouldEqual, nil)
				So(f.Backend(), ShouldEqual, "graphite-prefix")
			})

			Convey("Substring should match ", func() {
				f := prere.FilterList[1]
				matches, rejected, err := f.Match(statstring)
				So(matches, ShouldEqual, true)
				So(rejected, ShouldEqual, true)
				So(err, ShouldEqual, nil)
				So(f.Backend(), ShouldEqual, "graphite-substring")
			})
			Convey("Regex should match ", func() {
				f := prere.FilterList[2]
				matches, rejected, err := f.Match(statstring)
				So(matches, ShouldEqual, true)
				So(rejected, ShouldEqual, false)
				So(err, ShouldEqual, nil)
				So(f.Backend(), ShouldEqual, "graphite-regex")
			})
			Convey("Firstmatch filter of `"+statstring+"` should match ", func() {
				f, rejected, err := prere.FirstMatchFilter(statstring)
				So(f, ShouldEqual, prere.FilterList[0])
				So(rejected, ShouldEqual, false)
				So(err, ShouldEqual, nil)
				So(f.Backend(), ShouldEqual, "graphite-prefix")
			})

			Convey("Firstmatch filter of `123123` should match noop", func() {
				f, rejected, err := prere.FirstMatchFilter("123123")
				So(f, ShouldEqual, prere.FilterList[3])
				So(rejected, ShouldEqual, false)
				So(err, ShouldEqual, nil)
			})

			Convey("FirstMatchBackend filter of `123123` should be `defaultbackend` ", func() {
				bk, rejected, err := prere.FirstMatchBackend("123123")
				So(bk, ShouldEqual, "defaultbackend")
				So(rejected, ShouldEqual, false)
				So(err, ShouldEqual, nil)
			})

			Convey("FirstMatchBackend filter of `moo.am.goo` should be `defaultbackend` ", func() {
				bk, rejected, err := prere.FirstMatchBackend("moo.am.goo")
				So(bk, ShouldEqual, "graphite-substring")
				So(rejected, ShouldEqual, true)
				So(err, ShouldEqual, nil)
			})

			// full mapper
			Convey("MatchingFilters filter reg map of `123123` should match ", func() {
				f := maper.MatchingFilters("123123")
				So(len(f), ShouldEqual, 1)
			})

			Convey("MatchingBackends filter reg map of `123123` should match `defaultbackend`", func() {
				bk, rej, err := maper.FirstMatchBackends("123123")
				So(len(bk), ShouldEqual, 1)
				So(bk[0], ShouldEqual, "defaultbackend")
				So(len(rej), ShouldEqual, 1)
				So(len(err), ShouldEqual, 1)
			})

			Convey("MatchingBackends filter reg map of `"+statstring+"` should match `graphite-prefix`", func() {
				bk, rej, err := maper.FirstMatchBackends(statstring)
				So(len(bk), ShouldEqual, 1)
				So(bk[0], ShouldEqual, "graphite-prefix")
				So(len(rej), ShouldEqual, 1)
				So(len(err), ShouldEqual, 1)
			})

			Convey("logging", func() {
				maper.LogConfig()
				prere.FilterList[0].ToString()
				So(prere.FilterList[0].Type(), ShouldEqual, "prefix")

			})

		})

	})

}
