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
   Fire up an HTTP server for an http interface to the

   metrics/indexer interfaces

   example config
   [graphite-proxy-map.accumulator.api]
        base_path = "/graphite/"
        listen = "0.0.0.0:8083"
            [graphite-proxy-map.accumulator.api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            # this is the read cache that will keep the latest goods in ram
            read_cache_max_items=102400
            read_cache_max_bytes_per_metric=8192

            [graphite-proxy-map.accumulator.api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"
*/

package writers

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"gopkg.in/op/go-logging.v1"
	golog "log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var errTargetRequired = errors.New("Target is required")

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
	Listen                     string           `toml:"listen"`
	Logfile                    string           `toml:"log_file"`
	BasePath                   string           `toml:"base_path"`
	ApiMetricOptions           ApiMetricConfig  `toml:"metrics"`
	ApiIndexerOptions          ApiIndexerConfig `toml:"indexer"`
	MaxReadCacheBytes          int              `toml:"read_cache_total_bytes"`
	MaxReadCacheBytesPerMetric int              `toml:"read_cache_max_bytes_per_metric"`
}

func (re *ApiConfig) GetMetrics(resolution float64) (metrics.MetricsReader, error) {
	reader, err := metrics.NewReaderMetrics(re.ApiMetricOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.ApiMetricOptions.Options == nil {
		re.ApiMetricOptions.Options = make(map[string]interface{})
	}
	re.ApiMetricOptions.Options["dsn"] = re.ApiMetricOptions.DSN
	re.ApiMetricOptions.Options["resolution"] = resolution

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
	Metrics metrics.MetricsReader
	Indexer indexer.Indexer

	shutdown chan bool
	log      *logging.Logger

	ReadCache           *metrics.ReadCache
	activate_cache_chan chan *metrics.RawRenderItem

	started bool
}

func ParseConfigString(inconf string) (rl *ApiLoop, err error) {

	rl = new(ApiLoop)
	if _, err := toml.Decode(inconf, &rl.Conf); err != nil {
		return nil, err
	}

	rl.Metrics, err = rl.Conf.GetMetrics(10.0) // stub "10s" as the res
	if err != nil {
		return nil, err
	}

	rl.Indexer, err = rl.Conf.GetIndexer()
	if err != nil {
		return nil, err
	}
	rl.started = false
	rl.Metrics.SetIndexer(rl.Indexer)
	rl.SetBasePath(rl.Conf.BasePath)
	rl.log = logging.MustGetLogger("reader.http")

	return rl, nil
}

func (re *ApiLoop) Config(conf ApiConfig, resolution float64) (err error) {
	if conf.Logfile == "" {
		conf.Logfile = "stdout"
	}
	re.Conf = conf

	re.Metrics, err = conf.GetMetrics(resolution)
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

	// readcache
	mx_ram := metrics.READ_CACHER_TOTAL_RAM
	mx_stats := metrics.READ_CACHER_MAX_SERIES_BYTES
	if conf.MaxReadCacheBytes > 0 {
		mx_ram = conf.MaxReadCacheBytes
	}
	if conf.MaxReadCacheBytesPerMetric > 0 {
		mx_stats = conf.MaxReadCacheBytesPerMetric
	}

	re.ReadCache = metrics.InitReadCache(mx_ram, mx_stats, metrics.READ_CACHER_MAX_LAST_ACCESS)

	return nil
}

func (re *ApiLoop) Stop() {
	shutdown.AddToShutdown()
	defer shutdown.ReleaseFromShutdown()

	if re.shutdown != nil {
		re.shutdown <- true
	}
}

func (re *ApiLoop) activateCacheLoop() {
	for {
		select {
		case data, more := <-re.activate_cache_chan:
			if !more {
				return
			}
			if data == nil {
				continue
			}
			re.ReadCache.ActivateMetricFromRenderData(data)
		}
	}
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
	w.Header().Set("Content-Type", "text/plain")
	http.Error(w, msg, code)
	re.log.Error(msg)
}

func (re *ApiLoop) OutJson(w http.ResponseWriter, data interface{}) {
	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.http.ok", 1)
	w.Header().Set("Cache-Control", "public, max-age=60, cache")
	w.Header().Set("Content-Type", "application/json")

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

func (re *ApiLoop) parseForRender(w http.ResponseWriter, r *http.Request) (string, int64, int64, repr.SortingTags, error) {
	r.ParseForm()
	var target string
	var from string
	var to string
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

	from = strings.TrimSpace(r.Form.Get("from"))
	to = strings.TrimSpace(r.Form.Get("to"))

	if len(target) == 0 {
		return "", 0, 0, tags, errTargetRequired

	}

	if len(from) == 0 {
		from = "-1h"
	}

	if len(to) == 0 {
		to = "now"
	}

	start, err := metrics.ParseTime(from)
	if err != nil {
		return "", 0, 0, tags, fmt.Errorf("Invalid from time `%s` :: %v", from, err)
	}

	end, err := metrics.ParseTime(to)
	if err != nil {
		return "", 0, 0, tags, fmt.Errorf("Invalid from time `%s` :: %v", from, err)
	}
	if end < start {
		start, end = end, start
	}

	return target, start, end, tags, nil
}

// take a rawrender and make it a graphite api json format
func (re *ApiLoop) ToGraphiteRender(raw_data []*metrics.RawRenderItem) *metrics.WhisperRenderItem {
	whis := new(metrics.WhisperRenderItem)
	whis.Series = make(map[string][]metrics.DataPoint)
	if raw_data == nil {
		return nil
	}
	for _, data := range raw_data {
		if data == nil {
			continue
		}
		d_points := make([]metrics.DataPoint, data.Len(), data.Len())
		whis.End = data.End
		whis.Start = data.Start
		whis.Step = data.Step
		whis.RealEnd = data.RealEnd
		whis.RealStart = data.RealStart

		for idx, d := range data.Data {
			v := d.AggValue(data.AggFunc)
			d_points[idx] = metrics.DataPoint{Time: d.Time, Value: &v}
		}
		whis.Series[data.Metric] = d_points
	}
	return whis
}

func (re *ApiLoop) Render(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.render.get-time-ns", time.Now())

	target, from, to, _, err := re.parseForRender(w, r)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.RawRender(target, from, to)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if data == nil {
		re.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	render_data := re.ToGraphiteRender(data)

	for _, d := range data {
		// send to activator
		re.activate_cache_chan <- d
	}

	re.OutJson(w, render_data)
	return
}

func (re *ApiLoop) RawRender(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.render.get-time-ns", time.Now())

	target, from, to, _, err := re.parseForRender(w, r)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.RawRender(target, from, to)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if data == nil {
		re.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	// send to activator
	for _, d := range data {
		re.activate_cache_chan <- d
	}

	re.OutJson(w, data)
	return
}

// only return data that's in the write-back caches
// this is handy if you just want to poll what's currently being aggregated
// especially useful for the "series" style where writes to a DB system are much
// less then flat based mechanisms
func (re *ApiLoop) GetFromCache(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.cache-render.get-time-ns", time.Now())

	target, from, to, tags, err := re.parseForRender(w, r)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.CacheRender(target, from, to, tags)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if data == nil {
		re.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	re.OutJson(w, data)
	return
}

// this will return the Raw BINARY series for metrics in the write-caches
// Note that ONLY ONE metric can be queries in this fashion as there is
// multi-series binary format ... yet
func (re *ApiLoop) GetSeriesFromCache(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdNanoTimeFunc("reader.http.cache-render.get-time-ns", time.Now())

	target, from, to, tags, err := re.parseForRender(w, r)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}

	data, err := re.Metrics.CachedSeries(target, from, to, tags)
	if err != nil {
		re.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if data == nil {
		re.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	// see if we want to base64 the beast
	to_base_64 := r.Form.Get("base64")

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.http.ok", 1)

	w.Header().Set("Cache-Control", "public, max-age=60, cache")
	w.Header().Set("Content-Type", "application/cadent")
	w.Header().Set("X-CadentSeries-Key", data.Name.Key)
	w.Header().Set("X-CadentSeries-UniqueId", data.Name.UniqueIdString())

	t_str, _ := json.Marshal(data.Name.SortedTags())
	w.Header().Set("X-CadentSeries-Tags", string(t_str))
	t_str, _ = json.Marshal(data.Name.SortedMetaTags())
	w.Header().Set("X-CadentSeries-MetaTags", string(t_str))

	w.Header().Set("X-CadentSeries-Resolution", fmt.Sprintf("%d", data.Name.Resolution))
	w.Header().Set("X-CadentSeries-TTL", fmt.Sprintf("%d", data.Name.TTL))
	w.Header().Set("X-CadentSeries-Encoding", data.Series.Name())
	w.Header().Set("X-CadentSeries-Start", fmt.Sprintf("%d", data.Series.StartTime()))
	w.Header().Set("X-CadentSeries-End", fmt.Sprintf("%d", data.Series.LastTime()))
	w.Header().Set("X-CadentSeries-Points", fmt.Sprintf("%d", data.Series.Count()))

	switch to_base_64 {
	case "1":
		w.Header().Set("Content-Transfer-Encoding", "base64")
		b_encoder := base64.NewEncoder(base64.RawStdEncoding, w)
		b_encoder.Write(data.Series.Bytes())
		b_encoder.Close()
	default:
		w.Header().Set("Content-Type", "application/cadent")

		w.Write(data.Series.Bytes())

	}

	return
}

func (re *ApiLoop) NoOp(w http.ResponseWriter, r *http.Request) {
	golog.Printf("No handler for this URL %s", r.URL)
	http.Error(w,
		fmt.Sprintf("Nothing here .. try %s/find or %s/render or %s/expand or %s/cache", re.Conf.BasePath, re.Conf.BasePath, re.Conf.BasePath),
		http.StatusNotFound,
	)
	return
}

func (re *ApiLoop) corsHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS")

		if r.Method == "OPTIONS" {
			// nothing to do, CORS headers already sent
			return
		}
		handler(w, r)
	}
}

func (re *ApiLoop) Start() error {
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

	mux.HandleFunc(re.Conf.BasePath+"rawrender/", re.RawRender)
	mux.HandleFunc(re.Conf.BasePath+"rawrender", re.RawRender)

	mux.HandleFunc(re.Conf.BasePath+"cached/series/", re.GetSeriesFromCache)
	mux.HandleFunc(re.Conf.BasePath+"cached/series", re.GetSeriesFromCache)

	mux.HandleFunc(re.Conf.BasePath+"cache/", re.GetFromCache)
	mux.HandleFunc(re.Conf.BasePath+"cache", re.GetFromCache)

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

	tcpAddr, err := net.ResolveTCPAddr("tcp", re.Conf.Listen)
	if err != nil {
		return fmt.Errorf("Error resolving: %s", err)
	}

	conn, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("Could not make http socket: %s", err)
	}

	re.activate_cache_chan = make(chan *metrics.RawRenderItem, 256)

	// start up the activateCacheLoop
	go re.activateCacheLoop()

	go http.Serve(conn, re.corsHandler(WriteLog(mux, outlog)))

	re.shutdown = make(chan bool, 5)
	re.started = true

	for {
		select {
		case _, more := <-re.shutdown:
			// already done
			if !more {
				return nil
			}
			conn.Close()
			golog.Print("Shutdown of API http server...")
			close(re.activate_cache_chan)
			return nil
		}
	}
}
