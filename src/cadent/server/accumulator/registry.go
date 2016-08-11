/*
   Simple make of new objects
*/

package accumulator

import (
	"fmt"
)

func NewAccumulatorItem(name string) (AccumulatorItem, error) {
	switch {
	case name == "graphite":
		return new(GraphiteAccumulate), nil
	case name == "statsd":
		return new(StatsdAccumulate), nil
	case name == "carbon2" || name == "carbontwo":
		return new(CarbonTwoAccumulate), nil
	default:
		return nil, fmt.Errorf("Invalid accumulator %s", name)
	}
}

func NewFormatterItem(name string) (FormatterItem, error) {
	switch {
	case name == "graphite":
		return new(GraphiteFormatter), nil
	case name == "statsd":
		return new(StatsdFormatter), nil
	default:
		return nil, fmt.Errorf("Invalid formatter %s", name)
	}
}
