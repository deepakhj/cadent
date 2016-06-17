/*

LevelDB key/value store for local saving for indexes on the stat key space
and how it maps to files on disk.

This is much like how Prometheus does things, stores a index in levelDB
and single "custom" files for each series.

We have "2" dbs, one for a
{key} -> {file} mapping
and
{tag} -> {key} mapping.

If no tags are used (i.e. graphite like) then this will be basically empty

We follow a data pattern alot like the Cassandra one, as we still need to be able to do the
usual graphite like glob goodies

*/

package indexer

import (
	"cadent/server/stats"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"path/filepath"
	"regexp"
	"strings"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/gocql/gocql"
)



/*

Segment index

First set ::  key is SEG:{segment} value is {pos}
Second set :: key is SEGPOS:{segment}:{pos} value is {segment}
Third set :: key is POS:{pos} value is {segment}

*/

type LevelDBSegment struct {
	Pos     int
	Segment string
}



/*
 Path index

 First Set key is SEG:{segment}:{pos} value is {path}:{hasdata}
 First Set key is SEG:{segment}:{pos} value is {path}



*/
type LevelDBPath struct {
	Segment CassSegment
	Path    string
	Length  int
	Hasdata bool
}


/****************** Interfaces *********************/
type LevelDBIndexer struct {
	base_path string
	log       *logging.Logger
}

func NewLevelDBIndexer() *LevelDBIndexer {
	my := new(LevelDBIndexer)
	my.log = logging.MustGetLogger("indexer.leveldb")
	return my
}

func (ws *LevelDBIndexer) Config(conf map[string]interface{}) error {
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

func (ws *LevelDBIndexer) Stop() {
	//noop
}

//noop
func (ws *LevelDBIndexer) Write(skey string) error {
	return nil
}

// change {xxx,yyy} -> * as that's all the go lang glob can handle
// and so we turn it into t regex post
func (ws *LevelDBIndexer) toGlob(metric string) (string, string, []string) {

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
				reg_str += "|" // glob , -> regex |
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
func (ws *LevelDBIndexer) Find(metric string) (MetricFindItems, error) {

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

func (ws *LevelDBIndexer) Expand(metric string) (MetricExpandItem, error) {
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
