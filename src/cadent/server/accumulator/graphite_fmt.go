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
Dump the line graphite expects to get
*/

package accumulator

import (
	"cadent/server/repr"
	"cadent/server/utils"
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
	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)
	g.Write(buf, name, val, tstamp, stats_type, tags)
	return buf.String()
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
