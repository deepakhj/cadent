/*
   Carbon 2.0 data runner,

   <intrinsic_tags>  <meta_tags> <value> <timestamp>

   NOTE there are 2 spaces "  " between the intrinsic_tags and meta_tags

   intrinsic_tags is basically the "key" in the carbon1.0 format

*/

package splitter

import (
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
	inkey    string
	inline   string
	intime   time.Time
	infields []string
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][]string
}

func (g *CarbonTwoSplitItem) Key() string {
	return g.inkey
}

func (g *CarbonTwoSplitItem) Tags() [][]string {
	return g.tags
}

func (g *CarbonTwoSplitItem) HasTime() bool {
	return true
}

func (g *CarbonTwoSplitItem) Timestamp() time.Time {
	return g.intime
}

func (g *CarbonTwoSplitItem) Line() string {
	return g.inline
}

func (g *CarbonTwoSplitItem) Fields() []string {
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
func (g *CarbonTwoSplitter) ProcessLine(line string) (SplitItem, error) {

	line = CARBONTWO_REPLACER.Replace(line)

	stats_arr := strings.Split(line, "  ")
	var key string
	var vals []string

	if len(stats_arr) == 1 {
		key = stats_arr[0]
		t_vs := strings.Fields(line)
		vals = t_vs[1:]
	} else {
		key = stats_arr[0]
		vals = strings.Fields(stats_arr[1])
	}

	if len(vals) < 2 {
		return nil, errCarbonTwoNotValid
	}

	l_vals := len(vals)
	_intime := vals[l_vals-1] // should be unix timestamp

	t := time.Now()
	i, err := strconv.ParseInt(_intime, 10, 64)
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
	otags := make([][]string, 0)
	if l_vals >= 4 {
		for i := 0; i < l_vals-int(2); i++ {
			t_spl := strings.Split(vals[i], repr.EQUAL_SEPARATOR)
			otags = append(otags, t_spl)
		}
	}

	// log.Printf("IN GRAPHITE: %s ARR: %v  t_idx: %d, time: %s", graphite_array, line, graphite_array[job.time_index], t.String())
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
