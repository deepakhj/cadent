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
	case name == "json":
		return new(JsonAccumulate), nil
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
	case name == "carbon2" || name == "carbontwo":
		return new(CarbonTwoFormatter), nil
	default:
		return nil, fmt.Errorf("Invalid formatter %s", name)
	}
}
