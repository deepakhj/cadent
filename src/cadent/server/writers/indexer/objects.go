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
  Indexer Reader/Writer to match the GraphiteAPI .. but contains other things that may be useful for other API
  things
*/

package indexer

import (
	"cadent/server/repr"
	"errors"
)

var errWillNotBeImplimented = errors.New("CANNOT BE IMPLIMENTED")
var errNotYetImplimented = errors.New("Not yet implimented")

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

// Tag finder objects
type MetricTagItem struct {
	Id     interface{} `json:"id"`
	Name   string      `json:"name"`
	Value  string      `json:"value"`
	IsMeta bool        `json:"is_meta"`
}

type MetricTagItems []MetricTagItem
