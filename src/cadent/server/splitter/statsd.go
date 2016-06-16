/*
   statsd Runner and parser <key>:<value>|<type>
*/

package splitter

import (
	"fmt"
	"strings"
	"time"
)

const STATSD_NAME = "statsd"

type StatsdSplitItem struct {
	inkey    string
	inline   string
	infields []string
	inphase  Phase
	inorigin Origin
	inoname  string
}

func (g *StatsdSplitItem) Key() string {
	return g.inkey
}

func (g *StatsdSplitItem) HasTime() bool {
	return false
}

func (g *StatsdSplitItem) Timestamp() time.Time {
	return time.Time{}
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

func (g *StatsdSplitItem) Origin() Origin {
	return g.inorigin
}

func (g *StatsdSplitItem) SetOrigin(n Origin) {
	g.inorigin = n
}

func (g *StatsdSplitItem) OriginName() string {
	return g.inoname
}

func (g *StatsdSplitItem) SetOriginName(n string) {
	g.inoname = n
}
func (g *StatsdSplitItem) IsValid() bool {
	return len(g.inline) > 0
}

func (job *StatsdSplitItem) String() string {
	return fmt.Sprintf("Splitter: Statsd: %s", job.infields)
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
			inorigin: Other,
		}
		return si, nil
	}
	return nil, fmt.Errorf("Invalid Statsd line: " + line)

}
