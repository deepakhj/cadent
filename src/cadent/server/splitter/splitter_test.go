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
		op, _ := NewSplitterItem("opentsdb", conf)
		So(op.Name(), ShouldEqual, "opentsdb")
		_, err := NewSplitterItem("nothere", conf)
		So(err, ShouldNotEqual, nil)

	})
}

func TestGraphiteRunner(t *testing.T) {

	conf := make(map[string]interface{})
	conf["key_index"] = 0

	good_line := []byte("moo.goo.org 123 1465866540")
	Convey("Graphite Runner should parse lines nicely", t, func() {

		gr, _ := NewGraphiteSplitter(conf)
		spl, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "graphite")
		So(string(spl.Key()), ShouldEqual, "moo.goo.org")
		So(string(spl.Line()), ShouldEqual, string(good_line))
		So(spl.OriginName(), ShouldEqual, "")
		So(spl.Phase(), ShouldEqual, Parsed)
		So(spl.IsValid(), ShouldEqual, true)
		So(spl.HasTime(), ShouldEqual, true)
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)

		So(spl.Fields(), ShouldResemble, [][]byte{
			[]byte("moo.goo.org"),
			[]byte("123"),
			[]byte("1465866540"),
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


func TestOpenTSDBRunner(t *testing.T) {

	conf := make(map[string]interface{})
	conf["key_index"] = 0

	good_line := []byte("put moo.goo.org 123 1465866540")
	bad_line := []byte("moo.goo.org 123 1465866540")
	Convey("Opentsdb Runner should parse lines nicely", t, func() {

		gr, _ := NewOpenTSDBSplitter(conf)
		spl, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "opentsdb")
		So(string(spl.Key()), ShouldEqual, "moo.goo.org")
		So(string(spl.Line()), ShouldEqual, string("moo.goo.org 123 1465866540"))
		So(spl.OriginName(), ShouldEqual, "")
		So(spl.Phase(), ShouldEqual, Parsed)
		So(spl.IsValid(), ShouldEqual, true)
		So(spl.HasTime(), ShouldEqual, true)
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)

		So(spl.Fields(), ShouldResemble, [][]byte{
			[]byte("moo.goo.org"),
			[]byte("123"),
			[]byte("1465866540"),
		})

		spl.SetPhase(AccumulatedParsed)
		So(spl.Phase(), ShouldEqual, AccumulatedParsed)
		spl.SetOrigin(TCP)
		So(spl.Origin(), ShouldEqual, TCP)
		spl.SetOriginName("moo")
		So(spl.OriginName(), ShouldEqual, "moo")
	})

	Convey("OpenTSDB Runner should not parser this with a bad key index", t, func() {

		gr, _ := NewOpenTSDBSplitter(conf)
		spl, err := gr.ProcessLine(bad_line)
		t.Logf("'%s', %v", bad_line, spl)
		So(spl, ShouldEqual, nil)
		So(err, ShouldNotEqual, nil)
	})

}

func TestStatsdRunner(t *testing.T) {

	conf := make(map[string]interface{})

	good_line := []byte("moo.goo.org:123|ms|@0.1")
	bad_line := []byte("moo.goo.orgII123|0.1")

	Convey("Statsd Runner should parse lines nicely", t, func() {

		gr, _ := NewStatsdSplitter(conf)
		si, _ := gr.ProcessLine(good_line)
		So(string(si.Key()), ShouldEqual, "moo.goo.org")
		So(gr.Name(), ShouldEqual, "statsd")
		So(si.OriginName(), ShouldEqual, "")
		So(si.IsValid(), ShouldEqual, true)
		So(string(si.Line()), ShouldEqual, string(good_line))
		So(si.Fields(), ShouldResemble, [][]byte{
			[]byte("moo.goo.org"),
			[]byte("123|ms|@0.1"),
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

	good_line := []byte("Nov 18 01:22:36 web-7-frontend-lb-prod mfp-nginx: 190.172.234.165 - 10.31.133.179 momo")
	bad_line := []byte("abc123")
	bad_line_2 := []byte("Nov 18 01:22:36 web-7-frontend-lb-prod")

	reg := `(<\d+>)?(?P<Timestamp>[A-Z][a-z]+\s+\d+\s\d+:\d+:\d+) (?P<Key>\S+) (?P<Logger>\S+):(.*)`
	regger := regexp.MustCompile(reg)

	conf["regexp"] = regger
	conf["regexpNames"] = []string{"", "Timestamp", "Key", "Logger"}

	spl_string := [][]byte{
		good_line,
		[]byte(""),
		[]byte("Nov 18 01:22:36"),
		[]byte("web-7-frontend-lb-prod"),
		[]byte("mfp-nginx"),
		[]byte(" 190.172.234.165 - 10.31.133.179 momo"),
	}

	Convey("REgex Runner should parse lines nicely", t, func() {
		gr, _ := NewRegExSplitter(conf)
		ri, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "regex")
		So(string(ri.Key()), ShouldEqual, string("web-7-frontend-lb-prod"))
		So(string(ri.Line()), ShouldEqual, string(good_line))
		So(ri.Phase(), ShouldEqual, Parsed)
		So(ri.Origin(), ShouldEqual, Other)
		So(ri.OriginName(), ShouldEqual, "")
		So(len(ri.Fields()), ShouldEqual, len(spl_string))

		for i, b := range spl_string {
			So(string(ri.Fields()[i]), ShouldEqual, string(b))
		}
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

	good_line := []byte("moo.goo.org  moo=goo house=loop 345345 1465866540")
	good_line_2 := []byte("moo.goo.org  345345 1465866540")
	good_line_3 := []byte("host=me type=monkey stat=last mtype=counter 345345 1465866540")
	bad_line := []byte("moo.goo.org 3456n -890")
	Convey("CarbonTwo Runner should parse lines nicely", t, func() {

		gr, _ := NewCarbonTwoSplitter(conf)
		spl, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "carbon2")
		So(string(spl.Key()), ShouldEqual, "moo.goo.org")
		So(string(spl.Line()), ShouldEqual, string(good_line))
		So(spl.OriginName(), ShouldEqual, "")
		So(spl.Phase(), ShouldEqual, Parsed)
		So(spl.IsValid(), ShouldEqual, true)
		So(spl.HasTime(), ShouldEqual, true)
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)

		shouldbe := []string{
			"moo=goo",
			"house=loop",
			"345345",
			"1465866540",
		}
		So(len(spl.Fields()), ShouldEqual, len(shouldbe))
		for i, b := range shouldbe {
			So(string(spl.Fields()[i]), ShouldEqual, b)
		}

		spl.SetPhase(AccumulatedParsed)
		So(spl.Phase(), ShouldEqual, AccumulatedParsed)
		spl.SetOrigin(TCP)
		So(spl.Origin(), ShouldEqual, TCP)
		spl.SetOriginName("moo")
		So(spl.OriginName(), ShouldEqual, "moo")

		shouldtags := [][]string{
			{"moo", "goo"},
			{"house", "loop"},
		}
		So(len(spl.Tags()), ShouldEqual, len(shouldtags))
		for i, b := range shouldtags {
			So(string(spl.Tags()[i][0]), ShouldEqual, b[0])
			So(string(spl.Tags()[i][1]), ShouldEqual, b[1])
		}
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
		So(string(spl.Key()), ShouldEqual, "moo.goo.org")
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)
		So(spl.Tags(), ShouldResemble, [][][]byte{})

	})

	Convey("CarbonTwo Runner with opener tags", t, func() {

		gr, _ := NewCarbonTwoSplitter(conf)
		spl, err := gr.ProcessLine(good_line_3)
		So(spl, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		// host=me type=monkey stat=last mtype=counter
		So(string(spl.Key()), ShouldEqual, "host=me type=monkey stat=last mtype=counter")
		So(spl.Timestamp().Unix(), ShouldEqual, 1465866540)
		So(spl.Tags(), ShouldResemble, [][][]byte{})

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
