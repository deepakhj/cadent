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

type StatsdFormater struct{}

func (g *StatsdFormater) Type() string { return STATSD_FMT_NAME }
func (g *StatsdFormater) ToString(key string, val float64, tstamp int32, stats_type string, tags map[string]string) string {

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
