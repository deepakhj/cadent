/*
	THe Whisper "indexer"

	Does not "index" as the metrics writer will do that, but does do the globing matcher

*/

package indexer

import (
	"cadent/server/stats"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"path/filepath"
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
		ws.base_path = dsn[0 : len(dsn)-2]
	}
	return nil
}

//noop
func (ws *WhisperIndexer) Write(skey string) error {
	return nil
}

/**** READER ***/
func (ws *WhisperIndexer) Find(metric string) (MetricFindItems, error) {

	stats.StatsdClientSlow.Incr("indexer.whisper.finds", 1)
	glob_path := filepath.Join(ws.base_path, strings.Replace(metric, ".", "/", -1)) + "*"

	// golangs globber does not handle "{a,b}" things, so we need to basically do a "*" then
	// perform a regex of the form (a|b) on the result
	// TODO

	var mt MetricFindItems
	paths, err := filepath.Glob(glob_path)

	if err != nil {
		return mt, err
	}

	for _, p := range paths {
		var ms MetricFindItem

		// convert to the "." scheme again
		t := strings.Replace(p, ws.base_path+"/", "", 1)
		t = strings.Replace(t, "/", ".", -1)

		is_data := filepath.Ext(p) == ".wsp"
		t = strings.Replace(t, ".wsp", "", -1)

		spl := strings.Split(t, ".")

		ms.Text = spl[len(spl)-1]

		ms.Id = t
		ms.Path = p

		if is_data {
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
		} else {
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
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
