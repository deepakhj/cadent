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
	Elastic search types

	Prefixes are `_{resolution}s` (i.e. "_" + (uint32 resolution) + "s")


*/

package metrics

import (
	"fmt"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/utils"
	"cadent/server/writers/indexer"
	"gopkg.in/olivere/elastic.v3"
	"strings"
	"time"
)

const ELASTIC_METRICS_TEMPLATE = `
{
    "template":   "%s*",
    "mappings": {
        "_default_": {
            "_all": {
                "enabled": false
            },
            "dynamic_templates": [{
                "string_fields": {
                    "match": "*",
                    "match_mapping_type": "string",
                    "mapping": {
                        "index": "not_analyzed",
                        "omit_norms": true,
                        "type": "string"
                    }
                }
            }]
        }
    }
}`

const ELASTIC_METRICS_FLAT_TABLE = `
{
   "dynamic_templates": [{
        	"notanalyze": {
         		"mapping": {
            			"index": "not_analyzed",
            			"omit_norms": true
         		},
          		"match_mapping_type": "*",
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
        "path":{
            "type": "string",
            "index": "not_analyzed"
        },
        "time":{
            "type": "date",
            "index": "not_analyzed",
            "format": "strict_date_optional_time||epoch_millis"
        },
        "min":{
            "type": "double",
            "index": "not_analyzed"
        },
        "max":{
            "type": "double",
            "index": "not_analyzed"
        },
        "sum":{
            "type": "double",
            "index": "not_analyzed"
        },
        "last":{
            "type": "double",
            "index": "not_analyzed"
        },
        "count":{
            "type": "long",
            "index": "not_analyzed"
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
}`

const ELASTIC_METRICS_BLOB_TABLE = `
{
   "dynamic_templates": [{
        	"notanalyze": {
         		"mapping": {
            			"index": "not_analyzed",
            			"omit_norms": true
         		},
          		"match_mapping_type": "*",
          		"match": "*"
       		}
   }],
   "_all": {
	"enabled": false
   },
   "_source": {
	"enabled": false
   },

   "properties":{
        "uid":{
            "type": "string",
            "index": "not_analyzed"
        },
        "path":{
            "type": "string",
            "index": "not_analyzed"
        },
        "ptype":{
        	"type": "string",
            	"index": "not_analyzed"
        },
        "stime":{
            "type": "long",
            "index": "not_analyzed",
            "format": "strict_date_optional_time||epoch_millis"
        },
        "etime":{
            "type": "long",
            "index": "not_analyzed",
            "format": "strict_date_optional_time||epoch_millis"
        },
        "points":{
            "type": "binary",
            "index": "not_analyzed"
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
}`

type ESMetric struct {
	Uid   string          `json:"uid"`
	Path  string          `json:"path"`
	Time  time.Time       `json:"time"`
	Min   float64         `json:"min"`
	Max   float64         `json:"max"`
	Sum   float64         `json:"sum"`
	Last  float64         `json:"last"`
	Count int64           `json:"count"`
	Tags  []indexer.ESTag `json:"tags,omitempty"`
}

type ESBlobMetric struct {
	Uid    string          `json:"uid"`
	Path   string          `json:"path"`
	Stime  time.Time       `json:"stime"`
	Etime  time.Time       `json:"etime"`
	Ptype  string          `json:"ptype"`
	Points []byte          `json:"points"`
	Tags   []indexer.ESTag `json:"tags,omitempty"`
}

/****************** Interfaces *********************/
type ElasticMetricsSchema struct {
	conn        *elastic.Client
	tableBase   string
	resolutions [][]int
	mode        string
	log         *logging.Logger
	startstop   utils.StartStop
}

func NewElasticMetricsSchema(conn *elastic.Client, metricTable string, resolutions [][]int, mode string) *ElasticMetricsSchema {
	es := new(ElasticMetricsSchema)
	es.conn = conn
	es.tableBase = metricTable
	es.resolutions = resolutions
	es.mode = mode
	es.log = logging.MustGetLogger("writers.elastic.metric.schema")
	return es
}

func (es *ElasticMetricsSchema) AddMetricsTable() (err error) {
	es.startstop.Start(func() {

		if len(es.resolutions) == 0 {
			err = fmt.Errorf("Need resolutions")
			return
		}

		// do the main template
		tplQ := fmt.Sprintf(ELASTIC_METRICS_TEMPLATE, es.tableBase)
		es.log.Notice("Adding default metrics index template...")

		gTpl, terr := es.conn.IndexPutTemplate("metrics-template").BodyString(tplQ).Do()
		if terr != nil {
			err = terr
			return
		}
		if gTpl == nil {
			err = fmt.Errorf("ElasticSearch Schema Driver: Metric index failed, %v", gTpl)
			es.log.Errorf("%v", err)
			return
		}

		// resolutions are [resolution][ttl]
		es.log.Notice("Adding default elasticsearch schemas for resolutions %v ...", es.resolutions)
		for _, res := range es.resolutions {
			tname := fmt.Sprintf("%s_%ds", es.tableBase, res[0])
			es.log.Notice("Adding elastic metric indexes `%s`", tname)

			// Use the IndexExists service to check if a specified index exists.
			exists, terr := es.conn.IndexExists(tname).Do()
			if terr != nil {
				err = terr
				return
			}
			if !exists {
				_, err = es.conn.CreateIndex(tname).Do()
				// skip the "already made" errors
				if err != nil {
					if !strings.Contains(err.Error(), "index_already_exists_exception") {
						es.log.Errorf("ElasticSearch Schema Driver: Metric index failed, %v", err)
						return
					}
					err = nil
				}
			}

			var Q string
			if es.mode == "flat" {
				Q = fmt.Sprintf(ELASTIC_METRICS_FLAT_TABLE, tname)
			} else {
				Q = fmt.Sprintf(ELASTIC_METRICS_BLOB_TABLE, tname)

			}
			// see if it's there already
			got, terr := es.conn.GetMapping().Index(tname).Type("metric").Do()
			if terr != nil {
				err = terr
				return
			}
			es.log.Notice("ElasticSearch Schema Driver: have metric mapping %v", err)
			var putresp *elastic.PutMappingResponse
			if len(got) == 0 || err != nil {
				putresp, err = es.conn.PutMapping().Index(tname).Type("metric").BodyString(Q).Do()
				if err != nil {
					es.log.Errorf("ElasticSearch Schema Driver: Metric index failed, %v", err)
					return
				}
				if putresp == nil {
					es.log.Errorf("ElasticSearch Schema Driver: Metric index failed, no response")
					err = fmt.Errorf("ElasticSearch Schema Driver: Metric index failed, no response")
					return
				}
				if !putresp.Acknowledged {
					err = fmt.Errorf("expected put mapping ack; got: %v", putresp.Acknowledged)
					return
				}
			}
		}
	})
	return err
}
