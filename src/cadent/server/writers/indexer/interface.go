/*
  Indexer Reader/Writer

  just Read/Write the metric key
*/

package indexer

/****************** Data writers *********************/
type Indexer interface {
	Config(map[string]interface{}) error

	Write(metric string) error // write a metric key

	// reader methods
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
}
