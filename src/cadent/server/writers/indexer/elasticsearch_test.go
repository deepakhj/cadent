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

package indexer

// you best have some form of cassandra up

import (
	//. "github.com/smartystreets/goconvey/convey"

	"cadent/server/repr"
	"cadent/server/test/helper"
	"cadent/server/utils/options"
	"fmt"
	"testing"
)

var es_port int = 9200

var _es *ElasticIndexer

func getEsIndexer() (*ElasticIndexer, error) {
	if _es != nil {
		return _es, nil
	}
	on_ip := helper.DockerIp()

	config_opts := options.Options{}
	config_opts.Set("dsn", fmt.Sprintf("http://%s:%d", on_ip, es_port))
	config_opts.Set("metric_index", "test_mets")
	config_opts.Set("path_index", "test_path")
	config_opts.Set("segment_index", "test_seg")
	config_opts.Set("tag_index", "test_tag")
	config_opts.Set("sniff", false)
	config_opts.Set("enable_traceing", true)

	_es := NewElasticIndexer()
	err := _es.Config(config_opts)
	if err != nil {
		return nil, err
	}

	return _es, nil
}

func getDocker(t *testing.T) {
	// fire up the docker test helper
	helper.DockerUp("elasticsearch")
	ok := helper.DockerWaitUntilReady("elasticsearch")
	if !ok {
		t.Fatalf("Could not start the docker container for elasticserch")
	}

}

func TestElasticIndexer_New(t *testing.T) {

	getDocker(t)
	_, err := getEsIndexer()
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func TestElasticIndexer_Start(t *testing.T) {

	getDocker(t)

	es, _ := getEsIndexer()
	es.Start()

}

func writeSome(es *ElasticIndexer) error {

	nms := []string{
		"test.this.thing",
		"test.my.house",
		"monkey.lives.here.before",
		"monkey.my.house",
	}

	tgs := repr.SortingTagsFromString("moo=goo,loo=baz")

	for _, n := range nms {
		nm := repr.StatName{Key: n, Tags: tgs}
		err := es.WriteOne(&nm)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestElasticIndexer_WriteOne(t *testing.T) {

	getDocker(t)

	es, err := getEsIndexer()
	if err != nil {
		t.Fatalf("%v", err)
	}
	es.Start()

	err = writeSome(es)
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func TestElasticIndexer_FindRoot(t *testing.T) {

	getDocker(t)

	es, err := getEsIndexer()
	if err != nil {
		t.Fatalf("%v", err)
	}
	es.Start()

	err = writeSome(es)

	res, err := es.FindRoot(repr.SortingTags{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	for _, item := range res {
		t.Logf("Item: %s (%s)", item.UniqueId, item.Path)

		if item.Path != "test" && item.Path != "monkey" {
			t.Fatalf("Bad found item")
		}
	}
}

func TestElasticIndexer_Find(t *testing.T) {

	getDocker(t)

	es, err := getEsIndexer()
	if err != nil {
		t.Fatalf("%v", err)
	}
	es.Start()

	err = writeSome(es)

	res, err := es.Find("test.*", repr.SortingTags{})
	if err != nil {
		t.Fatalf("%v", err)
	}

	for _, item := range res {
		t.Logf("Item: %s (%s)", item.UniqueId, item.Path)

		if item.Path != "test.this" && item.Path != "test.my" {
			t.Fatalf("Bad found item")
		}
	}

	res, err = es.Find("test.*.*", repr.SortingTags{})
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(res) != 2 {
		t.Fatalf("Too many items found")
	}

	for _, item := range res {
		t.Logf("Item: %s (%s)", item.UniqueId, item.Path)
		if item.Path != "test.this.thing" && item.Path != "test.my.house" {
			t.Fatalf("Bad found item")
		}
		if item.AllowChildren == 1 {
			t.Fatalf("Item should have allow childer to false")
		}
	}

	tgs := repr.SortingTagsFromString("moo=goo,loo=baz")
	res, err = es.Find("test.*.*", tgs)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(res) != 2 {
		t.Fatalf("Too many items found")
	}

	for _, item := range res {
		t.Logf("Item: %s (%s)", item.UniqueId, item.Path)

		if item.AllowChildren == 1 {
			t.Fatalf("Item should have allow childer to false")
		}
	}
}

func TestElasticIndexer_GetTagsByName(t *testing.T) {

	getDocker(t)

	es, err := getEsIndexer()
	if err != nil {
		t.Fatalf("%v", err)
	}
	es.Start()

	err = writeSome(es)

	res, err := es.GetTagsByName("moo", 0)
	if err != nil {
		t.Fatalf("%v", err)
	}

	for _, item := range res {
		t.Logf("Item: %s (%s)", item.Name, item.Value)
		if item.Name != "moo" {
			t.Fatalf("Bad found item")
		}
	}

	res, err = es.GetTagsByName("m{o}*", 0)
	if err != nil {
		t.Fatalf("%v", err)
	}

	for _, item := range res {
		t.Logf("Item: %s (%s)", item.Name, item.Value)
		if item.Name != "moo" {
			t.Fatalf("Bad found item")
		}
	}
}
