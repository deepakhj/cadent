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
   Graphite data runner, <key> <value> <time> <thigns> ...
   we allow there to be more <things> after value, so this is really "graphite style"
   space based line entries with the key as the first item
*/

package splitter

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

/****************** RUNNERS *********************/
const GRAPHITE_NAME = "graphite"

var errBadGraphiteLine = errors.New("Invalid Graphite/Space line")

var GRAPHITE_REPLACER *strings.Replacer
var GRAPHITE_REPLACER_BYTES = [][][]byte{
	{[]byte("/"), []byte(".")},
	{[]byte(".."), []byte(".")},
	{[]byte("=="), []byte("=")},
	{[]byte(","), []byte("_")},
	{[]byte("*"), []byte("_")},
	{[]byte("("), []byte("_")},
	{[]byte(")"), []byte("_")},
	{[]byte("{"), []byte("_")},
	{[]byte("}"), []byte("_")},
	{[]byte("  "), []byte(" ")},
}

var GRAPHITE_REPLACER_BYTE_MAP map[byte]byte

func init() {
	GRAPHITE_REPLACER = strings.NewReplacer(
		"/", ".",
		"..", ".",
		",", "_",
		"==", "=",
		"*", "_",
		"(", "_",
		")", "_",
		"{", "_",
		"}", "_",
		"  ", " ",
	)
	GRAPHITE_REPLACER_BYTE_MAP := make(map[byte]byte)
	GRAPHITE_REPLACER_BYTE_MAP['/'] = '.'
	GRAPHITE_REPLACER_BYTE_MAP[','] = '_'
	GRAPHITE_REPLACER_BYTE_MAP['*'] = '_'
	GRAPHITE_REPLACER_BYTE_MAP['{'] = '_'
	GRAPHITE_REPLACER_BYTE_MAP['}'] = '_'
	GRAPHITE_REPLACER_BYTE_MAP['('] = '_'
	GRAPHITE_REPLACER_BYTE_MAP[')'] = '_'
	GRAPHITE_REPLACER_BYTE_MAP[' '] = '_'
}

func graphiteByteReplace(bs []byte) []byte {
	bs = bytes.TrimSpace(bs)
	for i, b := range bs {
		if g, ok := GRAPHITE_REPLACER_BYTE_MAP[b]; ok {
			bs[i] = g
		}
	}
	bs = bytes.Replace(bs, []byte(".."), []byte("."), -1)
	bs = bytes.Replace(bs, []byte("=="), []byte("="), -1)
	return bs
}

type GraphiteSplitItem struct {
	inkey    []byte
	inline   []byte
	intime   time.Time
	infields [][]byte
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][][]byte
}

func (g *GraphiteSplitItem) Key() []byte {
	return g.inkey
}

func (g *GraphiteSplitItem) Tags() [][][]byte {
	return g.tags
}

func (g *GraphiteSplitItem) HasTime() bool {
	return true
}

func (g *GraphiteSplitItem) Timestamp() time.Time {
	return g.intime
}

func (g *GraphiteSplitItem) Line() []byte {
	return g.inline
}

func (g *GraphiteSplitItem) Fields() [][]byte {
	return g.infields
}

func (g *GraphiteSplitItem) Phase() Phase {
	return g.inphase
}

func (g *GraphiteSplitItem) SetPhase(n Phase) {
	g.inphase = n
}

func (g *GraphiteSplitItem) Origin() Origin {
	return g.inorigin
}

func (g *GraphiteSplitItem) SetOrigin(n Origin) {
	g.inorigin = n
}

func (g *GraphiteSplitItem) OriginName() string {
	return g.inoname
}

func (g *GraphiteSplitItem) SetOriginName(n string) {
	g.inoname = n
}

func (g *GraphiteSplitItem) IsValid() bool {
	return len(g.inline) > 0
}

func (g *GraphiteSplitItem) String() string {
	return fmt.Sprintf("Splitter: Graphite: %s @ %s", g.infields, g.intime)
}

type GraphiteSplitter struct {
	keyIndex  int
	timeIndex int
}

func (g *GraphiteSplitter) Name() (name string) { return GRAPHITE_NAME }

func NewGraphiteSplitter(conf map[string]interface{}) (*GraphiteSplitter, error) {

	//<key> <value> <time> <thigns>
	job := &GraphiteSplitter{
		keyIndex:  0,
		timeIndex: 2,
	}
	// allow for a config option to pick the proper thing in the line
	if idx, ok := conf["key_index"].(int); ok {
		job.keyIndex = idx
	}
	if idx, ok := conf["time_index"].(int); ok {
		job.timeIndex = idx
	}
	return job, nil
}

func (g *GraphiteSplitter) ProcessLine(line []byte) (SplitItem, error) {
	//<key> <value> <time> <more> <more>

	// clean out not-so-good chars
	line = graphiteByteReplace(line)

	//oddly strings Fields is faster then bytes fields, and we need strings later
	graphiteArray := strings.Fields(string(line))
	if len(graphiteArray) > g.keyIndex {

		// graphite timestamps are in unix seconds
		t := time.Time{}
		if len(graphiteArray) > g.timeIndex {
			i, err := strconv.ParseInt(graphiteArray[g.timeIndex], 10, 64)
			if err == nil {
				// nano or second tstamps
				if i > 2147483647 {
					t = time.Unix(0, i)
				} else {
					t = time.Unix(i, 0)
				}
			}
		}
		fs := [][]byte{}
		for _, j := range graphiteArray {
			fs = append(fs, []byte(j))
		}
		gi := getGraphiteItem()
		gi.inkey = []byte(graphiteArray[g.keyIndex])
		gi.inline = line
		gi.intime = t
		gi.infields = fs
		gi.inphase = Parsed
		gi.inorigin = Other
		/*
			// log.Printf("IN GRAPHITE: %s ARR: %v  t_idx: %d, time: %s", graphiteArray, line, graphiteArray[job.timeIndex], t.String())

			gi := &GraphiteSplitItem{
				inkey:    []byte(graphiteArray[g.keyIndex]),
				inline:   []byte(t_l),
				intime:   t,
				infields: fs,
				inphase:  Parsed,
				inorigin: Other,
			}*/
		return gi, nil
	}
	return nil, errBadGraphiteLine

}

var graphiteItemPool sync.Pool

func getGraphiteItem() *GraphiteSplitItem {
	x := graphiteItemPool.Get()
	if x == nil {
		return new(GraphiteSplitItem)
	}
	return x.(*GraphiteSplitItem)
}

func putGraphiteItem(spl *GraphiteSplitItem) {
	graphiteItemPool.Put(spl)
}
