/*
   Fire up an HTTP server for using readers
*/

package readers

import (
	"consthash/server/stats"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"gopkg.in/op/go-logging.v1"
	"net/http"
	"os"
	"strings"
	"time"
)

type ReaderConfigOptions struct {
	Driver  string                 `toml:"driver"`
	DSN     string                 `toml:"dsn"`
	Options map[string]interface{} `toml:"options"` // option=[ [key, value], [key, value] ...]
}

type ReaderConfig struct {
	Listen      string              `toml:"listen"`
	Logfile     string              `toml:"log_file"`
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
	if conf.Logfile == "" {
		conf.Logfile = "stdout"
	}
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

	defer stats.StatsdClient.Incr("reader.http.errors", 1)
	w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
	http.Error(w, msg, code)
	re.log.Error(msg)
}

func (re *ReaderLoop) OutJson(w http.ResponseWriter, data interface{}) {
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

func (re *ReaderLoop) Find(w http.ResponseWriter, r *http.Request) {
	defer stats.StatsdNanoTimeFunc("reader.http.find.get-time-ns", time.Now())
	r.ParseForm()
	var query string

	query = strings.TrimSpace(r.Form.Get("query"))

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

	data, err := re.Reader.Expand(query)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.OutJson(w, data)
}

func (re *ReaderLoop) Render(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.render.get-time-ns", time.Now())

	r.ParseForm()
	var target string
	var from string
	var to string

	for _, tar := range r.Form["target"] {
		target += strings.TrimSpace(tar) + ","
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

	data, err := re.Reader.Render(target, from, to)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	re.OutJson(w, data)
	return
}

func (re *ReaderLoop) NoOp(w http.ResponseWriter, r *http.Request) {
	log.Warning("No handler for this URL %s", r.URL)
}

func (re *ReaderLoop) Start() {
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
			log.Error("Could not open Logfile %s, setting to stdout", re.Conf.Listen)
			outlog = os.Stdout

		}
	}
	http.ListenAndServe(re.Conf.Listen, WriteLog(mux, outlog))
}
