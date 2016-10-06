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
	THe Whisper "indexer"

	Does not "index" as the metrics writer will do that, but does do the globing matcher

*/

package indexer

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"path/filepath"
	"regexp"
	"strings"
)

/****************** Interfaces *********************/
type WhisperIndexer struct {
	base_path string
	log       *logging.Logger
}

func NewWhisperIndexer() *WhisperIndexer {
	my := new(WhisperIndexer)
	my.log = logging.MustGetLogger("indexer.whisper")
	return my
}

func (ws *WhisperIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` /root/path/of/data is needed for whisper config")
	}

	ws.base_path = dsn

	// remove trialing "/"
	if strings.HasSuffix(dsn, "/") {
		ws.base_path = dsn[0 : len(dsn)-1]
	}
	return nil
}

func (ws *WhisperIndexer) Start() {
	//noop
}

func (ws *WhisperIndexer) Stop() {
	//noop
}

func (ws *WhisperIndexer) Name() string { return "whisper-indexer" }

func (ws *WhisperIndexer) Delete(name *repr.StatName) error {
	return nil //noop
}

//noop
func (ws *WhisperIndexer) Write(skey repr.StatName) error {
	return nil
}

// change {xxx,yyy} -> * as that's all the go lang glob can handle
// and so we turn it into a regex post
func (ws *WhisperIndexer) toGlob(metric string) (string, string, []string) {

	base_reg, out_strs := toGlob(metric)

	// need to include our base paths as we really are globing the file system
	glob_str := filepath.Join(ws.base_path, strings.Replace(base_reg, ".", "/", -1))
	reg_str := filepath.Join(ws.base_path, strings.Replace(base_reg, ".", "/", -1))
	reg_str = strings.Replace(base_reg, "/", "\\/", -1) + "(.*|.wsp)"

	return glob_str, reg_str, out_strs
}

/**** READER ***/
func (ws *WhisperIndexer) Find(metric string, tags repr.SortingTags) (MetricFindItems, error) {

	stats.StatsdClientSlow.Incr("indexer.whisper.finds", 1)

	// golangs globber does not handle "{a,b}" things, so we need to basically do a "*" then
	// perform a regex of the form (a|b) on the result ..
	glob_path, reg_str, _ := ws.toGlob(metric)

	var reger *regexp.Regexp
	var err error
	var mt MetricFindItems
	reger, err = regexp.Compile(reg_str)
	if err != nil {
		return mt, err
	}

	paths, err := filepath.Glob(glob_path)

	if err != nil {
		return mt, err
	}

	// a little special case for "exact" data metric matches
	// i.e. servers.all-1-stats-infra-integ.iostat.xvdg.writes
	// will match servers.all-1-stats-infra-integ.iostat.xvdg.writes and
	// servers.all-1-stats-infra-integ.iostat.xvdg.writes_bytes ...
	// so if we get an exact data hit .. just return that one
	for _, p := range paths {
		var ms MetricFindItem

		// convert to the "." scheme again
		t := strings.Replace(p, ws.base_path+"/", "", 1)

		is_data := filepath.Ext(p) == ".wsp"
		t = strings.Replace(t, ".wsp", "", -1)
		//ws.log.Critical("REG: %s, %s, %s", glob_path, reg_str, p)

		if !reger.MatchString(p) {
			continue
		}
		t = strings.Replace(t, "/", ".", -1)

		spl := strings.Split(t, ".")

		ms.Text = spl[len(spl)-1]

		ms.Id = p   // ID is whatever we want
		ms.Path = t // "path" is what our interface expects to be the "moo.goo.blaa" thing

		stat_name := repr.StatName{Key: p}
		if is_data {
			uid := stat_name.UniqueIdString()
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
			ms.UniqueId = uid
		} else {
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
		}

		// exact match special case
		if is_data && t == metric {
			mt_ext := make(MetricFindItems, 1)
			mt_ext[0] = ms
			return mt_ext, nil
		}

		mt = append(mt, ms)
	}
	return mt, nil
}

func (ws *WhisperIndexer) Expand(metric string) (MetricExpandItem, error) {
	// a basic "Dir" operation
	glob_path := filepath.Join(ws.base_path, strings.Replace(metric, ".", "/", -1))

	// golang's globber does not handle "{a,b}" things, so we need to basically do a "*" then
	// perform a regex of the form (a|b) on the result
	// TODO

	var mt MetricExpandItem
	paths, err := filepath.Glob(glob_path)
	if err != nil {
		return mt, err
	}

	for _, p := range paths {
		// convert to the "." scheme again
		t := strings.Replace(p, ws.base_path+"/", "", 1)
		t = strings.Replace(t, "/", ".", -1)

		is_data := filepath.Ext(p) == "wsp"
		t = strings.Replace(t, ".wsp", "", -1)

		if is_data {
			mt.Results = append(mt.Results, p)
		}

	}
	return mt, nil
}

func (ws *WhisperIndexer) List(has_data bool, page int) (MetricFindItems, error) {
	return MetricFindItems{}, errWillNotBeImplimented
}

func (ws *WhisperIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {
	return tags, metatags, errWillNotBeImplimented
}

func (ws *WhisperIndexer) GetTagsByName(name string, page int) (tags MetricTagItems, err error) {
	return tags, errWillNotBeImplimented
}

func (ws *WhisperIndexer) GetTagsByNameValue(name string, value string, page int) (tags MetricTagItems, err error) {
	return tags, errWillNotBeImplimented
}

func (ws *WhisperIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	return uids, errWillNotBeImplimented
}
