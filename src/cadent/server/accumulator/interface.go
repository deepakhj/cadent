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
   Here we accumulate statsd metrics and then push to a output format of whatever
*/

package accumulator

import (
	"cadent/server/repr"
	"io"
	"time"
)

/****************** Interfaces *********************/
type StatItem interface {
	Key() repr.StatName
	StatTime() time.Time
	Type() string
	Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem)
	Accumulate(val float64, sample float64, stattime time.Time) error
	ZeroOut() error
	Repr() *repr.StatRepr
}

type AccumulatorItem interface {
	Init(FormatterItem) error
	Stats() map[string]StatItem
	Flush(buf io.Writer) *flushedList
	FlushList() *flushedList
	Name() string
	ProcessLine(line []byte) error
	ProcessRepr(stat *repr.StatRepr) error
	Reset() error
	Tags() repr.SortingTags
	SetKeepKeys(bool) error
	SetTagMode(mode uint8) error
	SetTags(repr.SortingTags)
	SetResolution(time.Duration) error
	GetResolution() time.Duration
	SetOptions([][]string) error
	GetOption(name string, defaults interface{}) interface{}
}

type FormatterItem interface {
	ToString(name *repr.StatName, val float64, tstamp int32, stats_type string, tags repr.SortingTags) string
	Write(buf io.Writer, name *repr.StatName, val float64, tstamp int32, stats_type string, tags repr.SortingTags)
	Type() string
	Init(...string) error
	SetAccumulator(AccumulatorItem)
	GetAccumulator() AccumulatorItem
}

// This is an internal struct used for the Accumulator to get both lines and StatReprs on a Flush
type flushedList struct {
	Lines []string
	Stats []*repr.StatRepr
}

func (fl *flushedList) Add(lines []string, stat *repr.StatRepr) {
	fl.Lines = append(fl.Lines, lines...)
	fl.Stats = append(fl.Stats, stat)
}

func (fl *flushedList) AddStat(stat *repr.StatRepr) {
	fl.Stats = append(fl.Stats, stat)
}
