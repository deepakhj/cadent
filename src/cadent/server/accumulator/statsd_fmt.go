/*
Copyright 2016 Under Armour, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
	Dump the line statsd expects to get
*/

package accumulator

import (
	"cadent/server/repr"
	"cadent/server/utils"
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
func (g *StatsdFormatter) ToString(name *repr.StatName, val float64, tstamp int32, stats_type string, tags repr.SortingTags) string {

	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)
	g.Write(buf, name, val, tstamp, stats_type, tags)
	return buf.String()
}

//tags not suppported
func (g *StatsdFormatter) Write(buf io.Writer, name *repr.StatName, val float64, tstamp int32, stats_type string, tags repr.SortingTags) {

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
	name.MergeMetric2Tags(tags)

	//remove the "stats.(counters|gauges|timers)" prefix .. don't want weird recursion
	key := strings.Replace(
		strings.Replace(
			strings.Replace(name.Key, "stats.counters.", "", 1),
			"stats.gauges.", "", 1),
		"stats.timers.", "", 1)
	fmt.Fprintf(buf, "%s:%v|%s", key, val, stats_type)

	// tags are in the DataGram sort of format
	// http://docs.datadoghq.com/guides/dogstatsd/#datagram-format
	e_tags := name.Tags.IsEmpty()
	if !e_tags {
		buf.Write(repr.DATAGRAM_SEPARATOR_BYTES)
		name.Tags.WriteBytes(buf, repr.COLON_SEPARATOR_BYTE, repr.COMMA_SEPARATOR_BYTE)
	}
	if !name.MetaTags.IsEmpty() {
		if !e_tags {
			buf.Write(repr.COMMA_SEPARATOR_BYTE)
		} else {
			buf.Write(repr.DATAGRAM_SEPARATOR_BYTES)
		}
		name.MetaTags.WriteBytes(buf, repr.COLON_SEPARATOR_BYTE, repr.COMMA_SEPARATOR_BYTE)

	}
	buf.Write(repr.NEWLINE_SEPARATOR_BYTES)
}
