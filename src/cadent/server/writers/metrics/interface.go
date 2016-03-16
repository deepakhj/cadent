/*
   Writers/Readers of stats

   We are attempting to mimic the Graphite API json blobs throughout the process here
   such that we can hook this in directly to either graphite-api/web

   NOTE: this is not a full graphite DSL, just paths and metrics, we leave the fancy functions inside
   graphite-api/web to work their magic .. one day we'll impliment the full DSL, but until then ..

   Currently just implimenting /find /expand and /render (json only) for graphite-api
*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/writers/indexer"
	logging "gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("metrics")

/****************** Data readers *********************/

type Metrics interface {

	// need to able to set what our resolutions are for ease of resolution picking
	SetResolutions([][]int)

	Config(map[string]interface{}) error

	// need an Indexer 99% of the time to deal with render
	SetIndexer(indexer.Indexer) error

	// Writer
	Write(repr.StatRepr) error

	// Reader

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
	Render(path string, from string, to string) (WhisperRenderItem, error)
}
