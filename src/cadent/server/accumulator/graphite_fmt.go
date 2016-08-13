/*
Dump the line graphite expects to get
*/

package accumulator

import (
	"cadent/server/repr"
	"fmt"
	"io"
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
func (g *GraphiteFormatter) ToString(name *repr.StatName, val float64, tstamp int32, stats_type string, tags repr.SortingTags) string {
	if tstamp <= 0 {
		tstamp = int32(time.Now().Unix())
	}
	name.MergeMetric2Tags(tags)
	t_str := fmt.Sprintf("%s %f %d", name.Key, val, tstamp)
	if !name.Tags.IsEmpty() {
		t_str += repr.SPACE_SEPARATOR + name.MetaTags.ToStringSep(repr.EQUAL_SEPARATOR, repr.SPACE_SEPARATOR)
		if !name.MetaTags.IsEmpty() {
			t_str += repr.SPACE_SEPARATOR
		}
	}
	if !name.MetaTags.IsEmpty() {
		t_str += repr.SPACE_SEPARATOR + name.MetaTags.ToStringSep(repr.EQUAL_SEPARATOR, repr.SPACE_SEPARATOR)
	}
	return t_str
}

func (g *GraphiteFormatter) Write(buf io.Writer, name *repr.StatName, val float64, tstamp int32, stats_type string, tags repr.SortingTags) {
	if tstamp <= 0 {
		tstamp = int32(time.Now().Unix())
	}
	// merge things
	name.MergeMetric2Tags(tags)

	fmt.Fprintf(buf, "%s %f %d", name.Key, val, tstamp)
	if !name.Tags.IsEmpty() {
		buf.Write(repr.SPACE_SEPARATOR_BYTE)
		name.Tags.WriteBytes(buf, repr.EQUAL_SEPARATOR_BYTE, repr.SPACE_SEPARATOR_BYTE)
		if !name.MetaTags.IsEmpty() {
			buf.Write(repr.SPACE_SEPARATOR_BYTE)
		}
	}
	if !name.MetaTags.IsEmpty() {
		buf.Write(repr.SPACE_SEPARATOR_BYTE)
		name.MetaTags.WriteBytes(buf, repr.EQUAL_SEPARATOR_BYTE, repr.SPACE_SEPARATOR_BYTE)
	}
	buf.Write(repr.NEWLINE_SEPARATOR_BYTES)
}
