/*
   Writers/Readers of stats

   We are attempting to mimic the Graphite API json blobs throughout the process here
   such that we can hook this in directly to either graphite-api/web

   NOTE: this is not a full graphite DSL, just paths and metrics, we leave the fancy functions inside
   graphite-api/web to work their magic .. one day we'll impliment the full DSL, but until then ..

   Currently just implimenting /find /expand and /render (json only) for graphite-api
*/

package readers

import (
	logging "gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("readers")

/****************** Data readers *********************/

type Reader interface {

	// need to able to set what our resolutions are for ease of resolution picking
	SetResolutions([][]int)

	Config(map[string]interface{}) error

	// /metrics/find/?query=stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.*
	/*
			[
		{
		text: "accumulator",
		expandable: 1,
		leaf: 0,
		id: "stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.accumulator",
		allowChildren: 1
		}
		]
	*/
	Find(metric string) (MetricFindItems, error)

	// /metric/expand?query=stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.*
	/*
			{
		results: [
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.send",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.sent-bytes",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.valid-lines",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.valid-lines-sent-to-workers"
		]
		}
	*/
	Expand(metric string) (MetricExpandItem, error)

	// /render?target=XXXX&from=-24h&to=now
	/*
			{
		target: "scaleToSeconds(stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.writer.cassandra.noncached-writes-path,1)",
		datapoints: [
		[
		0,
		1456087430
		],...
		]}
	*/
	Render(path string, from string, to string) (RenderItems, error)
}
