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
	THe ElasticSearch write

*/

package dbs

import (
	"cadent/server/utils/options"
	"fmt"
	"gopkg.in/olivere/elastic.v3"
	logging "gopkg.in/op/go-logging.v1"

	"strings"
)

type ElasticSearchLog struct{}

func (el *ElasticSearchLog) Printf(str string, args ...interface{}) {
	fmt.Printf(str, args...)
}

/****************** Interfaces *********************/
type ElasticSearch struct {
	Client       *elastic.Client
	metricIndex  string
	pathIndex    string
	segmentIndex string
	tagIndex     string

	PathType    string
	TagType     string
	SegmentType string
	MetricType  string
	log         *logging.Logger
}

func NewElasticSearch() *ElasticSearch {
	my := new(ElasticSearch)
	my.log = logging.MustGetLogger("writers.elastic")
	my.PathType = "path"
	my.TagType = "tag"
	my.SegmentType = "segment"
	my.MetricType = "metric"

	return my
}

func (my *ElasticSearch) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` http://host:port,http://host:port is needed for elatic search config")
	}

	ops_funcs := make([]elastic.ClientOptionFunc, 0)
	ops_funcs = append(ops_funcs, elastic.SetURL(strings.Split(dsn, ",")...))
	ops_funcs = append(ops_funcs, elastic.SetMaxRetries(10))
	ops_funcs = append(ops_funcs, elastic.SetSniff(conf.Bool("sniff", false)))

	user := conf.String("user", "")
	pass := conf.String("password", "")
	if user != "" {
		ops_funcs = append(ops_funcs, elastic.SetBasicAuth(user, pass))
	}

	if conf.Bool("enable_traceing", false) {
		ops_funcs = append(ops_funcs, elastic.SetTraceLog(new(ElasticSearchLog)))
	}

	my.Client, err = elastic.NewClient(ops_funcs...)

	if err != nil {
		if conf.Bool("sniff", true) {
			my.log.Errorf("You may wish to turn off sniffing `sniff=false` if the nodes are behind a loadbalencer")
		}
		return err
	}
	my.metricIndex = conf.String("metric_index", "metrics")
	my.pathIndex = conf.String("path_index", "metric_path")
	my.segmentIndex = conf.String("segment_index", "metric_segment")
	my.tagIndex = conf.String("tag_index", "metric_tag")

	return nil
}

func (my *ElasticSearch) Tablename() string {
	return my.metricIndex
}

func (my *ElasticSearch) PathTable() string {
	return my.pathIndex
}

func (my *ElasticSearch) SegmentTable() string {
	return my.segmentIndex
}

func (my *ElasticSearch) TagTable() string {
	return my.tagIndex
}

func (my *ElasticSearch) Connection() DBConn {
	return my.Client
}
