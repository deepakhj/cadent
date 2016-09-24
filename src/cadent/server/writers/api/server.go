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

package api

import (
	"cadent/server/stats"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/gorilla/mux"
	logging "gopkg.in/op/go-logging.v1"
	golog "log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
)

const MAX_METRIC_POINTS uint32 = 20480
const DEFAULT_MIN_RESOLUTION uint32 = 1

type ApiLoop struct {
	Conf    ApiConfig
	Metrics metrics.Metrics
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

func (re *ApiLoop) AddToCache(data []*metrics.RawRenderItem) {
	if re.activate_cache_chan == nil {
		return
	}
	for _, d := range data {
		// send to activator
		re.activate_cache_chan <- d
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

func (re *ApiLoop) withRecover(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			msg := fmt.Sprintf("Panic/Recovered: Error: %s", err)
			re.log.Critical(msg)
			debug.PrintStack()
		}
	}()

	fn()
}

func (re *ApiLoop) OutError(w http.ResponseWriter, msg string, code int) {

	defer stats.StatsdClient.Incr("reader.http.errors", 1)
	w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
	w.Header().Set("Content-Type", "text/plain")
	http.Error(w, msg, code)
	re.log.Error(msg)
}

func (re *ApiLoop) OutJson(w http.ResponseWriter, data interface{}) {

	// trap any encoding issues here
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("Json Out Render Err: %v", r)
			re.log.Critical(msg)
			re.OutError(w, msg, http.StatusInternalServerError)
			debug.PrintStack()
			return
		}
	}()

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.http.ok", 1)
	w.Header().Set("Cache-Control", "public, max-age=60, cache")
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(data)

}

func (re *ApiLoop) NoOp(w http.ResponseWriter, r *http.Request) {
	golog.Printf("No handler for this URL %s", r.URL)
	base := re.Conf.BasePath
	http.Error(w,
		fmt.Sprintf(
			"Nothing here .. try %sfind, %srender, %sexpand, %scache, %scached/series",
			base, base, base, base, base,
		),
		http.StatusNotFound,
	)
	return
}

// based on the min resolution, figure out the real min "resample" to match the max points allowed
func (re *ApiLoop) minResolution(start int64, end int64, cur_step uint32) uint32 {

	use_min := cur_step
	if DEFAULT_MIN_RESOLUTION > use_min {
		use_min = DEFAULT_MIN_RESOLUTION
	}
	cur_pts := uint32(end-start) / use_min

	if re.Metrics == nil {
		if cur_pts > MAX_METRIC_POINTS {
			return uint32(end-start) / MAX_METRIC_POINTS
		}
		return use_min
	}
	return use_min
}

func (re *ApiLoop) Start() error {
	re.log.Notice("Starting reader http server on %s, base path: %s", re.Conf.Listen, re.Conf.BasePath)

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

	var conn net.Listener

	// certs if needed
	if len(re.Conf.TLSKeyPath) > 0 && len(re.Conf.TLSCertPath) > 0 {
		cer, err := tls.LoadX509KeyPair(re.Conf.TLSCertPath, re.Conf.TLSKeyPath)
		if err != nil {
			golog.Panicf("Could not start https server: %v", err)
			return err
		}
		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		conn, err = tls.Listen("tcp", re.Conf.Listen, config)
		if err != nil {
			return fmt.Errorf("Could not make http socket: %s", err)
		}

	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", re.Conf.Listen)
		if err != nil {
			return fmt.Errorf("Error resolving: %s", err)
		}
		conn, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return fmt.Errorf("Could not make http socket: %s", err)
		}
	}
	re.activate_cache_chan = make(chan *metrics.RawRenderItem, 256)

	// start up the activateCacheLoop
	go re.activateCacheLoop()

	r := mux.NewRouter()
	base := r.PathPrefix(re.Conf.BasePath).Subrouter()

	// the mess of handlers
	NewFindAPI(re).AddHandlers(base)
	NewCacheAPI(re).AddHandlers(base)
	NewMetricsAPI(re).AddHandlers(base)
	NewTagAPI(re).AddHandlers(base)
	NewPrometheusAPI(re).AddHandlers(base)
	NewInfoAPI(re).AddHandlers(base)

	// websocket routes (need a brand new one lest the middleware get mixed)
	ws := base.PathPrefix("/ws").Subrouter()
	NewMetricsSocket(re).AddHandlers(ws)

	mux := http.NewServeMux()
	mux.Handle("/ws", WriteLog(ws, outlog))
	mux.Handle("/", WriteLog(CompressHandler(CorsHandler(base)), outlog))

	go http.Serve(conn, mux)

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
