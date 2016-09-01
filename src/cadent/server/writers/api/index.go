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
	metrics metrics.MetricsReader
}

func NewFindAPI(a *ApiLoop) *FindAPI {
	return &FindAPI{
		a:       a,
		indexer: a.Indexer,
		metrics: a.Metrics,
	}
}

func (f *FindAPI) AddHandlers(mux *mux.Router) {
	mux.HandleFunc("/find", f.Find)
	mux.HandleFunc("/paths", f.Find)
	mux.HandleFunc("/expand", f.Expand)
}

func (re *FindAPI) Find(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.find.get-time-ns", time.Now())
	r.ParseForm()
	var query string

	query = strings.TrimSpace(r.Form.Get("query"))

	if len(query) == 0 {
		re.a.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := re.indexer.Find(query)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.a.OutJson(w, data)
	return
}

func (re *FindAPI) Expand(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.expand.get-time-ns", time.Now())
	r.ParseForm()
	var query string

	if r.Method == "GET" {
		query = strings.TrimSpace(r.Form.Get("query"))
	} else {
		query = strings.TrimSpace(r.FormValue("query"))
	}

	if len(query) == 0 {
		re.a.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := re.indexer.Expand(query)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.a.OutJson(w, data)
}