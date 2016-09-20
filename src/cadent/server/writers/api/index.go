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

   Http handlers for the "find me a metric" please

*/

package api

import (
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
	"time"
)

type FindAPI struct {
	a       *ApiLoop
	indexer indexer.Indexer
	metrics metrics.Metrics
}

func NewFindAPI(a *ApiLoop) *FindAPI {
	return &FindAPI{
		a:       a,
		indexer: a.Indexer,
		metrics: a.Metrics,
	}
}

func (f *FindAPI) AddHandlers(mux *mux.Router) {
	// these are for a "raw" graphite like finder (skipping the graphite-api)
	mux.HandleFunc("/metrics/find/", f.Find)
	mux.HandleFunc("/metrics/find", f.Find)

	// for 'py-cadent'
	mux.HandleFunc("/find", f.Find)
	mux.HandleFunc("/paths", f.Find)
	mux.HandleFunc("/expand", f.Expand)

	mux.HandleFunc("/list", f.List)
}

func (re *FindAPI) Find(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdSlowNanoTimeFunc("reader.http.find.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.find.hits", 1)
	r.ParseForm()

	args, err := ParseFindQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	if len(args.Query) == 0 {
		re.a.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := re.indexer.Find(args.Query, args.Tags)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.find.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	stats.StatsdClientSlow.Incr("reader.http.find.ok", 1)
	re.a.OutJson(w, data)
	return
}

func (f *FindAPI) Expand(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdSlowNanoTimeFunc("reader.http.expand.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.expand.hits", 1)
	r.ParseForm()
	var query string

	if r.Method == "GET" {
		query = strings.TrimSpace(r.Form.Get("query"))
	} else {
		query = strings.TrimSpace(r.FormValue("query"))
	}

	if len(query) == 0 {
		f.a.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := f.indexer.Expand(query)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.expand.errors", 1)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	stats.StatsdClientSlow.Incr("reader.http.expand.ok", 1)
	f.a.OutJson(w, data)
}

func (f *FindAPI) List(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdSlowNanoTimeFunc("reader.http.list.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.list.hits", 1)

	args, err := ParseFindQuery(r)

	if err != nil {
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}

	datas, err := f.indexer.List(args.HasData, args.Page)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.list.errors", 1)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if datas == nil {
		f.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}
	res := []string{}
	for _, data := range datas {

		res = append(res, data.Path)
	}
	stats.StatsdClientSlow.Incr("reader.http.list.ok", 1)

	f.a.OutJson(w, res)
}
