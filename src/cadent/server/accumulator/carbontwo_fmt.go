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
Dump the line carbontwo expects to get
*/

package accumulator

import (
	"cadent/server/repr"
	"cadent/server/utils"
	"fmt"
	"io"
	"sort"
	"time"
)

const CARBONTWO_FMT_NAME = "carbontwo_formater"

type CarbonTwoFormatter struct {
	acc AccumulatorItem
}

func (g *CarbonTwoFormatter) Init(items ...string) error {
	return nil
}

func (g *CarbonTwoFormatter) GetAccumulator() AccumulatorItem {
	return g.acc
}
func (g *CarbonTwoFormatter) SetAccumulator(acc AccumulatorItem) {
	g.acc = acc
}

func (g *CarbonTwoFormatter) Type() string { return CARBONTWO_FMT_NAME }

func (g *CarbonTwoFormatter) ToString(name *repr.StatName, val float64, tstamp int32, stats_type string, tags repr.SortingTags) string {
	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)
	g.Write(buf, name, val, tstamp, stats_type, tags)
	return buf.String()
}

// there is a bit of "trouble" w/ the carbon2 format from other formats.
// metrics2.0 requires mtype and unit .. we can "infer" an mtype, but not a unit
// also the "metric" is then the Key (if no tags)
// if no unit, we then do the "catch all" which is "jiff"
func (g *CarbonTwoFormatter) Write(buf io.Writer, name *repr.StatName, val float64, tstamp int32, stats_type string, tags repr.SortingTags) {
	if tstamp <= 0 {
		tstamp = int32(time.Now().Unix())
	}

	// merge things
	name.MergeMetric2Tags(tags)

	tags_empty := name.Tags.IsEmpty()
	mtype := name.Tags.Mtype()
	if mtype == "" {
		switch {
		case stats_type == "g" || stats_type == "gauge":
			mtype = "gauge"
			break
		case stats_type == "ms" || stats_type == "rate":
			mtype = "rate"
			break
		default:
			mtype = "count"
		}

		if !tags_empty {
			name.Tags.Set("mtype", mtype)
		}
	}

	unit := name.Tags.Mtype()
	if unit == "" {
		unit = "jiff"
		if !tags_empty {
			name.Tags.Set("unit", unit)
		}
	}

	//fmt.Printf("IN: %v OUT: %v", tags, name.Tags)

	// if there are "tags" we use the Tags for the name, otherwise, we just the Key
	if !tags_empty {
		sort.Sort(name.Tags)
		name.Tags.WriteBytes(buf, repr.EQUAL_SEPARATOR_BYTE, repr.SPACE_SEPARATOR_BYTE)
	} else {
		s_tags := repr.SortingTags([][]string{
			{"mtype", mtype},
			{"unit", unit},
			{"what", name.Key},
		})
		s_tags.WriteBytes(buf, repr.EQUAL_SEPARATOR_BYTE, repr.SPACE_SEPARATOR_BYTE)

	}
	if !name.MetaTags.IsEmpty() {
		sort.Sort(name.MetaTags)
		buf.Write(repr.DOUBLE_SPACE_SEPARATOR_BYTE)
		name.MetaTags.WriteBytes(buf, repr.EQUAL_SEPARATOR_BYTE, repr.SPACE_SEPARATOR_BYTE)
	}
	buf.Write(repr.SPACE_SEPARATOR_BYTE)
	fmt.Fprintf(buf, "%f %d", val, tstamp)
	buf.Write(repr.NEWLINE_SEPARATOR_BYTES)
}
