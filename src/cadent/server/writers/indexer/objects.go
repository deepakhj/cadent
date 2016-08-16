/*
  Indexer Reader/Writer to match the GraphiteAPI .. but contains other things that may be useful for other API
  things
*/

package indexer

import (
	"cadent/server/repr"
)

// the basic metric json blob for find
type MetricFindItem struct {
	Text          string           `json:"text" `
	Expandable    int              `json:"expandable"`
	Leaf          int              `json:"leaf"`
	Id            string           `json:"id"`
	Path          string           `json:"path"`
	AllowChildren int              `json:"allowChildren"`
	UniqueId      string           `json:"uniqueid"` // can be nil
	Tags          repr.SortingTags `json:"tags"`
	MetaTags      repr.SortingTags `json:"meta_tags"`
}

// attempt to pick the "correct" metric based on the stats name
func (m *MetricFindItem) SelectValue() repr.AggType {
	if m.Leaf == 0 {
		return repr.SUM // not data
	}
	// stat wins
	tg := m.Tags.Stat()
	if tg != "" {
		return repr.AggTypeFromTag(tg)
	}
	return repr.GuessReprValueFromKey(m.Id)
}
func (m *MetricFindItem) StatName() *repr.StatName {
	return &repr.StatName{Key: m.Id, Tags: m.Tags, MetaTags: m.MetaTags}
}

type MetricFindItems []MetricFindItem

// the basic metric json blob for find
type MetricExpandItem struct {
	Results []string `json:"results"`
}
