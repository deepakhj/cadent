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

package splitter

import (
	. "github.com/smartystreets/goconvey/convey"
	"regexp"
	"testing"
)

func TestRegistryRunner(t *testing.T) {

	conf := make(map[string]interface{})
	regg := regexp.MustCompile(`^(<\d+>)?(?P<Timestamp>[A-Z][a-z]+\s+\d+\s\d+:\d+:\d+) (?P<Key>\S+) (?P<Logger>\S+):(.*)`)
	conf["regexp"] = regg
	conf["regexpNames"] = regg.SubexpNames()
	Convey("Should get all the items in the registry", t, func() {

		gr, _ := NewSplitterItem("graphite", conf)
		So(gr.Name(), ShouldEqual, "graphite")
		st, _ := NewSplitterItem("statsd", conf)
		So(st.Name(), ShouldEqual, "statsd")
		re, _ := NewSplitterItem("regex", conf)
		So(re.Name(), ShouldEqual, "regex")
		ca, _ := NewSplitterItem("carbon2", conf)
		So(ca.Name(), ShouldEqual, "carbon2")
		_, err := NewSplitterItem("nothere", conf)
		So(err, ShouldNotEqual, nil)

	})
}

func TestGraphiteRunner(t *testing.T) {

	conf := make(map[string]interface{})
	conf["key_index"] = 0

	good_line := "moo.goo.org 123 1465866540"
	Convey("Graphite Runner should parse lines nicely", t, func() {

		gr, _ := NewGraphiteSplitter(conf)
		spl, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "graphite")
		So(spl.Key(), ShouldEqual, "moo.goo.org")
		So(spl.Line(), ShouldEqual, good_line)
		So(spl.OriginName(), ShouldEqual, "")
		So(spl.Phase(), ShouldEqual, Parsed)
		So(spl.IsValid(), ShouldEqual, true)
		So(spl.HasTime(), ShouldEqual, true)
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)

		So(spl.Fields(), ShouldResemble, []string{
			"moo.goo.org",
			"123",
			"1465866540",
		})

		spl.SetPhase(AccumulatedParsed)
		So(spl.Phase(), ShouldEqual, AccumulatedParsed)
		spl.SetOrigin(TCP)
		So(spl.Origin(), ShouldEqual, TCP)
		spl.SetOriginName("moo")
		So(spl.OriginName(), ShouldEqual, "moo")
	})

	conf["key_index"] = 10
	Convey("Graphite Runner should not parser this with a bad key index", t, func() {

		gr, _ := NewGraphiteSplitter(conf)
		spl, err := gr.ProcessLine(good_line)
		t.Logf("'%s', %v", good_line, spl)
		So(spl, ShouldEqual, nil)
		So(err, ShouldNotEqual, nil)
	})

}

func TestStatsdRunner(t *testing.T) {

	conf := make(map[string]interface{})

	good_line := "moo.goo.org:123|ms|@0.1"
	bad_line := "moo.goo.orgII123|0.1"

	Convey("Statsd Runner should parse lines nicely", t, func() {

		gr, _ := NewStatsdSplitter(conf)
		si, _ := gr.ProcessLine(good_line)
		So(si.Key(), ShouldEqual, "moo.goo.org")
		So(gr.Name(), ShouldEqual, "statsd")
		So(si.OriginName(), ShouldEqual, "")
		So(si.IsValid(), ShouldEqual, true)
		So(si.Line(), ShouldEqual, good_line)
		So(si.Fields(), ShouldResemble, []string{
			"moo.goo.org",
			"123|ms|@0.1",
		})
		So(si.Phase(), ShouldEqual, Parsed)
		si.SetPhase(AccumulatedParsed)
		So(si.Phase(), ShouldEqual, AccumulatedParsed)
		si.SetOrigin(UDP)
		So(si.Origin(), ShouldEqual, UDP)
		si.SetOriginName("moo")
		So(si.OriginName(), ShouldEqual, "moo")

	})

	Convey("Statsd Runner should not parser this", t, func() {

		gr, _ := NewStatsdSplitter(conf)
		si, err := gr.ProcessLine(bad_line)
		So(si, ShouldEqual, nil)
		So(err, ShouldNotEqual, nil)
	})

}

func TestRegexRunner(t *testing.T) {

	conf := make(map[string]interface{})

	good_line := "Nov 18 01:22:36 web-7-frontend-lb-prod mfp-nginx: 190.172.234.165 - 10.31.133.179 momo"
	bad_line := "abc123"
	bad_line_2 := "Nov 18 01:22:36 web-7-frontend-lb-prod"

	reg := `(<\d+>)?(?P<Timestamp>[A-Z][a-z]+\s+\d+\s\d+:\d+:\d+) (?P<Key>\S+) (?P<Logger>\S+):(.*)`
	regger := regexp.MustCompile(reg)

	conf["regexp"] = regger
	conf["regexpNames"] = []string{"", "Timestamp", "Key", "Logger"}

	spl_string := []string{
		good_line,
		"",
		"Nov 18 01:22:36",
		"web-7-frontend-lb-prod",
		"mfp-nginx",
		" 190.172.234.165 - 10.31.133.179 momo",
	}

	Convey("REgex Runner should parse lines nicely", t, func() {
		gr, _ := NewRegExSplitter(conf)
		ri, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "regex")
		So(ri.Key(), ShouldEqual, "web-7-frontend-lb-prod")
		So(ri.Line(), ShouldEqual, good_line)
		So(ri.Phase(), ShouldEqual, Parsed)
		So(ri.Origin(), ShouldEqual, Other)
		So(ri.OriginName(), ShouldEqual, "")
		So(ri.Fields(), ShouldResemble, spl_string)
		So(ri.IsValid(), ShouldEqual, true)
		ri.SetPhase(AccumulatedParsed)
		So(ri.Phase(), ShouldEqual, AccumulatedParsed)
		ri.SetOrigin(TCP)
		So(ri.Origin(), ShouldEqual, TCP)
		ri.SetOriginName("moo")
		So(ri.OriginName(), ShouldEqual, "moo")
	})

	Convey("REgex Runner should not parser this", t, func() {

		gr, _ := NewRegExSplitter(conf)
		ri, err := gr.ProcessLine(bad_line_2)
		So(ri, ShouldEqual, nil)
		So(err, ShouldNotEqual, nil)
	})

	conf["regexpNames"] = []string{"Timestamp", "Logger"}

	Convey("REgex Runner should not parser this as well", t, func() {

		gr, _ := NewRegExSplitter(conf)
		ri, err := gr.ProcessLine(bad_line)
		So(ri, ShouldEqual, nil)
		So(err, ShouldNotEqual, nil)
	})

}

func TestCarbonTwoRunner(t *testing.T) {

	conf := make(map[string]interface{})

	good_line := "moo.goo.org  moo=goo house=loop 345345 1465866540"
	good_line_2 := "moo.goo.org  345345 1465866540"
	good_line_3 := "host=me type=monkey stat=last mtype=counter 345345 1465866540"
	bad_line := "moo.goo.org 3456n -890"
	Convey("CarbonTwo Runner should parse lines nicely", t, func() {

		gr, _ := NewCarbonTwoSplitter(conf)
		spl, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "carbon2")
		So(spl.Key(), ShouldEqual, "moo.goo.org")
		So(spl.Line(), ShouldEqual, good_line)
		So(spl.OriginName(), ShouldEqual, "")
		So(spl.Phase(), ShouldEqual, Parsed)
		So(spl.IsValid(), ShouldEqual, true)
		So(spl.HasTime(), ShouldEqual, true)
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)

		So(spl.Fields(), ShouldResemble, []string{
			"moo=goo",
			"house=loop",
			"345345",
			"1465866540",
		})

		spl.SetPhase(AccumulatedParsed)
		So(spl.Phase(), ShouldEqual, AccumulatedParsed)
		spl.SetOrigin(TCP)
		So(spl.Origin(), ShouldEqual, TCP)
		spl.SetOriginName("moo")
		So(spl.OriginName(), ShouldEqual, "moo")

		So(spl.Tags(), ShouldResemble, [][]string{
			{"moo", "goo"},
			{"house", "loop"},
		})

	})

	Convey("CarbonTwo Runner should not parser this with a bad key index", t, func() {

		gr, _ := NewCarbonTwoSplitter(conf)
		spl, err := gr.ProcessLine(bad_line)
		t.Logf("'%s', %v", good_line, spl)
		So(spl, ShouldEqual, nil)
		So(err, ShouldNotEqual, nil)
	})

	Convey("CarbonTwo Runner not need meta tags", t, func() {

		gr, _ := NewCarbonTwoSplitter(conf)
		spl, err := gr.ProcessLine(good_line_2)
		So(spl, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		So(spl.Key(), ShouldEqual, "moo.goo.org")
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)
		So(spl.Tags(), ShouldResemble, [][]string{})

	})

	Convey("CarbonTwo Runner with opener tags", t, func() {

		gr, _ := NewCarbonTwoSplitter(conf)
		spl, err := gr.ProcessLine(good_line_3)
		So(spl, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		// host=me type=monkey stat=last mtype=counter
		So(spl.Key(), ShouldEqual, "host=me type=monkey stat=last mtype=counter")
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)
		So(spl.Tags(), ShouldResemble, [][]string{})

	})

}

func TestUnkRunner(t *testing.T) {

	conf := make(map[string]interface{})

	Convey("UNknonwn Runner should parse lines nicely", t, func() {
		gr, _ := NewUnknownSplitter(conf)
		ri, _ := gr.ProcessLine("")
		So(gr.Name(), ShouldEqual, "unknown")
		So(ri.Key(), ShouldEqual, "")
		So(ri.Line(), ShouldEqual, "")
		So(ri.Phase(), ShouldEqual, Parsed)
		ri.SetPhase(AccumulatedParsed)
		So(ri.Fields(), ShouldResemble, []string{})
		So(ri.IsValid(), ShouldEqual, false)
	})

	Convey("Unkown should return the blank singleton", t, func() {

		gr := BlankSplitterItem()
		So(gr.Line(), ShouldEqual, "")
		So(gr, ShouldNotEqual, nil)
	})

}
