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
   Find Tags api handlers
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

type TagAPI struct {
	a       *ApiLoop
	Indexer indexer.Indexer
	Metrics metrics.MetricsReader
}

func NewTagAPI(a *ApiLoop) *TagAPI {
	return &TagAPI{
		a:       a,
		Indexer: a.Indexer,
		Metrics: a.Metrics,
	}
}

func (t *TagAPI) AddHandlers(mux *mux.Router) {
	// tags
	mux.HandleFunc("/tag/find/byname", t.FindTagsByName)

	mux.HandleFunc("/tag/find/bynamevalue", t.FindTagsByNameAndValue)

	mux.HandleFunc("/tag/uid/bytags", t.FindUidsByTags)

}

func (re *TagAPI) FindTagsByName(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.find.tag.name.get-time-ns", time.Now())
	stats.StatsdClientSlow("reader.http.tagbyname.hits", 1)
	r.ParseForm()

	args, err := ParseFindQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}
	if len(args.Query) == 0 {
		re.a.OutError(w, "Name is required", http.StatusBadRequest)
		return
	}

	data, err := re.Indexer.GetTagsByName(args.Query, args.Page)
	if err != nil {
		stats.StatsdClientSlow("reader.http.tagbyname.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	stats.StatsdClientSlow("reader.http.tagbyname.ok", 1)
	re.a.OutJson(w, data)
	return
}

func (re *TagAPI) FindTagsByNameAndValue(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.find.tag.namevalue.get-time-ns", time.Now())
	stats.StatsdClientSlow("reader.http.tagbynameval.hits", 1)
	r.ParseForm()
	args, err := ParseFindQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}
	if len(args.Query) == 0 {
		re.a.OutError(w, "Name is required", http.StatusBadRequest)
		return
	}
	if len(args.Value) == 0 {
		re.a.OutError(w, "Value is required", http.StatusBadRequest)
		return
	}

	data, err := re.Indexer.GetTagsByNameValue(args.Query, args.Value, args.Page)
	if err != nil {
		stats.StatsdClientSlow("reader.http.tagbynameval.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	stats.StatsdClientSlow("reader.http.tagbynameval.ok", 1)
	re.a.OutJson(w, data)
	return
}

// finduidsbytag?q={name=val, name=val, ...}

func (re *TagAPI) FindUidsByTags(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.find.tag.uids.get-time-ns", time.Now())
	stats.StatsdClientSlow("reader.http.uidbytag.hits", 1)
	r.ParseForm()
	args, err := ParseFindQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}
	if len(args.Query) == 0 {
		re.a.OutError(w, "`query` is required", http.StatusBadRequest)
		return
	}

	key, tags, err := indexer.ParseOpenTSDBTags(args.Query)

	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%s", err), http.StatusBadRequest)
		return
	}
	if len(tags) == 0 {
		re.a.OutError(w, fmt.Sprintf("No tags found"), http.StatusBadRequest)
		return
	}
	if len(tags) > 64 {
		re.a.OutError(w, fmt.Sprintf("Too many tags found"), http.StatusBadRequest)
		return
	}

	data, err := re.Indexer.GetUidsByTags(key, tags, args.Page)
	if err != nil {
		stats.StatsdClientSlow("reader.http.uidbytag.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	stats.StatsdClientSlow("reader.http.uidbytag.ok", 1)
	re.a.OutJson(w, data)
	return
}
