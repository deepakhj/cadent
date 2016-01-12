/*
	Dump the line statsd expects to get
*/

package accumulator

import (
	"fmt"
	"strings"
)

/****************** RUNNERS *********************/
const STATSD_FMT_NAME = "statsd_formater"

type StatsdFormatter struct{}

func (g *StatsdFormatter) Init(items ...string) error {
	return nil
}
func (g *StatsdFormatter) Type() string { return STATSD_FMT_NAME }
func (g *StatsdFormatter) ToString(key string, val float64, tstamp int32, stats_type string, tags []AccumulatorTags) string {

	if stats_type == "" {
		stats_type = "c"
	}

	//remove the "stats.(counters|gauges|timers)" prefix .. don't want weird recursion
	key = strings.Replace(
		strings.Replace(
			strings.Replace(key, "stats.counters.", "", 1),
			"stats.gauges.", "", 1),
		"stats.timers.", "", 1)

	return fmt.Sprintf("%s:%f|%s", key, val, stats_type)
}
