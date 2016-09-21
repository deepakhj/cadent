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
  A "mock" prometheus handlers, not all apis are complete
  nor is the full query language

*/

package api

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

/**************** fake some prometheus like APIs ******************/

type PrometheusAPI struct {
	a       *ApiLoop
	Indexer indexer.Indexer
	Metrics metrics.Metrics
}

func NewPrometheusAPI(a *ApiLoop) *PrometheusAPI {
	return &PrometheusAPI{
		a:       a,
		Indexer: a.Indexer,
		Metrics: a.Metrics,
	}
}

func (p *PrometheusAPI) AddHandlers(mux *mux.Router) {
	mux.HandleFunc("/prometheus/api/v1/query_range", p.QueryRange).Methods("GET")

	// special list o paths query
	mux.HandleFunc("/prometheus/api/v1/label/__name__/values", p.MetricNames).Methods("GET")
	mux.HandleFunc("/prometheus/api/v1/label/{name}/values", p.TagValues).Methods("GET")

}

func (p *PrometheusAPI) OutError(w http.ResponseWriter, msg string, code int) {
	defer stats.StatsdClient.Incr(fmt.Sprintf("reader.http.prometheus.%d.errors", code), 1)
	w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
	w.Header().Set("Content-Type", "application/json")
	jmsg := fmt.Sprintf(`{"status": "error", "error":"%s"}`, msg)
	http.Error(w, jmsg, code)
	p.a.log.Errorf(msg)
}

// curl 'http://localhost:9090/api/v1/query_range?query=up&start=2015-07-01T20:10:30.781Z&end=2015-07-01T20:11:00.781Z&step=15s'
/*
{
   "status" : "success",
   "data" : {
      "resultType" : "matrix",
      "result" : [
         {
            "metric" : {
               "__name__" : "up",
               "job" : "prometheus",
               "instance" : "localhost:9090"
            },
            "values" : [
               [ 1435781430.781, "1" ],
               [ 1435781445.781, "1" ],
               [ 1435781460.781, "1" ]
            ]
         },
         {
            "metric" : {
               "__name__" : "up",
               "job" : "node",
               "instance" : "localhost:9091"
            },
            "values" : [
               [ 1435781430.781, "0" ],
               [ 1435781445.781, "0" ],
               [ 1435781460.781, "1" ]
            ]
         }
      ]
   }
}
*/

type promMetricResult struct {
	Metric map[string]string       `json:"metric"`
	Values [][]repr.NilJsonFloat64 `json:"values"`
}

type promMatrixResultType struct {
	ResultType string             `json:"resultType"`
	Result     []promMetricResult `json:"result"`
}

type promMetricResponse struct {
	Status string               `json:"status"`
	Data   promMatrixResultType `json:"data"`
}

func (p *PrometheusAPI) QueryRange(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.promrender.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.promrender.hits", 1)

	args, err := ParseMetricQuery(r)
	if err != nil {
		p.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}
	datas, err := p.Metrics.RawRender(
		args.Target, args.Start, args.End, args.Tags, p.a.minResolution(args.Start, args.End, args.Step),
	)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.promrender.errors", 1)
		p.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if datas == nil {
		stats.StatsdClientSlow.Incr("reader.http.promrender.nodata", 1)
		p.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	resample := args.Step

	// now for the remapping into a prometheuos expected item
	res := promMetricResponse{
		Status: "success",
		Data: promMatrixResultType{
			ResultType: "matrix",
		},
	}
	for _, data := range datas {
		if data == nil {
			continue
		}
		p_data := promMetricResult{
			Metric: make(map[string]string, 0),
		}
		p_data.Metric["__name__"] = data.Metric

		if resample > 0 {
			data.Resample(resample)
		}

		if !data.Tags.IsEmpty() {
			for _, tg := range data.Tags {
				p_data.Metric[tg[0]] = tg[1]
			}
		}
		if !data.MetaTags.IsEmpty() {
			for _, tg := range data.MetaTags {
				p_data.Metric[tg[0]] = tg[1]
			}
		}
		nm := &repr.StatName{Key: data.Metric, Tags: data.Tags}
		agg_func := nm.AggType()
		if args.Agg > 0 {
			agg_func = args.Agg
		}
		for _, vals := range data.Data {
			p_data.Values = append(
				p_data.Values,
				[]repr.NilJsonFloat64{
					repr.NilJsonFloat64(vals.Time),
					repr.NilJsonFloat64(vals.AggValue(agg_func)),
				})
		}
		res.Data.Result = append(res.Data.Result, p_data)
	}

	// send to activator
	p.a.AddToCache(datas)
	stats.StatsdClientSlow.Incr("reader.http.promrender.ok", 1)

	p.a.OutJson(w, res)
}

// here is just a "list" of the path names in the DB
// we have to limit this as it can be huge so we have to paginate it

type promNamesResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

func (p *PrometheusAPI) MetricNames(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdSlowNanoTimeFunc("reader.http.promlist.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.promlist.hits", 1)

	args, err := ParseFindQuery(r)

	if err != nil {
		p.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}

	datas, err := p.Indexer.List(args.HasData, args.Page)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.promlist.errors", 1)
		p.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if datas == nil {
		p.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	// now for the remapping into a prometheuos expected item
	res := promNamesResponse{
		Status: "success",
	}
	for _, data := range datas {

		res.Data = append(res.Data, data.Path)
	}
	stats.StatsdClientSlow.Incr("reader.http.promlist.ok", 1)

	p.a.OutJson(w, res)
}

func (p *PrometheusAPI) TagValues(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdSlowNanoTimeFunc("reader.http.promnames.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.promnames.ok", 1)

	args, err := ParseFindQuery(r)

	if err != nil {
		p.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}

	datas, err := p.Indexer.GetTagsByName(args.Query, args.Page)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.promnames.errors", 1)
		p.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if datas == nil {
		p.OutError(w, "No data found", http.StatusNoContent)
		return
	}
	res := promNamesResponse{
		Status: "success",
	}
	for _, data := range datas {

		res.Data = append(res.Data, data.Value)
	}
	stats.StatsdClientSlow.Incr("reader.http.promnames.ok", 1)
	p.a.OutJson(w, res)
}
