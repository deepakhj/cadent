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

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/repr"
	"cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrorMustBeSingleCacheType  = errors.New("The cacher for this writer needs to be of the Single Type")
	ErrorMustBeChunkedCacheType = errors.New("The cacher for this writer needs to be of the Chunked Type")
)

// Cache singletons as the readers may need to use this too

// the singleton
var _CACHER_SINGLETON map[string]Cacher
var _cacher_mutex sync.RWMutex

func GetCacherSingleton(nm string, mode string) (Cacher, error) {
	_cacher_mutex.Lock()
	defer _cacher_mutex.Unlock()

	if val, ok := _CACHER_SINGLETON[nm]; ok {
		return val, nil
	}
	var cacher Cacher
	switch mode {
	case "chunk":
		cacher = NewCacherChunk()

	default:
		cacher = NewSingleCacher()

	}
	_CACHER_SINGLETON[nm] = cacher
	cacher.SetName(nm)
	return cacher, nil
}

// just GET by name if it exists
func GetCacherByName(nm string) Cacher {
	_cacher_mutex.RLock()
	defer _cacher_mutex.RUnlock()

	if val, ok := _CACHER_SINGLETON[nm]; ok {
		return val
	}
	return nil
}

// special onload init
func init() {
	_CACHER_SINGLETON = make(map[string]Cacher)
}

// interface for cache objects

type Cacher interface {
	// Add a metric to the cache
	Add(name *repr.StatName, stat *repr.StatRepr) (err error)

	// set the cache name
	SetName(nm string)

	// get the cache name
	GetName() string

	// set the cache prefix
	SetPrefix(nm string)

	// get the cache prefix
	GetPrefix() string

	// SetMaxKeys for a the cache
	SetMaxKeys(m int)

	// GetMaxKeys for a the cache
	GetMaxKeys() int

	// SetMaxBytesPerMetric max number of bytes stored per metric
	SetMaxBytesPerMetric(m int)

	// GetMaxBytesPerMetric max number of bytes stored per metric
	GetMaxBytesPerMetric() int

	// SetSeriesEncoding internal encocing type
	SetSeriesEncoding(m string)

	// SetMaxTimeInCache max time allowed in cache
	SetMaxTimeInCache(m uint32)

	// SetOverFlowMethod bytes overflow method "drop" or "chan"
	SetOverFlowMethod(m string)

	// GetMaxOverFlowMethod bytes overflow method "drop" or "chan"
	GetOverFlowMethod() string

	// SetMaxBroadcastLen length for the broadcast overflow channel
	SetMaxBroadcastLen(m int)

	// SetLowFruitRate how often to choose "low count" stats vs high
	SetLowFruitRate(m float64)

	// SetPrimaryWriter set oe writer per cache
	SetPrimaryWriter(mw Metrics) bool

	// IsPrimaryWriter is this metric driver the primary writer
	IsPrimaryWriter(mw Metrics) bool

	// GetOverFlowChan grab a listener to the overflow broadcast channel
	GetOverFlowChan() *broadcast.Listener

	// Len number of metrics in the cache
	Len() int

	// Start the cacher
	Start()

	// Stop the cacher
	Stop()

	// Get an element from the cache
	Get(name *repr.StatName) (repr.StatReprSlice, error)

	// GetAsRawRenderItem an element as a rendered item
	GetAsRawRenderItem(name *repr.StatName) (*RawRenderItem, error)

	// GetById gets a metric by the unique ID
	GetById(metric_id repr.StatId) (*repr.StatName, repr.StatReprSlice, error)

	// GetSeries get raw timeseries by name
	GetSeries(name *repr.StatName) (*repr.StatName, []series.TimeSeries, error)

	// GetSeriesById gets a timeseries by the unique ID
	GetSeriesById(metric_id repr.StatId) (*repr.StatName, []series.TimeSeries, error)

	// GetCurrentSeries get raw timeseries by name
	GetCurrentSeries(name *repr.StatName) (*repr.StatName, series.TimeSeries, error)

	// GetCurrentSeriesById gets a timeseries by the unique ID
	GetCurrentSeriesById(metric_id repr.StatId) (*repr.StatName, series.TimeSeries, error)
}

// Base cacher object
type CacherBase struct {
	mu             *sync.RWMutex
	qmu            *sync.Mutex
	maxTimeInCache uint32
	maxKeys        int // max num of keys to keep before we have to drop
	maxBytes       int // max num of points per key to keep before we have to drop
	seriesType     string
	curSize        int64
	numCurPoint    int
	numCurKeys     int
	lowFruitRate   float64                // % of the time we reverse the max sortings to persist low volume stats
	shutdown       *broadcast.Broadcaster // when recieved stop allowing adds and updates

	Name         string `json:"name"` // just a human name for things
	statsdPrefix string
	Prefix       string `json:"prefix"` // caches names are "Prefix:{resolution}" or should be

	// for the overflow cached items::
	// these caches can be shared for a given writer set, and the caches may provide the data for
	// multiple writers, we need to specify that ONE of the writers is the "main" one otherwise
	// the Metrics Write function will add the points over again, which is not a good thing
	// when the accumulator flushes things to the multi writer
	// The Writer needs to know it's "not" the primary writer and thus will not "add" points to the
	// cache .. so the cache basically gets "one" primary writer pointed (first come first serve)
	// the `Write` function in the writer object should check it's the primary
	PrimaryWriter Metrics `json:"-"`

	//overflow pieces
	overFlowMethod string

	// allow for multiple registering entities
	overFlowBroadcast *broadcast.Broadcaster // should pass in *TotalTimeSeries

	startstop utils.StartStop
	started   bool
	inited    bool
}

func (wc *CacherBase) SetName(nm string) {
	wc.Name = nm
	wc.statsdPrefix = fmt.Sprintf("cacher.%s.metrics.", stats.SanitizeName(nm))
}

func (wc *CacherBase) GetName() string {
	return wc.Name
}

func (wc *CacherBase) SetPrefix(nm string) {
	wc.Prefix = nm
}

func (wc *CacherBase) GetPrefix() string {
	return wc.Prefix
}

func (wc *CacherBase) SetMaxKeys(m int) {
	wc.maxKeys = m
}

func (wc *CacherBase) GetMaxKeys() int {
	return wc.maxKeys
}

func (wc *CacherBase) SetMaxBytesPerMetric(m int) {
	wc.maxBytes = m
}

func (wc *CacherBase) GetMaxBytesPerMetric() int {
	return wc.maxBytes
}

func (wc *CacherBase) SetSeriesEncoding(m string) {
	wc.seriesType = m
}
func (wc *CacherBase) SetMaxTimeInCache(m uint32) {
	wc.maxTimeInCache = m
}
func (wc *CacherBase) SetOverFlowMethod(m string) {
	wc.overFlowMethod = m
}
func (wc *CacherBase) GetOverFlowMethod() string {
	return wc.overFlowMethod
}
func (wc *CacherBase) SetMaxBroadcastLen(m int) {
	wc.overFlowBroadcast = broadcast.New(m)
}
func (wc *CacherBase) SetLowFruitRate(m float64) {
	wc.lowFruitRate = m
}

func (wc *CacherBase) SetPrimaryWriter(mw Metrics) bool {
	if wc.PrimaryWriter == nil {
		wc.PrimaryWriter = mw
		return true
	}
	return false
}

func (wc *CacherBase) IsPrimaryWriter(mw Metrics) bool {
	return wc.PrimaryWriter == mw
}

func (wc *CacherBase) GetOverFlowChan() *broadcast.Listener {
	return wc.overFlowBroadcast.Listen()
}
