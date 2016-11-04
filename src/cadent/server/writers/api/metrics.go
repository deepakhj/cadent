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
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"sort"
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

func (re *MetricsAPI) GetMetrics(args MetricQuery) ([]*metrics.RawRenderItem, error) {
	stp := re.a.minResolution(args.Start, args.End, args.Step)
	return re.Metrics.RawRender(args.Target, args.Start, args.End, args.Tags, stp)
}

// WhisperRenderItem object for whisper/graphite outs
type WhisperRenderItem struct {
	RealStart uint32                          `json:"real_start,omitempty"`
	RealEnd   uint32                          `json:"real_end,omitempty"`
	Start     uint32                          `json:"start,omitempty"`
	End       uint32                          `json:"end,omitempty"`
	From      uint32                          `json:"from,omitempty"`
	To        uint32                          `json:"to,omitempty"`
	Step      uint32                          `json:"step,omitempty"`
	Series    map[string][]*metrics.DataPoint `json:"series,omitempty"`
}

// MarshalJSON custom json-er for sorting the series by keys
func (w *WhisperRenderItem) MarshalJSON() ([]byte, error) {
	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)

	buf.Write(repr.LEFT_BRACE_BYTES)

	fmt.Fprintf(buf, "\"real_start\": %d,\"real_end\": %d,\"start\": %d,\"end\": %d,\"from\": %d,\"to\": %d,\"step\": %d,\"series\": {",
		w.RealStart,
		w.RealEnd,
		w.Start,
		w.End,
		w.From,
		w.To,
		w.Step,
	)

	// grab names and sort them
	tNames := make([]string, len(w.Series))
	idx := 0
	for n := range w.Series {
		tNames[idx] = n
		idx++
	}
	sort.Strings(tNames)
	l := len(tNames)
	for i, n := range tNames {
		bytes, err := json.Marshal(w.Series[n])
		if err != nil {
			fmt.Println("Json Error: ", err)
			continue
		}
		fmt.Fprintf(buf, "\"%s\": ", n)
		buf.Write(bytes)
		if i < l-1 {
			buf.Write(repr.COMMA_SEPARATOR_BYTE)
		}
	}

	buf.Write(repr.RIGHT_BRACE_BYTES)
	buf.Write(repr.RIGHT_BRACE_BYTES)
	return buf.Bytes(), nil
}

// ToGraphiteRender take a rawrender and make it a graphite api json format
// meant for PyCadent hooked into the graphite-api backend storage item
func (re *MetricsAPI) ToGraphiteRender(raw_data []*metrics.RawRenderItem) *WhisperRenderItem {
	whis := new(WhisperRenderItem)
	whis.Series = make(map[string][]*metrics.DataPoint)

	if raw_data == nil {
		return nil
	}
	for _, data := range raw_data {
		if data == nil {
			continue
		}
		d_points := make([]*metrics.DataPoint, data.Len(), data.Len())
		whis.End = data.End
		whis.Start = data.Start
		whis.From = data.Start
		whis.To = data.End
		whis.Step = data.Step
		whis.RealEnd = data.RealEnd
		whis.RealStart = data.RealStart

		for idx, d := range data.Data {
			v := d.AggValue(data.AggFunc)
			d_points[idx] = &metrics.DataPoint{Time: d.Time, Value: v}
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

// Sorting for the target outputs
func (p GraphiteApiItems) Len() int { return len(p) }
func (p GraphiteApiItems) Less(i, j int) bool {
	return p[i].Target < p[j].Target
}
func (p GraphiteApiItems) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

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
			d_points[idx] = metrics.DataPoint{Time: d.Time, Value: v}
		}
		g_item.Datapoints = d_points
		graphite = append(graphite, g_item)
	}
	sort.Sort(graphite)
	return graphite
}

// if using another system (aka grafana, that expects the graphite-api delivered format)
func (re *MetricsAPI) GraphiteRender(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdSlowNanoTimeFunc("reader.http.graphite-render.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.graphite-render.hits", 1)

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.GetMetrics(args)
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

	data, err := re.GetMetrics(args)

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

	data, err := re.GetMetrics(args)

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

	// adds the gotten metrics to the internal read cache
	re.a.AddToCache(data)
	stats.StatsdClientSlow.Incr("reader.http.rawrender.ok", 1)

	re.a.OutJson(w, data)
	return
}
