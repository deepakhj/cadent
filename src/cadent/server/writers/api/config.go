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
	"cadent/server/utils/options"
	"cadent/server/writers"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
)

type ApiMetricConfig struct {
	Driver   string          `toml:"driver" json:"driver,omitempty"`
	DSN      string          `toml:"dsn"  json:"dsn,omitempty"`
	UseCache string          `toml:"cache"  json:"cache,omitempty"`
	Options  options.Options `toml:"options"  json:"options,omitempty"`
}

type ApiIndexerConfig struct {
	Driver  string          `toml:"driver"  json:"driver,omitempty"`
	DSN     string          `toml:"dsn"  json:"dsn,omitempty"`
	Options options.Options `toml:"options"  json:"options,omitempty"`
}

// ApiConfig each writer for both metrics and indexers should get the http API attached to it
type ApiConfig struct {
	Listen                     string           `toml:"listen"  json:"listen,omitempty"`
	GPRCListen                 string           `toml:"grpc_listen" json:"grpc_listen,omitempty"`
	Logfile                    string           `toml:"log_file"  json:"log-file,omitempty"`
	BasePath                   string           `toml:"base_path"  json:"base-path,omitempty"`
	TLSKeyPath                 string           `toml:"key"  json:"key,omitempty"`
	TLSCertPath                string           `toml:"cert"  json:"cert,omitempty"`
	ApiMetricOptions           ApiMetricConfig  `toml:"metrics"  json:"metrics,omitempty"`
	ApiIndexerOptions          ApiIndexerConfig `toml:"indexer"  json:"indexer,omitempty"`
	MaxReadCacheBytes          int              `toml:"read_cache_total_bytes"  json:"read-cache-total-bytes,omitempty"`
	MaxReadCacheBytesPerMetric int              `toml:"read_cache_max_bytes_per_metric"  json:"read-cache-max-bytes-per-metric,omitempty"`
}

// SoloApiConfig for when you just want to run the API only (i.e. a reader only node)
type SoloApiConfig struct {
	ApiConfig
	Seed        string  `toml:"seed" json:"seed,omitempty"` // only used in "api only" mode
	Resolutions []int64 `toml:"resolutions" json:"resolutions,omitempty"`
}

func (s *SoloApiConfig) GetApiConfig() ApiConfig {
	return ApiConfig{
		Listen:                     s.Listen,
		GPRCListen:                 s.GPRCListen,
		Logfile:                    s.Logfile,
		BasePath:                   s.BasePath,
		TLSKeyPath:                 s.TLSKeyPath,
		TLSCertPath:                s.TLSCertPath,
		ApiMetricOptions:           s.ApiMetricOptions,
		ApiIndexerOptions:          s.ApiIndexerOptions,
		MaxReadCacheBytes:          s.MaxReadCacheBytes,
		MaxReadCacheBytesPerMetric: s.MaxReadCacheBytesPerMetric,
	}
}

// GetMetrics creates a new metrics object
func (re *ApiConfig) GetMetrics(resolution float64) (metrics.Metrics, error) {
	reader, err := metrics.NewWriterMetrics(re.ApiMetricOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.ApiMetricOptions.Options == nil {
		re.ApiMetricOptions.Options = options.New()
	}
	re.ApiMetricOptions.Options.Set("dsn", re.ApiMetricOptions.DSN)
	re.ApiMetricOptions.Options.Set("resolution", resolution)

	// need to match caches
	// use the defined cacher object
	if len(re.ApiMetricOptions.UseCache) == 0 {
		return nil, writers.ErrCacheOptionRequired
	}

	// find the proper cache to use
	res := uint32(resolution)
	proper_name := fmt.Sprintf("%s:%d", re.ApiMetricOptions.UseCache, res)
	c, err := metrics.GetCacherSingleton(proper_name, metrics.CacheNeededName(re.ApiMetricOptions.Driver))
	if err != nil {
		return nil, err
	}
	re.ApiMetricOptions.Options.Set("cache", c)

	err = reader.Config(&re.ApiMetricOptions.Options)

	if err != nil {
		return nil, err
	}
	return reader, nil
}

// GetIndexer creates a new indexer object
func (re *ApiConfig) GetIndexer() (indexer.Indexer, error) {
	idx, err := indexer.NewIndexer(re.ApiIndexerOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.ApiIndexerOptions.Options == nil {
		re.ApiIndexerOptions.Options = options.New()
	}
	re.ApiIndexerOptions.Options.Set("dsn", re.ApiIndexerOptions.DSN)
	err = idx.Config(&re.ApiIndexerOptions.Options)
	if err != nil {
		return nil, err
	}
	return idx, nil
}
