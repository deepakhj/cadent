package prereg

import (
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"os"
	"testing"
)

func TestPreRegConfig(t *testing.T) {

	//some tester strings
	t_config := `
	[graphite-regex-map]
	listen_server="graphite-proxy"
	default_backend="graphite-proxy"

	# another backend
	[[graphite-regex-map.map]]
	#what we're regexed from the input
	regex="""^servers.main.*"""
	backend="graphite-proxy"

	# anything that starts with the prefix (in lue of a more expesive regex)
	[[graphite-regex-map.map]]
	prefix="""servers.main-"""
	backend="graphite-statsd"

	[[graphite-regex-map.map]]
	substring=".servers."
	reject=true  # special "reject me" type

	[statsd-regex-map]
	default_backend="statsd-proxy"
	listen_server="statsd-proxy"

	# another backend
	[[statsd-regex-map.map]]
	regex="""^stats.timers..*"""   #what we're regexed from the input
	backend="statsd-proxy"

	# anything that has this sub string
	[[statsd-regex-map.map]]
	substring=".here."
	backend="statsd-servers"

	[[statsd-regex-map.map]]
	regex="""^statdtest.house..*"""
	reject=true  # special "reject me" type`

	// Only pass t into top-level Convey calls
	Convey("Given a config string", t, func() {

		prm, _ := ParseConfigString(t_config)

		Convey("We should have 2 main sections", func() {
			So(len(prm), ShouldEqual, 2)
		})
		Convey("Each with 3 filters", func() {
			for _, pr := range prm {
				So(len(pr.FilterList), ShouldEqual, 3)
			}
		})
		pr := prm["graphite-regex-map"]

		Convey("And Filter should be a of the right type", func() {
			So(pr.FilterList[0].Type(), ShouldEqual, "regex")
			So(pr.FilterList[1].Type(), ShouldEqual, "prefix")
			So(pr.FilterList[2].Type(), ShouldEqual, "substring")
		})
		Convey("The thrid filter should be a rejection", func() {
			So(pr.FilterList[2].Rejecting(), ShouldEqual, true)
		})
	})
	// Only pass t into top-level Convey calls
	Convey("Given a config file", t, func() {
		fname := "/tmp/__tonfig.toml"
		ioutil.WriteFile(fname, []byte(t_config), 0644)
		defer os.Remove(fname)
		prm, _ := ParseConfigFile(fname)

		Convey("We should have 2 main sections", func() {
			So(len(prm), ShouldEqual, 2)
		})

	})

}
