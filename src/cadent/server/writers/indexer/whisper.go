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
		ws.base_path = dsn[0 : len(dsn)-2]
	}
	return nil
}

//noop
func (ws *WhisperIndexer) Write(skey string) error {
	return nil
}

// change {xxx,yyy} -> * as that's all the globber can handle
func (ws *WhisperIndexer) toGlob(metric string) (string, string, []string) {

	outgs := []string{}
	got_first := false
	p_glob := ""
	out_str := ""
	reg_str := ""
	for _, _c := range metric {
		c := string(_c)
		switch c {
		case "{":
			got_first = true
			reg_str += "("
		case "}":
			if got_first && len(p_glob) > 0 {
				outgs = append(outgs, p_glob)
				reg_str += ")" //end regex
				out_str += "*" //glob
				got_first = false
			}
		case ",":
			if got_first {
				reg_str += "|" // glob , -> |
			} else {
				out_str += c
			}
		default:
			if !got_first {
				out_str += c
			} else {
				p_glob += c
			}
			reg_str += c

		}
	}
	// make a proper regex
	reg_str = strings.Replace(reg_str, "*", ".*", -1)
	if !strings.HasSuffix(out_str, "*") {
		out_str += "*"
	}
	glob_str := filepath.Join(ws.base_path, strings.Replace(out_str, ".", "/", -1))
	reg_str = filepath.Join(ws.base_path, strings.Replace(reg_str, ".", "/", -1))
	reg_str = strings.Replace(reg_str, "/", "\\/", -1) + "(.*|.wsp)"

	return glob_str, reg_str, outgs
}

/**** READER ***/
func (ws *WhisperIndexer) Find(metric string) (MetricFindItems, error) {

	stats.StatsdClientSlow.Incr("indexer.whisper.finds", 1)

	// golangs globber does not handle "{a,b}" things, so we need to basically do a "*" then
	// perform a regex of the form (a|b) on the result
	glob_path, reg_str, _ := ws.toGlob(metric)

	do_reg := reg_str != glob_path
	var reger *regexp.Regexp
	var err error
	var mt MetricFindItems
	if do_reg {
		reger, err = regexp.Compile(reg_str)
		if err != nil {
			return mt, err
		}
	}

	paths, err := filepath.Glob(glob_path)

	if err != nil {
		return mt, err
	}

	for _, p := range paths {
		var ms MetricFindItem

		// convert to the "." scheme again
		t := strings.Replace(p, ws.base_path+"/", "", 1)

		is_data := filepath.Ext(p) == ".wsp"
		t = strings.Replace(t, ".wsp", "", -1)
		ws.log.Critical("REG: %s, %s, %s", glob_path, reg_str, p)

		if do_reg {
			if !reger.Match([]byte(p)) {
				continue
			}
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
