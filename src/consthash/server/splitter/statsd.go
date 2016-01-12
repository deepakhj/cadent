/*
   statsd Runner and parser <key>:<value>|<type>
*/

package splitter

import (
	"fmt"
	"strings"
)

const STATSD_NAME = "statsd"

type StatsdSplitItem struct {
	inkey    string
	inline   string
	infields []string
	inphase  Phase
}

func (g *StatsdSplitItem) Key() string {
	return g.inkey
}

func (g *StatsdSplitItem) Line() string {
	return g.inline
}

func (g *StatsdSplitItem) Fields() []string {
	return g.infields
}

func (g *StatsdSplitItem) Phase() Phase {
	return g.inphase
}

func (g *StatsdSplitItem) SetPhase(n Phase) {
	g.inphase = n
}

func (g *StatsdSplitItem) IsValid() bool {
	return len(g.inline) > 0
}

type StatsdSplitter struct {
}

func (job *StatsdSplitter) Name() (name string) { return STATSD_NAME }

func NewStatsdSplitter(conf map[string]interface{}) (*StatsdSplitter, error) {

	//<key>:blaaa
	job := &StatsdSplitter{}
	return job, nil
}

func (job *StatsdSplitter) ProcessLine(line string) (SplitItem, error) {

	statd_array := strings.Split(line, ":")
	if len(statd_array) >= 2 {
		si := &StatsdSplitItem{
			inkey:    statd_array[0],
			inline:   line,
			infields: statd_array,
			inphase:  Parsed,
		}
		return si, nil
	}
	return nil, fmt.Errorf("Invalid Statsd line: " + line)

}
