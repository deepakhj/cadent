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
   Carbon 2.0 data runner,

   <intrinsic_tags>  <meta_tags> <value> <timestamp>

   NOTE there are 2 spaces "  " between the intrinsic_tags and meta_tags

   intrinsic_tags is basically the "key" in the carbon1.0 format

*/

package splitter

import (
	"bytes"
	"cadent/server/repr"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

/****************** RUNNERS *********************/
const CARBONTWO_NAME = "carbon2"

var errCarbonTwoNotValid = errors.New("Invalid Carbon2.0 line")
var errCarbonTwoUnitRequired = errors.New("unit Tag is required")
var errCarbonTwoMTypeRequired = errors.New("mtype Tag is required")

var CARBONTWO_REPLACER *strings.Replacer
var CARBONTWO_REPLACER_BYTES = [][][]byte{
	{[]byte(".."), []byte(".")},
	{[]byte(","), []byte("_")},
	{[]byte("*"), []byte("_")},
	{[]byte("("), []byte("_")},
	{[]byte(")"), []byte("_")},
	{[]byte("{"), []byte("_")},
	{[]byte("}"), []byte("_")},
}

func init() {
	CARBONTWO_REPLACER = strings.NewReplacer(
		"..", ".",
		",", "_",
		"*", "_",
		"(", "_",
		")", "_",
		"{", "_",
		"}", "_",
	)
}

type CarbonTwoSplitItem struct {
	inkey    []byte
	inline   []byte
	intime   time.Time
	infields [][]byte
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][][]byte
}

func (g *CarbonTwoSplitItem) Key() []byte {
	return g.inkey
}

func (g *CarbonTwoSplitItem) Tags() [][][]byte {
	return g.tags
}

func (g *CarbonTwoSplitItem) HasTime() bool {
	return true
}

func (g *CarbonTwoSplitItem) Timestamp() time.Time {
	return g.intime
}

func (g *CarbonTwoSplitItem) Line() []byte {
	return g.inline
}

func (g *CarbonTwoSplitItem) Fields() [][]byte {
	return g.infields
}

func (g *CarbonTwoSplitItem) Phase() Phase {
	return g.inphase
}

func (g *CarbonTwoSplitItem) SetPhase(n Phase) {
	g.inphase = n
}

func (g *CarbonTwoSplitItem) Origin() Origin {
	return g.inorigin
}

func (g *CarbonTwoSplitItem) SetOrigin(n Origin) {
	g.inorigin = n
}

func (g *CarbonTwoSplitItem) OriginName() string {
	return g.inoname
}

func (g *CarbonTwoSplitItem) SetOriginName(n string) {
	g.inoname = n
}

func (g *CarbonTwoSplitItem) IsValid() bool {
	return len(g.inline) > 0
}

func (g *CarbonTwoSplitItem) String() string {
	return fmt.Sprintf("Splitter: Carbon2: %s @ %s", g.infields, g.intime)
}

type CarbonTwoSplitter struct {
}

func (g *CarbonTwoSplitter) Name() (name string) { return CARBONTWO_NAME }

func NewCarbonTwoSplitter(conf map[string]interface{}) (*CarbonTwoSplitter, error) {

	//<intrinsic_tags>  <meta_tags> <value> <timestamp>
	job := &CarbonTwoSplitter{}

	return job, nil
}

/* <tag> <tag> <tag>  <metatags> <metatags> <metatags> <value> <time>
the hash key is <intrinsic_tags>
metatags are not part of the unique identifier so
should not be included in the hash key for accumulators
*/
func (g *CarbonTwoSplitter) ProcessLine(line []byte) (SplitItem, error) {

	//line = CARBONTWO_REPLACER.Replace(line)
	for _, repls := range CARBONTWO_REPLACER_BYTES {
		line = bytes.Replace(line, repls[0], repls[1], -1)
	}

	stats_arr := bytes.Split(line, repr.DOUBLE_SPACE_SEPARATOR_BYTE)
	var key []byte
	var vals [][]byte

	if len(stats_arr) == 1 { // the <tag> <tag> <tag> <value> <time> case
		t_vs := bytes.Fields(line)
		l_f := len(t_vs)
		if l_f < 3 {
			return nil, errCarbonTwoNotValid
		}
		key = bytes.Join(t_vs[0:l_f-2], repr.SPACE_SEPARATOR_BYTE)
		vals = t_vs[l_f-2:]

	} else { // the <tag> <tag> <tag>  <meta> ... <value> <time> case
		key = stats_arr[0]
		vals = bytes.Fields(stats_arr[1])
	}

	if len(vals) < 2 {
		return nil, errCarbonTwoNotValid
	}

	l_vals := len(vals)
	_intime := vals[l_vals-1] // should be unix timestamp

	t := time.Now()
	i, err := strconv.ParseInt(string(_intime), 10, 64)
	if err == nil {
		if i <= 0 {
			return nil, errCarbonTwoNotValid
		}
		// nano or second tstamps
		if i > 2147483647 {
			t = time.Unix(0, i)
		} else {
			t = time.Unix(i, 0)
		}
	}

	// parse tags
	otags := make([][][]byte, 0)
	if l_vals >= 4 {
		for i := 0; i < l_vals-int(2); i++ {
			t_spl := bytes.Split(vals[i], repr.EQUAL_SEPARATOR_BYTE)
			otags = append(otags, t_spl)
		}
	}

	gi := &CarbonTwoSplitItem{
		inkey:    key,
		inline:   line,
		intime:   t,
		infields: vals,
		tags:     otags,
		inphase:  Parsed,
		inorigin: Other,
	}
	return gi, nil

}
