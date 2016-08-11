/*
  Indexer Reader/Writer
*/

package indexer

import (
	"cadent/server/repr"
)

// the basic metric json blob for find
type MetricFindItem struct {
	Text          string `json:"text"`
	Expandable    int    `json:"expandable"`
	Leaf          int    `json:"leaf"`
	Id            string `json:"id"`
	Path          string `json:"path"`
	AllowChildren int    `json:"allowChildren"`
	UniqueId      string `json:"uniqueid"` // can be nil
}

// attempt to pick the "correct" metric based on the stats name
func (m *MetricFindItem) SelectValue() repr.AggType {
	if m.Leaf == 0 {
		return repr.SUM // not data
	}
	return repr.GuessReprValueFromKey(m.Id)
}
func (m *MetricFindItem) StatName() *repr.StatName {
	return &repr.StatName{Key: m.Id}
}

type MetricFindItems []MetricFindItem

// the basic metric json blob for find
type MetricExpandItem struct {
	Results []string `json:"results"`
}
