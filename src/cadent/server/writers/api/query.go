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
   functions to help parse the input params from ye old http internface
*/

package api

import (
	"cadent/server/repr"
	"cadent/server/writers/metrics"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var errTargetRequired = errors.New("Target is required")
var errInvalidStep = errors.New("Invalid `step` size")
var errInvalidStartTime = errors.New("Invalid `start` time")
var errInvalidEndTime = errors.New("Invalid `end` time")
var errInvalidPage = errors.New("Invalid `page`")

type MetricQuery struct {
	Target string
	Tags   repr.SortingTags
	Start  int64
	End    int64
	Step   uint32
	Agg    repr.AggType //sum, count, min, max, last, mean
}

type MetricQueryParsed struct {
	Metric  string
	Tags    repr.SortingTags
	AggFunc string
}

// parse a tag query of the form key{name=val, name=val...}
func ParseNameToTags(query string) (key string, tags repr.SortingTags, err error) {
	// find the bits inside the {}

	inner := ""
	collecting := false
	key_collecting := true
	for _, char := range query {
		switch char {
		case '{':
			collecting = true
			key_collecting = false
		case '}':
			collecting = false
		default:
			if collecting {
				inner += string(char)
			}
			if key_collecting {
				key += string(char)
			}
		}
	}

	if len(inner) == 0 || collecting {
		return key, tags, fmt.Errorf("Invalid Tag query `{name=val, name=val}`")
	}
	t_arr := strings.Split(inner, ",")
	for _, tg := range t_arr {
		t_split := strings.Split(strings.TrimSpace(tg), "=")
		if len(t_split) == 2 {
			tags = tags.Set(t_split[0], strings.Replace(t_split[1], "\"", "", -1))
		}
	}

	return key, tags, nil

}

func ParseMetricQuery(r *http.Request) (mq MetricQuery, err error) {
	r.ParseForm()
	var target string
	var from string
	var to string
	var agg repr.AggType
	var _tags string
	var tags repr.SortingTags

	l := len(r.Form["target"])
	for idx, tar := range r.Form["target"] {
		target += strings.TrimSpace(tar)
		switch {
		case idx < l-1:
			target += ","
		}
	}

	// if no target try "path"
	if len(target) == 0 {
		l = len(r.Form["path"])
		for idx, tar := range r.Form["path"] {
			target += strings.TrimSpace(tar)
			switch {
			case idx < l-1:
				target += ","
			}
		}
	}

	// if no target try "query"
	if len(target) == 0 {
		l = len(r.Form["query"])
		for idx, tar := range r.Form["query"] {
			target += strings.TrimSpace(tar)
			switch {
			case idx < l-1:
				target += ","
			}
		}
	}

	l = len(r.Form["tags"])
	for idx, tgs := range r.Form["tags"] {
		_tags += strings.TrimSpace(tgs)
		switch {
		case idx < l-1:
			_tags += ","
		}
	}
	if _tags != "" {
		tags = repr.SortingTagsFromString(_tags)
	}

	// find a agg if desired
	_agg := r.Form.Get("agg")
	if len(_agg) > 0 {
		agg = repr.AggTypeFromTag(_agg)
	}

	if len(target) > 0 {
		// see if the name is a key{tag,tag} thinggy
		a_key, a_tags, err := ParseNameToTags(target)
		if err == nil {
			target = a_key
			tags.Merge(a_tags)
		}
	}

	from = strings.TrimSpace(r.Form.Get("from"))
	to = strings.TrimSpace(r.Form.Get("to"))

	if len(target) == 0 {
		return mq, errTargetRequired
	}

	if len(from) == 0 {
		// try "start"
		from = strings.TrimSpace(r.Form.Get("start"))
	}
	if len(from) == 0 {
		from = "-1h"
	}
	if len(to) == 0 {
		// try "end"
		to = strings.TrimSpace(r.Form.Get("end"))
	}
	if len(to) == 0 {
		to = "now"
	}

	start, err := metrics.ParseTime(from)
	if err != nil {
		return mq, errInvalidStartTime
	}

	end, err := metrics.ParseTime(to)
	if err != nil {
		return mq, errInvalidEndTime
	}
	if end < start {
		start, end = end, start
	}

	// grab a step if desired (resolution resampling)
	_step := strings.TrimSpace(r.Form.Get("step"))

	step := uint64(0)
	if len(_step) > 0 {
		step, err = (strconv.ParseUint(_step, 10, 32))
		if err != nil {
			return mq, errInvalidStep
		}
	}

	return MetricQuery{
		Target: target,
		Start:  start,
		End:    end,
		Step:   uint32(step),
		Tags:   tags,
		Agg:    agg,
	}, nil
}

type IndexQuery struct {
	Query   string
	Value   string
	Page    int
	HasData bool
}

func ParseFindQuery(r *http.Request) (mq IndexQuery, err error) {
	r.ParseForm()

	vars := mux.Vars(r)

	query := strings.TrimSpace(r.Form.Get("name"))
	val := strings.TrimSpace(r.Form.Get("value"))
	inpage := strings.TrimSpace(r.Form.Get("page"))
	has_data := strings.TrimSpace(r.Form.Get("hasdata"))

	if len(query) == 0 {
		query = strings.TrimSpace(r.Form.Get("query"))
	}
	if len(query) == 0 {
		query = vars["name"]
	}

	if len(val) == 0 {
		val = vars["value"]
	}

	mq.Query = query
	mq.Value = val
	mq.HasData = true

	if len(has_data) > 0 && (has_data == "0" || has_data == "false") {
		mq.HasData = false
	}

	if len(inpage) > 0 {
		pg, err := (strconv.ParseUint(inpage, 10, 32))
		if err != nil {
			return mq, errInvalidPage
		}
		mq.Page = int(pg)
	}

	return

}