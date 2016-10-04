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
	Client        *elastic.Client
	metric_index  string
	path_index    string
	segment_index string
	tag_index     string

	PathType    string
	TagType     string
	SegmentType string
	log         *logging.Logger
}

func NewElasticSearch() *ElasticSearch {
	my := new(ElasticSearch)
	my.log = logging.MustGetLogger("writers.elastic")
	my.PathType = "path"
	my.TagType = "tag"
	my.SegmentType = "segment"

	return my
}

func (my *ElasticSearch) Config(conf options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` http://host:port,http://host:port is needed for elatic search config")
	}

	ops_funcs := make([]elastic.ClientOptionFunc, 0)
	ops_funcs = append(ops_funcs, elastic.SetURL(strings.Split(dsn, ",")...))
	ops_funcs = append(ops_funcs, elastic.SetMaxRetries(10))
	ops_funcs = append(ops_funcs, elastic.SetSniff(conf.Bool("sniff", true)))

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
	my.metric_index = conf.String("metric_index", "metrics")
	my.path_index = conf.String("path_index", "metric_path")
	my.segment_index = conf.String("segment_index", "metric_segment")
	my.tag_index = conf.String("tag_index", "metric_tag")

	return nil
}

func (my *ElasticSearch) Tablename() string {
	return my.metric_index
}

func (my *ElasticSearch) PathTable() string {
	return my.path_index
}

func (my *ElasticSearch) SegmentTable() string {
	return my.segment_index
}

func (my *ElasticSearch) TagTable() string {
	return my.tag_index
}

func (my *ElasticSearch) Connection() DBConn {
	return my.Client
}
