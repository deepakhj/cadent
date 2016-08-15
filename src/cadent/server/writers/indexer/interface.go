/*
  Indexer Reader/Writer

  just Read/Write the StatName object

  StatNames have 4 writable indexes: key, uniqueId, tags

  The Unique ID is basically a hash of the key:sortedByName(tags)


*/

package indexer

import "cadent/server/repr"

/****************** Data writers *********************/
type Indexer interface {
	Config(map[string]interface{}) error

	// some identifier mostly used for logs
	Name() string

	Write(metric repr.StatName) error // write a metric key

	// reader methods this is an "extra" graphite based entity
	// /metrics/find/?query=stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.*
	/*
		[
			{
			text: "accumulator",
			expandable: 1,
			leaf: 0,
			key: "stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.accumulator"
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

	Start()

	// stop all processing
	Stop()

	// remove an item from the index
	Delete(name *repr.StatName) error
}
