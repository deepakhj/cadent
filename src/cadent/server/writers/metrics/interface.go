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
	"cadent/server/series"
	"cadent/server/utils"
	"cadent/server/writers/indexer"
	logging "gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("metrics")

type WritersNeeded int

const (
	AllResolutions  WritersNeeded = iota // all of them
	FirstResolution                      // just one
)

// Writer interface ..
type Metrics interface {

	// the name of the driver
	Driver() string

	// need to able to set what our resolutions are for ease of resolution picking
	// the INT return tells the agg loop if we need to have MULTI writers
	// i.e. for items that DO NOT self rollup (DBs) we need as many writers as resolutions
	// for Whisper (or 'other' things) we only need the Lowest time for the writers
	// as the whisper file rolls up internally
	SetResolutions([][]int) int
	GetResolutions() [][]int

	// we can have a writer per resolution, so this just sets the one we are currently on
	SetCurrentResolution(int)

	IsPrimaryWriter() bool
	Cache() *Cacher
	CachePrefix() string

	Config(map[string]interface{}) error

	// need an Indexer 99% of the time to deal with render
	SetIndexer(indexer.Indexer) error

	// Writer
	Write(repr.StatRepr) error

	RawRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error)

	// just get data in the write-back caches
	CacheRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error)

	// return the cached data as the raw binary series
	// note for now only ONE metric can be returned using this method
	CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*TotalTimeSeries, error)

	Stop()  // kill stuff
	Start() // fire it up
}

// for those writers that are "blob" writers, we need them to match this interface
// so that we can do resolution rollups
type DBMetrics interface {

	// the name of the driver
	Driver() string

	// gets the latest point(s) writen
	GetLatestFromDB(name *repr.StatName, resolution uint32) (DBSeriesList, error)

	// get the series that fit in a window
	GetRangeFromDB(name *repr.StatName, start uint32, end uint32, resolution uint32) (DBSeriesList, error)

	// update a db row
	UpdateDBSeries(dbs *DBSeries, ts series.TimeSeries) error

	//add a new row
	InsertDBSeries(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (int, error)
}

/***************
	a "base" object for all writers
*************************/

type WriterBase struct {
	indexer           indexer.Indexer
	resolutions       [][]int
	currentResolution int
	static_tags       repr.SortingTags

	// this is for Render where we may have several caches, but only "one"
	// cacher get picked for the default render (things share the cache from writers
	// and the api render, but not all the caches, so we need to be able to get the
	// the cache singleton keys
	// `cache:series:seriesMaxMetrics:seriesEncoding:seriesMaxBytes:maxTimeInCache`
	cacherPrefix string
	cacher       *Cacher
	is_primary   bool // is this the primary writer to the cache?

	shutitdown bool
	startstop  utils.StartStop
}

// Resolutions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (wb *WriterBase) SetResolutions(res [][]int) int {
	wb.resolutions = res
	return len(res) // need as many writers as bins
}

func (wb *WriterBase) GetResolutions() [][]int {
	return wb.resolutions
}

func (wb *WriterBase) SetCurrentResolution(res int) {
	wb.currentResolution = res
}

func (wb *WriterBase) SetIndexer(idx indexer.Indexer) error {
	wb.indexer = idx
	return nil
}

func (wc *WriterBase) IsPrimaryWriter() bool {
	return wc.is_primary
}

func (wc *WriterBase) Cache() *Cacher {
	return wc.cacher
}

func (wc *WriterBase) CachePrefix() string {
	return wc.cacherPrefix
}
