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
	Metrics metrics.Metrics
}

func NewMetricsAPI(a *ApiLoop) *MetricsAPI {
	return &MetricsAPI{
		a:       a,
		Indexer: a.Indexer,
		Metrics: a.Metrics,
	}
}

func (m *MetricsAPI) AddHandlers(mux *mux.Router) {
	// raw graphite (skipping graphite-api)
	mux.HandleFunc("/render", m.GraphiteRender)

	// for py-cadent
	mux.HandleFunc("/metrics", m.Render)

	// cadent specific
	mux.HandleFunc("/rawrender", m.RawRender)
}

// take a rawrender and make it a graphite api json format
// meant for PyCadent hooked into the graphite-api backend storage item
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

// take a rawrender and make it a graphite api json format
// meant for PyCadent hooked into the graphite-api backend storage item
type GraphiteApiItem struct {
	Target     string              `json:"target"`
	Datapoints []metrics.DataPoint `json:"datapoints"`
}
type GraphiteApiItems []*GraphiteApiItem

func (re *MetricsAPI) ToGraphiteApiRender(raw_data []*metrics.RawRenderItem) GraphiteApiItems {
	graphite := make(GraphiteApiItems, 0)

	if raw_data == nil {
		return nil
	}

	for _, data := range raw_data {
		if data == nil {
			continue
		}
		g_item := new(GraphiteApiItem)
		d_points := make([]metrics.DataPoint, data.Len(), data.Len())
		g_item.Target = data.Metric

		for idx, d := range data.Data {
			v := d.AggValue(data.AggFunc)
			d_points[idx] = metrics.DataPoint{Time: d.Time, Value: &v}
		}
		g_item.Datapoints = d_points
		graphite = append(graphite, g_item)
	}
	return graphite
}

// if using another system (aka grafan, that expects the graphite-api delivered format)
func (re *MetricsAPI) GraphiteRender(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdSlowNanoTimeFunc("reader.http.graphite-render.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.graphite-render.hits", 1)

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.RawRender(args.Target, args.Start, args.End, args.Tags, args.Step)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.graphite-render.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if data == nil {
		stats.StatsdClientSlow.Incr("reader.http.graphite-render.nodata", 1)
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	for idx := range data {
		if data == nil {
			continue
		}
		// graphite needs the "nils" and expects a "full list" to match the step + start/end
		data[idx].Start = uint32(args.Start)
		data[idx].End = uint32(args.End)
		data[idx].Quantize()

	}
	render_data := re.ToGraphiteApiRender(data)

	re.a.AddToCache(data)
	stats.StatsdClientSlow.Incr("reader.http.graphite-render.ok", 1)

	re.a.OutJson(w, render_data)
	return
}

func (re *MetricsAPI) Render(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdSlowNanoTimeFunc("reader.http.render.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.render.hits", 1)

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.RawRender(args.Target, args.Start, args.End, args.Tags, args.Step)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.render.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if data == nil {
		stats.StatsdClientSlow.Incr("reader.http.render.nodata", 1)
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	for idx := range data {
		// graphite needs the "nils" and expects a "full list" to match the step + start/end
		data[idx].Start = uint32(args.Start)
		data[idx].End = uint32(args.End)
		data[idx].Quantize()
	}
	render_data := re.ToGraphiteRender(data)

	re.a.AddToCache(data)

	stats.StatsdClientSlow.Incr("reader.http.render.ok", 1)
	re.a.OutJson(w, render_data)
	return
}

func (re *MetricsAPI) RawRender(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdSlowNanoTimeFunc("reader.http.rawrender.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.rawrender.hits", 1)

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.RawRender(args.Target, args.Start, args.End, args.Tags, args.Step)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.rawrender.error", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if data == nil {
		stats.StatsdClientSlow.Incr("reader.http.rawrender.nodata", 1)
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	resample := args.Step
	if resample > 0 {
		for idx := range data {
			data[idx].Resample(resample)
		}
	}

	re.a.AddToCache(data)
	stats.StatsdClientSlow.Incr("reader.http.rawrender.ok", 1)

	re.a.OutJson(w, data)
	return
}
