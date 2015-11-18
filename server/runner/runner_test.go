package runner

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"regexp"
)


func TestGraphiteRunner(t *testing.T) {

	conf := make(map[string]interface{})
	conf["key_index"] = 0

	good_line := "moo.goo.org 123 123123"
	Convey("Graphite Runner should parse lines nicely", t, func() {

		gr, _ := NewGraphiteRunner(conf)
		key, orig_line, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "graphite")
		So(key, ShouldEqual, "moo.goo.org")
		So(orig_line, ShouldEqual, good_line)
	})


	conf["key_index"] = 10
	Convey("Graphite Runner should not parser this with a bad key index", t, func() {

		gr, _ := NewGraphiteRunner(conf)
		key, _, err := gr.ProcessLine(good_line)
		So(key, ShouldEqual, "")
		So(err, ShouldNotEqual, nil)
	})

}

func TestStatsdRunner(t *testing.T) {

	conf := make(map[string]interface{})

	good_line := "moo.goo.org:123|0.1"
	bad_line := "moo.goo.orgII123|0.1"

	Convey("Statsd Runner should parse lines nicely", t, func() {

		gr, _ := NewStatsdRunner(conf)
		key, orig_line, _ := gr.ProcessLine(good_line)
		So(key, ShouldEqual, "moo.goo.org")
		So(gr.Name(), ShouldEqual, "statsd")
		So(orig_line, ShouldEqual, good_line)
	})


	Convey("Statsd Runner should not parser this", t, func() {

		gr, _ := NewStatsdRunner(conf)
		key, _, err := gr.ProcessLine(bad_line)
		So(key, ShouldEqual, "")
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

	Convey("REgex Runner should parse lines nicely", t, func() {

		gr, _ := NewRegExRunner(conf)
		key, orig_line, _ := gr.ProcessLine(good_line)
		So(gr.Name(), ShouldEqual, "regex")
		So(key, ShouldEqual, "web-7-frontend-lb-prod")
		So(orig_line, ShouldEqual, good_line)
	})


	Convey("REgex Runner should not parser this", t, func() {

		gr, _ := NewRegExRunner(conf)
		key, _, err := gr.ProcessLine(bad_line_2)
		So(key, ShouldEqual, "")
		So(err, ShouldNotEqual, nil)
	})

	conf["regexpNames"] = []string{"Timestamp", "Logger"}

	Convey("REgex Runner should not parser this", t, func() {

		gr, _ := NewRegExRunner(conf)
		key, _, err := gr.ProcessLine(bad_line)
		So(key, ShouldEqual, "")
		So(err, ShouldNotEqual, nil)
	})

}
