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
   Get Metrics Handlers for the internal caches
*/

package api

import (
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type CacheAPI struct {
	a       *ApiLoop
	Indexer indexer.Indexer
	Metrics metrics.MetricsReader
}

func NewCacheAPI(a *ApiLoop) *CacheAPI {
	return &CacheAPI{
		a:       a,
		Indexer: a.Indexer,
		Metrics: a.Metrics,
	}
}

func (c *CacheAPI) AddHandlers(mux *mux.Router) {
	mux.HandleFunc("/cached/series/", c.GetSeriesFromCache)
	mux.HandleFunc("/cached/series", c.GetSeriesFromCache)

	mux.HandleFunc("/cache/", c.GetFromCache)
	mux.HandleFunc("/cache", c.GetFromCache)
}

// only return data that's in the write-back caches
// this is handy if you just want to poll what's currently being aggregated
// especially useful for the "series" style where writes to a DB system are much
// less then flat based mechanisms
func (re *CacheAPI) GetFromCache(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.cache-render.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.cache.hits", 1)

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.CacheRender(args.Target, args.Start, args.End, args.Tags)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.cache.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if data == nil {
		stats.StatsdClientSlow.Incr("reader.http.cache.nodata", 1)
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	stats.StatsdClientSlow.Incr("reader.http.cache.ok", 1)
	re.a.OutJson(w, data)
	return
}

// this will return the Raw BINARY series for metrics in the write-caches
// Note that ONLY ONE metric can be queries in this fashion as there is
// multi-series binary format ... yet
func (re *CacheAPI) GetSeriesFromCache(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.cache-render.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.cache-series.hits", 1)

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.CachedSeries(args.Target, args.Start, args.End, args.Tags)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.cache-series.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if data == nil {
		stats.StatsdClientSlow.Incr("reader.http.cache-series.nodata", 1)
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	// see if we want to base64 the beast
	to_base_64 := r.Form.Get("base64")

	w.Header().Set("Cache-Control", "public, max-age=60, cache")
	w.Header().Set("Content-Type", "application/cadent")
	w.Header().Set("X-CadentSeries-Key", data.Name.Key)
	w.Header().Set("X-CadentSeries-UniqueId", data.Name.UniqueIdString())

	t_str, _ := json.Marshal(data.Name.SortedTags())
	w.Header().Set("X-CadentSeries-Tags", string(t_str))
	t_str, _ = json.Marshal(data.Name.SortedMetaTags())
	w.Header().Set("X-CadentSeries-MetaTags", string(t_str))

	w.Header().Set("X-CadentSeries-Resolution", fmt.Sprintf("%d", data.Name.Resolution))
	w.Header().Set("X-CadentSeries-TTL", fmt.Sprintf("%d", data.Name.TTL))
	w.Header().Set("X-CadentSeries-Encoding", data.Series.Name())
	w.Header().Set("X-CadentSeries-Start", fmt.Sprintf("%d", data.Series.StartTime()))
	w.Header().Set("X-CadentSeries-End", fmt.Sprintf("%d", data.Series.LastTime()))
	w.Header().Set("X-CadentSeries-Points", fmt.Sprintf("%d", data.Series.Count()))

	switch to_base_64 {
	case "1":
		w.Header().Set("Content-Transfer-Encoding", "base64")
		b_encoder := base64.NewEncoder(base64.RawStdEncoding, w)
		b_encoder.Write(data.Series.Bytes())
		b_encoder.Close()
	default:
		w.Header().Set("Content-Type", "application/cadent")

		w.Write(data.Series.Bytes())

	}
	stats.StatsdClientSlow.Incr("reader.http.cache-series.ok", 1)

	return
}
