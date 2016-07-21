/*
Dump the line graphite expects to get
*/

package accumulator

import (
	"cadent/server/repr"
	"fmt"
	"time"
)

const GRAPHITE_FMT_NAME = "graphite_formater"

type GraphiteFormatter struct {
	acc AccumulatorItem
}

func (g *GraphiteFormatter) Init(items ...string) error {
	return nil
}

func (g *GraphiteFormatter) GetAccumulator() AccumulatorItem {
	return g.acc
}
func (g *GraphiteFormatter) SetAccumulator(acc AccumulatorItem) {
	g.acc = acc
}

func (g *GraphiteFormatter) Type() string { return GRAPHITE_FMT_NAME }
func (g *GraphiteFormatter) ToString(name repr.StatName, val float64, tstamp int32, stats_type string, tags []AccumulatorTags) string {
	if tstamp <= 0 {
		tstamp = int32(time.Now().Unix())
	}
	return fmt.Sprintf("%s %f %d", name.Key, val, tstamp)
}

func (g *GraphiteFormatter) Write(buf BinaryWriter, name repr.StatName, val float64, tstamp int32, stats_type string, tags []AccumulatorTags) {
	if tstamp <= 0 {
		tstamp = int32(time.Now().Unix())
	}
	fmt.Fprint(buf, "%s %f %d", name.Key, val, tstamp)
}
