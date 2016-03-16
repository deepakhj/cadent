/*
   Writers/Readers of stats
*/

package writers

import (
	"cadent/server/repr"
)

/****************** Data writers *********************/
type Writer interface {
	Config(map[string]interface{}) error
	Write(repr.StatRepr) error
}

type Reader interface {
	Config(map[string]interface{}) error

	// impliments the graphite /metrics/find/?query=stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.*
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
	Find(metric string)
}
