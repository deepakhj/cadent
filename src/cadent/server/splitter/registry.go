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
	case name == "opentsdb":
		return NewOpenTSDBSplitter(conf)
	default:
		return nil, fmt.Errorf("Invalid splitter `%s`", name)
	}
}
