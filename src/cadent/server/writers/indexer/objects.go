/*
  Indexer Reader/Writer
*/

package indexer

import (
	"cadent/server/repr"
	"regexp"
	"strings"
)

var _upperReg *regexp.Regexp
var _upperMaxReg *regexp.Regexp
var _lowerReg *regexp.Regexp
var _lowerMinReg *regexp.Regexp

func init() {
	_upperReg, _ = regexp.Compile(".*upper_[0-9]+$")
	_upperMaxReg, _ = regexp.Compile(".*max_[0-9]+$")
	_lowerReg, _ = regexp.Compile(".*lower_[0-9]+$")
	_lowerMinReg, _ = regexp.Compile(".*min_[0-9]+$")
}

func GuessAggregateType(metric string) string {
	spl := strings.Split(metric, ".")
	last_path := spl[len(spl)-1]

	// statsd like things are "mean_XX", "upper_XX", "lower_XX", "count_XX"
	if strings.Contains(last_path, "mean") {
		return "mean"
	}
	if strings.Contains(last_path, "avg") {
		return "mean"
	}
	// specials for "counts"
	if strings.HasSuffix(metric, "count") || strings.HasPrefix(metric, "stats.count") {
		return "sum"
	}
	// last/gauges use the last val
	if strings.HasSuffix(metric, "last") {
		return "last"
	}
	if strings.HasSuffix(metric, "gauge") || strings.HasPrefix(metric, "stats.gauge") {
		return "last"
	}

	// specials for "requests"
	if strings.HasSuffix(metric, "requests") {
		return "sum"
	}
	if strings.Contains(last_path, "sum") {
		return "sum"
	}
	if strings.HasSuffix(last_path, "errors") || strings.HasSuffix(last_path, "error") {
		return "sum"
	}
	if strings.Contains(last_path, "std") { // standard deviation
		return "mean"
	}
	if strings.Contains(last_path, "average") {
		return "mean"
	}
	if _upperReg.Match([]byte(last_path)) {
		return "max"
	}
	if strings.HasSuffix(last_path, "max") || _upperMaxReg.MatchString(last_path) {
		return "max"
	}
	if strings.HasSuffix(last_path, "min") || _lowerMinReg.MatchString(last_path) {
		return "min"
	}
	if _lowerReg.MatchString(last_path) {
		return "min"
	}
	if strings.HasSuffix(last_path, "lower") || _lowerReg.MatchString(last_path) {
		return "min"
	}
	return "mean"
}

// the basic metric json blob for find
type MetricFindItem struct {
	Text          string `json:"text"`
	Expandable    int    `json:"expandable"`
	Leaf          int    `json:"leaf"`
	Id            string `json:"id"`
	Path          string `json:"path"`
	AllowChildren int    `json:"allowChildren"`
}

// attempt to pick the "correct" metric based on the stats name
func (m *MetricFindItem) SelectValue() string {
	if m.Leaf == 0 {
		return "sum" // not data
	}
	return GuessAggregateType(m.Id)
}
func (m *MetricFindItem) StatName() *repr.StatName {
	return &repr.StatName{Key: m.Id}
}

type MetricFindItems []MetricFindItem

// the basic metric json blob for find
type MetricExpandItem struct {
	Results []string `json:"results"`
}
