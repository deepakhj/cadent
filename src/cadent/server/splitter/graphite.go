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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

/****************** RUNNERS *********************/
const GRAPHITE_NAME = "graphite"

var errBadGraphiteLine = errors.New("Invalid Graphite/Space line")

var GRAHITE_REPLACER *strings.Replacer
var GRAHITE_REPLACER_BYTES = [][][]byte{
	{[]byte(".."), []byte(".")},
	{[]byte(","), []byte("_")},
	{[]byte("*"), []byte("_")},
	{[]byte("("), []byte("_")},
	{[]byte(")"), []byte("_")},
	{[]byte("{"), []byte("_")},
	{[]byte("}"), []byte("_")},
	{[]byte("  "), []byte(" ")},
}

func init() {
	GRAHITE_REPLACER = strings.NewReplacer(
		"..", ".",
		",", "_",
		"*", "_",
		"(", "_",
		")", "_",
		"{", "_",
		"}", "_",
		"  ", " ",
	)
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
	key_index  int
	time_index int
}

func (g *GraphiteSplitter) Name() (name string) { return GRAPHITE_NAME }

func NewGraphiteSplitter(conf map[string]interface{}) (*GraphiteSplitter, error) {

	//<key> <value> <time> <thigns>
	job := &GraphiteSplitter{
		key_index:  0,
		time_index: 2,
	}
	// allow for a config option to pick the proper thing in the line
	if idx, ok := conf["key_index"].(int); ok {
		job.key_index = idx
	}
	if idx, ok := conf["time_index"].(int); ok {
		job.time_index = idx
	}
	return job, nil
}

func (g *GraphiteSplitter) ProcessLine(line []byte) (SplitItem, error) {
	//<key> <value> <time> <more> <more>
	//graphite_array := strings.Fields(line)
	// clean the string of bad chars
	/*for _, repls := range GRAHITE_REPLACER_BYTES {
		line = bytes.Replace(line, repls[0], repls[1], -1)
	}*/
	t_l := string(line)
	t_l = GRAHITE_REPLACER.Replace(t_l)

	graphite_array := strings.Fields(t_l)
	if len(graphite_array) > g.key_index {

		// graphite timestamps are in unix seconds
		t := time.Time{}
		if len(graphite_array) > g.time_index {
			i, err := strconv.ParseInt(graphite_array[g.time_index], 10, 64)
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
		for _, j := range graphite_array {
			fs = append(fs, []byte(j))
		}
		// log.Printf("IN GRAPHITE: %s ARR: %v  t_idx: %d, time: %s", graphite_array, line, graphite_array[job.time_index], t.String())
		gi := &GraphiteSplitItem{
			inkey:    []byte(graphite_array[g.key_index]),
			inline:   []byte(t_l),
			intime:   t,
			infields: fs,
			inphase:  Parsed,
			inorigin: Other,
		}
		return gi, nil
	}
	return nil, errBadGraphiteLine

}
