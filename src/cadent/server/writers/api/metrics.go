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
   Get Metrics Handlers
*/

package api

import (
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type MetricsAPI struct {
	a       *ApiLoop
	Indexer indexer.Indexer
	Metrics metrics.MetricsReader
}

func NewMetricsAPI(a *ApiLoop) *MetricsAPI {
	return &MetricsAPI{
		a:       a,
		Indexer: a.Indexer,
		Metrics: a.Metrics,
	}
}

func (m *MetricsAPI) AddHandlers(mux *mux.Router) {
	mux.HandleFunc("/render", m.Render)
	mux.HandleFunc("/metrics", m.Render)
	mux.HandleFunc("/rawrender", m.RawRender)
}

// take a rawrender and make it a graphite api json format
func (re *MetricsAPI) ToGraphiteRender(raw_data []*metrics.RawRenderItem) *metrics.WhisperRenderItem {
	whis := new(metrics.WhisperRenderItem)
	whis.Series = make(map[string][]metrics.DataPoint)
	if raw_data == nil {
		return nil
	}
	for _, data := range raw_data {
		if data == nil {
			continue
		}
		d_points := make([]metrics.DataPoint, data.Len(), data.Len())
		whis.End = data.End
		whis.Start = data.Start
		whis.Step = data.Step
		whis.RealEnd = data.RealEnd
		whis.RealStart = data.RealStart

		for idx, d := range data.Data {
			v := d.AggValue(data.AggFunc)
			d_points[idx] = metrics.DataPoint{Time: d.Time, Value: &v}
		}
		whis.Series[data.Metric] = d_points
	}
	return whis
}

func (re *MetricsAPI) Render(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.render.get-time-ns", time.Now())

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.RawRender(args.Target, args.Start, args.End)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if data == nil {
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	render_data := re.ToGraphiteRender(data)
	re.a.AddToCache(data)

	re.a.OutJson(w, render_data)
	return
}

func (re *MetricsAPI) RawRender(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.rawrender.get-time-ns", time.Now())

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.RawRender(args.Target, args.Start, args.End)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if data == nil {
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	re.a.AddToCache(data)

	re.a.OutJson(w, data)
	return
}
