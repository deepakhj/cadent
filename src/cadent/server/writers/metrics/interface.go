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

   We are attempting to mimic the Graphite API json blobs throughout the process here
   such that we can hook this in directly to either graphite-api/web

   NOTE: this is not a full graphite DSL, just paths and metrics, we leave the fancy functions inside
   graphite-api/web to work their magic .. one day we'll impliment the full DSL, but until then ..

   Currently just implimenting /find /expand and /render (json only) for graphite-api
*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/writers/indexer"
	logging "gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("metrics")

type WritersNeeded int

const (
	AllResolutions  WritersNeeded = iota // all of them
	FirstResolution                      // just one
)

/****************** Data readers *********************/

type Metrics interface {

	// need to able to set what our resolutions are for ease of resolution picking
	// the INT return tells the agg loop if we need to have MULTI writers
	// i.e. for items that DO NOT self rollup (DBs) we need as many writers as resolutions
	// for Whisper (or 'other' things) we only need the Lowest time for the writers
	// as the whisper file rolls up internally
	SetResolutions([][]int) int

	Config(map[string]interface{}) error

	// need an Indexer 99% of the time to deal with render
	SetIndexer(indexer.Indexer) error

	// Writer
	Write(repr.StatRepr) error

	// Reader

	// /render?target=XXXX&from=-24h&to=now
	/*
			{
		target: "scaleToSeconds(stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.writer.cassandra.noncached-writes-path,1)",
		datapoints: [
		[
		0,
		1456087430
		],...
		]}
	*/
	Render(path string, from string, to string) (WhisperRenderItem, error)
	RawRender(path string, from string, to string) ([]*RawRenderItem, error)

	Stop()  // kill stuff
	Start() // fire it up
}
