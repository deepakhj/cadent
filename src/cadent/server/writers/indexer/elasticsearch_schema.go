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
	Create the Elastic search index + schema bits

	path index
	{
		uid: [string]
		path: [string]
		tags:[
			{
				name: [string],
				value: [string],
				is_meta: [boolean],
			}, ...
		]
	}

	tag index
	{
		name: [string],
		value: [string],
		is_meta: [boolean]
	}
*/

package indexer

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/repr"
	"cadent/server/utils"
	"gopkg.in/olivere/elastic.v3"
)

var ELASTIC_PATH_TYPE string = `{
    "dynamic_templates": [{
        	"notanalyze": {
         		"mapping": {
            			"index": "not_analyzed"
         		},
          		"match_mapping_type": "string",
          		"match": "*"
       		}
   }],
   "_all": {
	"enabled": false
   },
   "properties":{
	"uid":{
            "type": "string",
            "index": "not_analyzed"
        },
        "segment":{
            "type": "string",
            "index": "not_analyzed"
        },
        "path":{
            "type": "string",
            "index": "not_analyzed"
        },
        "pos": {
            "type": "integer",
            "index": "not_analyzed"
        },
        "length": {
            "type": "integer",
            "index": "not_analyzed"
        },
        "has_data": {
            "type": "boolean",
            "index":  "not_analyzed"
        },
        "tags":{
            "type": "nested",
            "properties":{
                "name": {
                    "type": "string",
                    "index": "not_analyzed"
                },
                "value": {
                    "type": "string",
                    "index": "not_analyzed"
                },
                "is_meta":{
                    "type":"boolean",
                    "index": "not_analyzed"
                }
            }
        }
    }
}
`

var ELASTIC_SEGMENT_TYPE string = `{
    "dynamic_templates": [{
        	"notanalyze": {
         		"mapping": {
            			"index": "not_analyzed"
         		},
          		"match_mapping_type": "string",
          		"match": "*"
       		}
   }],
   "_all": {
   	"enabled": false
   },
   "properties":{
	"segment":{
            "type": "string",
            "index": "not_analyzed"
        },
        "pos": {
            "type": "integer",
            "index": "not_analyzed"
        }
    }
}
`

var ELASTIC_TAG_TYPE string = `{
    "dynamic_templates": [{
        	"notanalyze": {

         		"mapping": {
            			"index": "not_analyzed"
         		},
          		"match_mapping_type": "string",
          		"match": "*"
       		}
   }],
   "_all": {
	"enabled": false
   },
   "properties": {
   	 "name":{
            "type": "string",
            "index": "not_analyzed"
        },
        "value": {
            "type": "string",
            "index":  "not_analyzed"
        },
        "is_meta": {
            "type": "boolean",
            "index": "not_analyzed"
        }
    }
}
`

type ESTag struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	IsMeta bool   `json:"is_meta"`
}

type ESSegment struct {
	Segment string `json:"segment"`
	Pos     int    `json:"pos"`
}

type ESPath struct {
	Uid     string  `json:"uid,omitempty"`
	Segment string  `json:"segment"`
	Path    string  `json:"path"`
	Pos     int     `json:"pos"`
	Length  int     `json:"length"`
	HasData bool    `json:"has_data"`
	Tags    []ESTag `json:"tags,omitempty"`
}

func (et *ESPath) ToSortedTags() (tags repr.SortingTags, metatags repr.SortingTags) {
	for _, tg := range et.Tags {
		switch tg.IsMeta {
		case true:
			metatags = metatags.Set(tg.Name, tg.Value)
		default:
			tags = tags.Set(tg.Name, tg.Value)
		}
	}
	return
}

type ElasticSearchSchema struct {
	conn         *elastic.Client
	pathIndex    string
	tagIndex     string
	segmentIndex string
	log          *logging.Logger
	startstop    utils.StartStop
}

func NewElasticSearchSchema(conn *elastic.Client, segment string, path string, tag string) *ElasticSearchSchema {
	my := new(ElasticSearchSchema)
	my.conn = conn
	my.pathIndex = path
	my.tagIndex = tag
	my.segmentIndex = segment

	my.log = logging.MustGetLogger("writers.elastic.index.schema")
	return my
}

func (es *ElasticSearchSchema) AddIndexTables() (err error) {
	es.startstop.Start(func() {
		es.log.Notice("Adding ElasticSearch index indexes `%s` `%s` `%s`", es.pathIndex, es.tagIndex, es.segmentIndex)
		indexes := []string{es.pathIndex, es.tagIndex, es.segmentIndex}
		for _, q := range indexes {

			// Use the IndexExists service to check if a specified index exists.
			exists, terr := es.conn.IndexExists(q).Do()
			if terr != nil {
				err = terr
				return
			}
			if !exists {
				_, err = es.conn.CreateIndex(q).Do()
				if err != nil {
					return
				}
			}
			if err != nil {
				es.log.Errorf("ElasticSearch Schema Driver: Indexer index failed, %v", err)
				break
			}
		}

		// see if it's there already
		got, terr := es.conn.GetMapping().Index(es.pathIndex).Type("path").Do()
		if terr != nil {
			err = terr
			return
		}
		es.log.Notice("ElasticSearch Schema Driver: have path mapping %v", err)
		var putresp *elastic.PutMappingResponse
		if len(got) == 0 || err != nil {
			putresp, err = es.conn.PutMapping().Index(es.pathIndex).Type("path").BodyString(ELASTIC_PATH_TYPE).Do()
			if err != nil {
				es.log.Errorf("ElasticSearch Schema Driver: Indexer index failed, %v", err)
				return
			}
			if putresp == nil {
				es.log.Errorf("ElasticSearch Schema Driver: Indexer index failed, no response")
				err = fmt.Errorf("ElasticSearch Schema Driver: Indexer index failed, no response")
				return
			}
			if !putresp.Acknowledged {
				err = fmt.Errorf("expected put mapping ack; got: %v", putresp.Acknowledged)
				return
			}
		}
		got, err = es.conn.GetMapping().Index(es.tagIndex).Type("tag").Do()
		if err != nil {
			return
		}
		es.log.Notice("ElasticSearch Schema Driver: have tag mapping, %v", err)
		if len(got) == 0 || err != nil {
			putresp, err = es.conn.PutMapping().Index(es.tagIndex).Type("tag").BodyString(ELASTIC_TAG_TYPE).Do()
			if err != nil {
				es.log.Errorf("ElasticSearch Schema Driver: Indexer index failed, %v", err)
				return
			}
			if putresp == nil {
				es.log.Errorf("ElasticSearch Schema Driver: Indexer index failed, no response")
				err = fmt.Errorf("ElasticSearch Schema Driver: Indexer index failed, no response")
				return
			}
			if !putresp.Acknowledged {
				err = fmt.Errorf("expected put mapping ack; got: %v", putresp.Acknowledged)
				return
			}
		}

		got, err = es.conn.GetMapping().Index(es.segmentIndex).Type("segment").Do()
		if err != nil {
			return
		}
		es.log.Notice("ElasticSearch Schema Driver: have segment mapping %v", err)
		if len(got) == 0 || err != nil {
			putresp, err = es.conn.PutMapping().Index(es.segmentIndex).Type("segment").BodyString(ELASTIC_SEGMENT_TYPE).Do()
			if err != nil {
				es.log.Errorf("ElasticSearch Schema Driver: Indexer index failed, %v", err)
				return
			}
			if putresp == nil {
				es.log.Errorf("ElasticSearch Schema Driver: Indexer index failed, no response")
				err = fmt.Errorf("ElasticSearch Schema Driver: Indexer index failed, no response")
				return
			}
			if !putresp.Acknowledged {
				err = fmt.Errorf("expected put mapping ack; got: %v", putresp.Acknowledged)
				return
			}
		}
	})
	return err
}
