/*
   Fire up an HTTP server for an http interface to the

   metrics/indexer interfaces
*/

package writers

import (
	"consthash/server/stats"
	"consthash/server/writers/indexer"
	"consthash/server/writers/metrics"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"gopkg.in/op/go-logging.v1"
	golog "log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ApiMetricConfig struct {
	Driver  string                 `toml:"driver"`
	DSN     string                 `toml:"dsn"`
	Options map[string]interface{} `toml:"options"` // option=[ [key, value], [key, value] ...]
}

type ApiIndexerConfig struct {
	Driver  string                 `toml:"driver"`
	DSN     string                 `toml:"dsn"`
	Options map[string]interface{} `toml:"options"` // option=[ [key, value], [key, value] ...]
}

type ApiConfig struct {
	Listen            string           `toml:"listen"`
	Logfile           string           `toml:"log_file"`
	BasePath          string           `toml:"base_path"`
	ApiMetricOptions  ApiMetricConfig  `toml:"metrics"`
	ApiIndexerOptions ApiIndexerConfig `toml:"indexer"`
}

func (re *ApiConfig) GetMetrics() (metrics.Metrics, error) {
	reader, err := metrics.NewMetrics(re.ApiMetricOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.ApiMetricOptions.Options == nil {
		re.ApiMetricOptions.Options = make(map[string]interface{})
	}
	re.ApiMetricOptions.Options["dsn"] = re.ApiMetricOptions.DSN
	err = reader.Config(re.ApiMetricOptions.Options)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func (re *ApiConfig) GetIndexer() (indexer.Indexer, error) {
	idx, err := indexer.NewIndexer(re.ApiIndexerOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.ApiIndexerOptions.Options == nil {
		re.ApiIndexerOptions.Options = make(map[string]interface{})
	}
	re.ApiIndexerOptions.Options["dsn"] = re.ApiIndexerOptions.DSN
	err = idx.Config(re.ApiIndexerOptions.Options)
	if err != nil {
		return nil, err
	}
	return idx, nil
}

type ApiLoop struct {
	Conf    ApiConfig
	Metrics metrics.Metrics
	Indexer indexer.Indexer

	log *logging.Logger
}

func ParseConfigString(inconf string) (rl *ApiLoop, err error) {

	rl = new(ApiLoop)
	if _, err := toml.Decode(inconf, &rl.Conf); err != nil {
		return nil, err
	}

	rl.Metrics, err = rl.Conf.GetMetrics()
	if err != nil {
		return nil, err
	}

	rl.Indexer, err = rl.Conf.GetIndexer()
	if err != nil {
		return nil, err
	}
	rl.Metrics.SetIndexer(rl.Indexer)
	rl.SetBasePath(rl.Conf.BasePath)
	rl.log = logging.MustGetLogger("reader.http")
	return rl, nil
}

func (re *ApiLoop) Config(conf ApiConfig) (err error) {
	if conf.Logfile == "" {
		conf.Logfile = "stdout"
	}
	re.Conf = conf

	re.Metrics, err = conf.GetMetrics()
	if err != nil {
		return err
	}

	re.Indexer, err = conf.GetIndexer()
	if err != nil {
		return err
	}
	re.Metrics.SetIndexer(re.Indexer)
	re.SetBasePath(conf.BasePath)
	if re.log == nil {
		re.log = logging.MustGetLogger("reader.http")
	}
	return nil
}

func (re *ApiLoop) SetBasePath(pth string) {
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

func (re *ApiLoop) SetResolutions(res [][]int) {
	re.Metrics.SetResolutions(res)
}

func (re *ApiLoop) OutError(w http.ResponseWriter, msg string, code int) {

	defer stats.StatsdClient.Incr("reader.http.errors", 1)
	w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
	http.Error(w, msg, code)
	re.log.Error(msg)
}

func (re *ApiLoop) OutJson(w http.ResponseWriter, data interface{}) {
	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.http.ok", 1)
	w.Header().Set("Cache-Control", "public, max-age=60, cache")

	stats, err := json.Marshal(data)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}
	w.Write(stats)
}

func (re *ApiLoop) Find(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.find.get-time-ns", time.Now())
	r.ParseForm()
	var query string

	query = strings.TrimSpace(r.Form.Get("query"))

	if len(query) == 0 {
		re.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := re.Indexer.Find(query)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.OutJson(w, data)
	return
}

func (re *ApiLoop) Expand(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.expand.get-time-ns", time.Now())
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

	data, err := re.Indexer.Expand(query)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.OutJson(w, data)
}

func (re *ApiLoop) Render(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.render.get-time-ns", time.Now())

	r.ParseForm()
	var target string
	var from string
	var to string

	for _, tar := range r.Form["target"] {
		target += strings.TrimSpace(tar) + ","
	}

	// if no target try "path"
	if len(target) == 0 {
		for _, tar := range r.Form["path"] {
			target += strings.TrimSpace(tar) + ","
		}
	}

	from = strings.TrimSpace(r.Form.Get("from"))
	to = strings.TrimSpace(r.Form.Get("to"))

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

	data, err := re.Metrics.Render(target, from, to)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.OutJson(w, data)
	return
}

func (re *ApiLoop) NoOp(w http.ResponseWriter, r *http.Request) {
	golog.Printf("No handler for this URL %s", r.URL)
	http.Error(w,
		fmt.Sprintf("Nothing here .. try %s/find or %s/render or %s/expand", re.Conf.BasePath, re.Conf.BasePath, re.Conf.BasePath),
		http.StatusNotFound,
	)
	return
}

func (re *ApiLoop) Start() {
	mux := http.NewServeMux()
	re.log.Notice("Starting reader http server on %s, base path: %s", re.Conf.Listen, re.Conf.BasePath)

	mux.HandleFunc(re.Conf.BasePath+"find/", re.Find)
	mux.HandleFunc(re.Conf.BasePath+"find", re.Find)
	mux.HandleFunc(re.Conf.BasePath+"paths/", re.Find)
	mux.HandleFunc(re.Conf.BasePath+"paths", re.Find)

	mux.HandleFunc(re.Conf.BasePath+"expand/", re.Expand)
	mux.HandleFunc(re.Conf.BasePath+"expand", re.Expand)

	mux.HandleFunc(re.Conf.BasePath+"render/", re.Render)
	mux.HandleFunc(re.Conf.BasePath+"render", re.Render)
	mux.HandleFunc(re.Conf.BasePath+"metrics/", re.Render)
	mux.HandleFunc(re.Conf.BasePath+"metrics", re.Render)

	mux.HandleFunc("/", re.NoOp)
	var outlog *os.File
	var err error
	if re.Conf.Logfile == "stderr" {
		outlog = os.Stderr
	} else if re.Conf.Logfile == "stdout" {
		outlog = os.Stdout
	} else if re.Conf.Logfile != "none" {
		outlog, err = os.OpenFile(re.Conf.Logfile, os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			golog.Panicf("Could not open Logfile %s, setting to stdout", re.Conf.Listen)
			outlog = os.Stdout

		}
	}
	http.ListenAndServe(re.Conf.Listen, WriteLog(mux, outlog))
}
