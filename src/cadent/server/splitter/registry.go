/*
   New maker of splitters
*/

package splitter

import (
	"fmt"
)

func NewSplitterItem(name string, conf map[string]interface{}) (Splitter, error) {
	switch {
	case name == "statsd":
		return NewStatsdSplitter(conf)
	case name == "graphite":
		return NewGraphiteSplitter(conf)
	case name == "carbon2":
		return NewCarbonTwoSplitter(conf)
	case name == "regex":
		return NewRegExSplitter(conf)
	default:
		return nil, fmt.Errorf("Invalid splitter `%s`", name)
	}
}
