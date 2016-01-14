/*
   Graphite data runner, <key> <value> <time> <thigns> ...
   we allow there to be more <things> after value, so this is really "graphite style"
   space based line entries with the key as the first item
*/

package splitter

import (
	"fmt"
	"strings"
)

/****************** RUNNERS *********************/
const GRAPHITE_NAME = "graphite"

type GraphiteSplitItem struct {
	inkey    string
	inline   string
	infields []string
	inphase  Phase
	inorigin Origin
}

func (g *GraphiteSplitItem) Key() string {
	return g.inkey
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

func (g *GraphiteSplitItem) IsValid() bool {
	return len(g.inline) > 0
}

type GraphiteSplitter struct {
	key_index int
}

func (job *GraphiteSplitter) Name() (name string) { return GRAPHITE_NAME }

func NewGraphiteSplitter(conf map[string]interface{}) (*GraphiteSplitter, error) {

	//<key> <value> <time> <thigns>
	job := &GraphiteSplitter{
		key_index: 0,
	}
	// allow for a config option to pick the proper thing in the line
	if idx, ok := conf["key_index"].(int); ok {
		job.key_index = idx
	}
	return job, nil
}

func (job *GraphiteSplitter) ProcessLine(line string) (SplitItem, error) {
	//<key> <value> <time> <more> <more>
	//graphite_array := strings.Fields(line)
	graphite_array := strings.Split(line, " ")
	if len(graphite_array) > job.key_index {
		gi := &GraphiteSplitItem{
			inkey:    graphite_array[job.key_index],
			inline:   line,
			infields: graphite_array,
			inphase:  Parsed,
			inorigin: Other,
		}
		return gi, nil
	}
	return nil, fmt.Errorf("Invalid Graphite/Space line")

}
