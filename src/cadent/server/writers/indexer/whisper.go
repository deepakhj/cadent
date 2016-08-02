/*
	THe Whisper "indexer"

	Does not "index" as the metrics writer will do that, but does do the globing matcher

*/

package indexer

import (
	"cadent/server/repr"
	"cadent/server/stats"
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

func (ws *WhisperIndexer) Config(conf map[string]interface{}) error {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` /root/path/of/data is needed for whisper config")
	}
	dsn := gots.(string)
	ws.base_path = dsn

	// remove trialing "/"
	if strings.HasSuffix(dsn, "/") {
		ws.base_path = dsn[0 : len(dsn)-1]
	}
	return nil
}

func (ws *WhisperIndexer) Stop() {
	//noop
}

func (my *WhisperIndexer) Delete(name *repr.StatName) error {
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
func (ws *WhisperIndexer) Find(metric string) (MetricFindItems, error) {

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

		if !reger.Match([]byte(p)) {
			continue
		}
		t = strings.Replace(t, "/", ".", -1)

		spl := strings.Split(t, ".")

		ms.Text = spl[len(spl)-1]

		ms.Id = p   // ID is whatever we want
		ms.Path = t // "path" is what our interface expects to be the "moo.goo.blaa" thing

		if is_data {
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
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
