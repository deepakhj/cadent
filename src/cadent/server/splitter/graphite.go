/*
   Graphite data runner, <key> <value> <time> <thigns> ...
   we allow there to be more <things> after value, so this is really "graphite style"
   space based line entries with the key as the first item
*/

package splitter

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

/****************** RUNNERS *********************/
const GRAPHITE_NAME = "graphite"

var GRAHITE_REPLACER *strings.Replacer

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
	inkey    string
	inline   string
	intime   time.Time
	infields []string
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][]string
}

func (g *GraphiteSplitItem) Key() string {
	return g.inkey
}

func (g *GraphiteSplitItem) Tags() [][]string {
	return g.tags
}

func (g *GraphiteSplitItem) HasTime() bool {
	return true
}

func (g *GraphiteSplitItem) Timestamp() time.Time {
	return g.intime
}

func (g *GraphiteSplitItem) Line() string {
	return g.inline
}

func (g *GraphiteSplitItem) Fields() []string {
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

func (g *GraphiteSplitter) ProcessLine(line string) (SplitItem, error) {
	//<key> <value> <time> <more> <more>
	//graphite_array := strings.Fields(line)
	// clean the string of bad chars
	line = GRAHITE_REPLACER.Replace(line)
	graphite_array := strings.Split(line, " ")
	if len(graphite_array) > g.key_index {

		// graphite timestamps are in unix seconds
		t := time.Time{}
		if len(graphite_array) >= g.time_index {
			i, err := strconv.ParseInt(graphite_array[g.time_index], 10, 64)
			if err == nil {
				t = time.Unix(i, 0)
			}
		}
		// log.Printf("IN GRAPHITE: %s ARR: %v  t_idx: %d, time: %s", graphite_array, line, graphite_array[job.time_index], t.String())
		gi := &GraphiteSplitItem{
			inkey:    graphite_array[g.key_index],
			inline:   line,
			intime:   t,
			infields: graphite_array,
			inphase:  Parsed,
			inorigin: Other,
		}
		return gi, nil
	}
	return nil, fmt.Errorf("Invalid Graphite/Space line")

}
