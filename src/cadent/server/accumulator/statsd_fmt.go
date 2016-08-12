/*
	Dump the line statsd expects to get
*/

package accumulator

import (
	"cadent/server/repr"
	"fmt"
	"io"
	"strings"
)

/****************** RUNNERS *********************/
const STATSD_FMT_NAME = "statsd_formater"

type StatsdFormatter struct {
	acc AccumulatorItem
}

func (g *StatsdFormatter) Init(items ...string) error {
	return nil
}

func (g *StatsdFormatter) GetAccumulator() AccumulatorItem {
	return g.acc
}
func (g *StatsdFormatter) SetAccumulator(acc AccumulatorItem) {
	g.acc = acc
}

func (g *StatsdFormatter) Type() string { return STATSD_FMT_NAME }
func (g *StatsdFormatter) ToString(name *repr.StatName, val float64, tstamp int32, stats_type string, tags []AccumulatorTags) string {

	switch {
	case stats_type == "g" || stats_type == "gauge":
		stats_type = "g"
		break
	case stats_type == "ms" || stats_type == "rate":
		stats_type = "ms"
		break
	default:
		stats_type = "c"
	}

	//remove the "stats.(counters|gauges|timers)" prefix .. don't want weird recursion
	key := strings.Replace(
		strings.Replace(
			strings.Replace(name.Key, "stats.counters.", "", 1),
			"stats.gauges.", "", 1),
		"stats.timers.", "", 1)

	return fmt.Sprintf("%s:%f|%s", key, val, stats_type)
}

func (g *StatsdFormatter) Write(buf io.Writer, name *repr.StatName, val float64, tstamp int32, stats_type string, tags []AccumulatorTags) {

	switch {
	case stats_type == "g" || stats_type == "gauge":
		stats_type = "g"
		break
	case stats_type == "ms" || stats_type == "rate":
		stats_type = "ms"
		break
	default:
		stats_type = "c"
	}

	//remove the "stats.(counters|gauges|timers)" prefix .. don't want weird recursion
	key := strings.Replace(
		strings.Replace(
			strings.Replace(name.Key, "stats.counters.", "", 1),
			"stats.gauges.", "", 1),
		"stats.timers.", "", 1)

	fmt.Fprintf(buf, "%s:%f|%s\n", key, val, stats_type)
}
