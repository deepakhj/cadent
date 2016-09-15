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
	Driver   string          `toml:"driver"`
	DSN      string          `toml:"dsn"`
	UseCache string          `toml:"cache"`
	Options  options.Options `toml:"options"` // option=[ [key, value], [key, value] ...]
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

	// need to match caches
	// use the defined cacher object
	if len(re.ApiMetricOptions.UseCache) == 0 {
		return nil, writers.ErrCacheOptionRequired
	}

	// find the proper cache to use
	res := uint32(resolution)
	proper_name := fmt.Sprintf("%s:%d", re.ApiMetricOptions.UseCache, res)
	c, err := metrics.GetCacherSingleton(proper_name)
	if err != nil {
		return nil, err
	}
	re.ApiMetricOptions.Options["cache"] = c

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
		re.ApiIndexerOptions.Options = options.New()
	}
	re.ApiIndexerOptions.Options["dsn"] = re.ApiIndexerOptions.DSN
	err = idx.Config(re.ApiIndexerOptions.Options)
	if err != nil {
		return nil, err
	}
	return idx, nil
}
