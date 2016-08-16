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
   Writers/Readers of stats
*/

package writers

import (
	"cadent/server/repr"
)

/****************** Data writers *********************/
type Writer interface {
	Config(map[string]interface{}) error
	Write(repr.StatRepr) error
}

type Reader interface {
	Config(map[string]interface{}) error

	// impliments the graphite /metrics/find/?query=stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.*
	/*
			[
		{
		text: "accumulator",
		expandable: 1,
		leaf: 0,
		id: "stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.accumulator",
		allowChildren: 1
		}
		]
	*/
	Find(metric string)
}
