/*
   Fire up an HTTP server for using readers
*/

package readers

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"gopkg.in/op/go-logging.v1"
	"net/http"
	"strings"
)

type ReaderConfigOptions struct {
	Driver  string                 `toml:"driver"`
	DSN     string                 `toml:"dsn"`
	Options map[string]interface{} `toml:"options"` // option=[ [key, value], [key, value] ...]
}

type ReaderConfig struct {
	Listen      string              `toml:"listen"`
	BasePath    string              `toml:"base_path"`
	ReadOptions ReaderConfigOptions `toml:"reader"`
}

func (re *ReaderConfig) GetReader() (Reader, error) {
	reader, err := NewReader(re.ReadOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.ReadOptions.Options == nil {
		re.ReadOptions.Options = make(map[string]interface{})
	}
	re.ReadOptions.Options["dsn"] = re.ReadOptions.DSN
	err = reader.Config(re.ReadOptions.Options)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

type ReaderLoop struct {
	Conf   ReaderConfig
	Reader Reader

	log *logging.Logger
}

func ParseConfigString(inconf string) (rl *ReaderLoop, err error) {

	rl = new(ReaderLoop)
	if _, err := toml.Decode(inconf, &rl.Conf); err != nil {
		log.Critical("Error decoding config file: %s", err)
		return nil, err
	}

	rl.Reader, err = rl.Conf.GetReader()
	if err != nil {
		return nil, err
	}

	rl.SetBasePath(rl.Conf.BasePath)
	rl.log = logging.MustGetLogger("reader.http")
	return rl, nil
}

func (re *ReaderLoop) Config(conf ReaderConfig) (err error) {
	re.Conf = conf
	re.Reader, err = conf.GetReader()
	if err != nil {
		return err
	}
	re.SetBasePath(conf.BasePath)
	if re.log == nil {
		re.log = logging.MustGetLogger("reader.http")
	}
	return nil
}

func (re *ReaderLoop) SetBasePath(pth string) {
	re.Conf.BasePath = pth
	if len(re.Conf.BasePath) == 0 {
		re.Conf.BasePath = "/"
	}
	if !strings.HasSuffix(re.Conf.BasePath, "/") {
		re.Conf.BasePath += "/"
	}
	if !strings.HasPrefix(re.Conf.BasePath, "/") {
		re.Conf.BasePath = "/" + re.Conf.BasePath
	}
}

func (re *ReaderLoop) SetResolutions(res [][]int) {
	re.Reader.SetResolutions(res)
}

func (re *ReaderLoop) OutError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
	http.Error(w, msg, code)
	re.log.Error(msg)
}

func (re *ReaderLoop) OutJson(w http.ResponseWriter, data interface{}) {
	// cache theses things for 60 secs
	w.Header().Set("Cache-Control", "public, max-age=60, cache")

	stats, err := json.Marshal(data)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}
	w.Write(stats)
}

func (re *ReaderLoop) Find(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var query string

	if r.Method == "GET" {
		query = strings.TrimSpace(r.Form.Get("query"))
	} else {
		query = strings.TrimSpace(r.FormValue("query"))
	}

	if len(query) == 0 {
		re.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := re.Reader.Find(query)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.OutJson(w, data)
	return
}

func (re *ReaderLoop) Expand(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var query string

	if r.Method == "GET" {
		query = strings.TrimSpace(r.Form.Get("query"))
	} else {
		query = strings.TrimSpace(r.FormValue("query"))
	}

	if len(query) == 0 {
		re.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := re.Reader.Expand(query)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.OutJson(w, data)
}

func (re *ReaderLoop) Render(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var target string
	var from string
	var to string

	if r.Method == "GET" {
		target = strings.TrimSpace(r.Form.Get("target"))
		from = strings.TrimSpace(r.Form.Get("from"))
		to = strings.TrimSpace(r.Form.Get("to"))

	} else {
		target = strings.TrimSpace(r.FormValue("target"))
		from = strings.TrimSpace(r.FormValue("from"))
		to = strings.TrimSpace(r.FormValue("to"))
	}

	if len(target) == 0 {
		re.OutError(w, "Target is required", http.StatusBadRequest)
		return
	}

	if len(from) == 0 {
		from = "-1h"
	}

	if len(to) == 0 {
		to = "now"
	}

	data, err := re.Reader.Render(target, from, to)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.OutJson(w, data)
	return
}

func (re *ReaderLoop) Start() {
	mux := http.NewServeMux()
	re.log.Notice("Starting reader http server on %s, base path: %s", re.Conf.Listen, re.Conf.BasePath)

	mux.HandleFunc(re.Conf.BasePath+"find/", re.Find)
	mux.HandleFunc(re.Conf.BasePath+"find", re.Find)

	mux.HandleFunc(re.Conf.BasePath+"expand/", re.Expand)
	mux.HandleFunc(re.Conf.BasePath+"expand", re.Expand)

	mux.HandleFunc(re.Conf.BasePath+"render/", re.Render)
	mux.HandleFunc(re.Conf.BasePath+"render", re.Render)

	http.ListenAndServe(re.Conf.Listen, mux)
}
