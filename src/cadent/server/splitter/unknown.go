/*
  the "i have no idea" runner
*/

package splitter

import (
	"fmt"
	"time"
)

const UNKNOWN_NAME = "unknown"

type UnkSplitItem struct {
}

func (g *UnkSplitItem) Key() string {
	return ""
}

func (g *UnkSplitItem) Line() string {
	return ""
}

func (g *UnkSplitItem) HasTime() bool {
	return false
}

func (g *UnkSplitItem) Timestamp() time.Time {
	return time.Time{}
}

func (g *UnkSplitItem) Fields() []string {
	return []string{}
}

func (g *UnkSplitItem) Phase() Phase {
	return Parsed
}
func (g *UnkSplitItem) SetPhase(n Phase) {
}

func (g *UnkSplitItem) Origin() Origin {
	return Other
}

func (g *UnkSplitItem) SetOrigin(n Origin) {
}

func (g *UnkSplitItem) OriginName() string {
	return "n/a"
}

func (g *UnkSplitItem) SetOriginName(n string) {
}

func (g *UnkSplitItem) IsValid() bool {
	return false
}

var _unk_single = &UnkSplitItem{}

type UnknownSplitter struct{}

func (job *UnknownSplitter) Name() (name string) { return UNKNOWN_NAME }

func (job UnknownSplitter) ProcessLine(line string) (*UnkSplitItem, error) {
	return _unk_single, fmt.Errorf("Unknonwn line")
}

func NewUnknownSplitter(conf map[string]interface{}) (*UnknownSplitter, error) {
	return new(UnknownSplitter), nil
}

func BlankSplitterItem() *UnkSplitItem {
	return _unk_single
}
