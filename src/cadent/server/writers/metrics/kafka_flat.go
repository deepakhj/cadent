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
	THe Kafka Metrics writer



*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	logging "gopkg.in/op/go-logging.v1"
	"time"
)

var errKafkaReaderNotImplimented = errors.New("KAFKA READER NOT IMPLMENTED")

/** kafka put object **/
type KafkaMetricObj struct {
	Type       string           `json:"type"`
	Time       int64            `json:"time"`
	Metric     string           `json:"metric"`
	Sum        repr.JsonFloat64 `json:"sum"`
	Min        repr.JsonFloat64 `json:"min"`
	Max        repr.JsonFloat64 `json:"max"`
	Count      int64            `json:"count"`
	Last       repr.JsonFloat64 `json:"last"`
	Resolution uint32           `json:"resolution"`
	Id         repr.StatId      `json:"id"`
	Uid        string           `json:"uid"`
	TTL        uint32           `json:"ttl"`
	Tags       [][]string       `json:"tags,omitempty"` // key1=value1,key2=value2...
	MetaTags   [][]string       `json:"meta_tags,omitempty"`

	encoded []byte
	err     error
}

func (kp *KafkaMetricObj) ensureEncoded() {
	if kp.encoded == nil && kp.err == nil {
		kp.encoded, kp.err = json.Marshal(kp)
	}
}

func (kp *KafkaMetricObj) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KafkaMetricObj) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

/****************** Interfaces *********************/
type KafkaFlatMetrics struct {
	db                *dbs.KafkaDB
	conn              sarama.AsyncProducer
	indexer           indexer.Indexer
	resolutions       [][]int
	currentResolution int
	static_tags       repr.SortingTags

	shutitdown bool
	startstop  utils.StartStop

	batches int // number of stats to "batch" per message (default 0)
	log     *logging.Logger
}

func NewKafkaFlatMetrics() *KafkaFlatMetrics {
	kf := new(KafkaFlatMetrics)
	kf.batches = 0
	kf.shutitdown = false
	kf.log = logging.MustGetLogger("writers.kafkaflat.metrics")
	return kf
}

func (kf *KafkaFlatMetrics) Config(conf map[string]interface{}) error {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (kafkahost1,kafkahost2) is needed for kafka config")
	}
	dsn := gots.(string)
	db, err := dbs.NewDB("kafka", dsn, conf)
	if err != nil {
		return err
	}

	kf.db = db.(*dbs.KafkaDB)
	kf.conn = db.Connection().(sarama.AsyncProducer)

	g_tag, ok := conf["tags"]
	if ok {
		kf.static_tags = repr.SortingTagsFromString(g_tag.(string))
	}
	return nil
}

func (kf *KafkaFlatMetrics) Driver() string {
	return "kafka-flat"
}

func (kf *KafkaFlatMetrics) Start() {
	//noop
}

func (kf *KafkaFlatMetrics) Stop() {
	kf.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		kf.shutitdown = true
		time.Sleep(time.Second) // wait to get things written if pending
		if err := kf.conn.Close(); err != nil {
			kf.log.Error("Failed to shut down producer cleanly %v", err)
		}
	})
}

func (kf *KafkaFlatMetrics) SetIndexer(idx indexer.Indexer) error {
	kf.indexer = idx
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (kf *KafkaFlatMetrics) SetResolutions(res [][]int) int {
	kf.resolutions = res
	return len(res) // need as many writers as bins
}

func (kf *KafkaFlatMetrics) SetCurrentResolution(res int) {
	kf.currentResolution = res
}

func (kf *KafkaFlatMetrics) Write(stat repr.StatRepr) error {

	if kf.shutitdown {
		return nil
	}

	stat.Name.MergeMetric2Tags(kf.static_tags)
	kf.indexer.Write(stat.Name) // to the indexer
	item := &KafkaMetricObj{
		Type:       "metric",
		Metric:     stat.Name.Key,
		Time:       time.Now().UnixNano(),
		Sum:        stat.Sum,
		Last:       stat.Last,
		Count:      stat.Count,
		Max:        stat.Max,
		Min:        stat.Min,
		Resolution: stat.Name.Resolution,
		TTL:        stat.Name.TTL,
		Id:         stat.Name.UniqueId(),
		Uid:        stat.Name.UniqueIdString(),
		Tags:       stat.Name.SortedTags(),
		MetaTags:   stat.Name.SortedMetaTags(),
	}

	stats.StatsdClientSlow.Incr("writer.kafkaflat.metrics.writes", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.DataTopic(),
		Key:   sarama.StringEncoder(stat.UniqueId()), // hash on unique id
		Value: item,
	}
	return nil
}

/**** READER ***/
// needed to match interface, but we obviously cannot do this

func (kf *KafkaFlatMetrics) Render(path string, from int64, to int64) (WhisperRenderItem, error) {
	return WhisperRenderItem{}, errKafkaReaderNotImplimented
}
func (kf *KafkaFlatMetrics) RawRender(path string, from int64, to int64) ([]*RawRenderItem, error) {
	return []*RawRenderItem{}, errKafkaReaderNotImplimented
}
func (kf *KafkaFlatMetrics) CacheRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error) {
	return nil, errKafkaReaderNotImplimented
}
func (kf *KafkaFlatMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*TotalTimeSeries, error) {
	return nil, errKafkaReaderNotImplimented
}
