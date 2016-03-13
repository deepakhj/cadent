/*
  Indexer Reader/Writer
*/

package indexer

import "strings"

// the basic metric json blob for find
type MetricFindItem struct {
	Text          string `json:"text"`
	Expandable    int    `json:"expandable"`
	Leaf          int    `json:"leaf"`
	Id            string `json:"id"`
	Path          string `json:"path"`
	AllowChildren int    `json:"allowChildren"`
}

func (m *MetricFindItem) SelectValue() string {
	if m.Leaf == 0 {
		return "sum" // not data
	}
	spl := strings.Split(m.Id, ".")
	last_path := spl[len(spl)-1]
	if strings.Contains(last_path, "mean") {
		return "mean"
	}
	if strings.Contains(last_path, "avg") {
		return "mean"
	}
	if strings.Contains(last_path, "average") {
		return "mean"
	}
	if strings.Contains(last_path, "upper") {
		return "max"
	}
	if strings.Contains(last_path, "max") {
		return "max"
	}
	if strings.Contains(last_path, "min") {
		return "min"
	}
	if strings.Contains(last_path, "lower") {
		return "min"
	}
	return "sum"
}

type MetricFindItems []MetricFindItem

// the basic metric json blob for find
type MetricExpandItem struct {
	Results []string `json:"results"`
}
