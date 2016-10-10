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
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"cadent/server/writers/schemas"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	logging "gopkg.in/op/go-logging.v1"
	"time"
)

var errKafkaReaderNotImplimented = errors.New("KAFKA READER NOT IMPLMENTED")

/****************** Interfaces *********************/
type KafkaFlatMetrics struct {
	WriterBase

	db   *dbs.KafkaDB
	conn sarama.AsyncProducer

	enctype schemas.SendEncoding
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

func (kf *KafkaFlatMetrics) Config(conf *options.Options) error {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (kafkahost1,kafkahost2) is needed for kafka config")
	}

	db, err := dbs.NewDB("kafka", dsn, conf)
	if err != nil {
		return err
	}

	g_tag := conf.String("tags", "")
	if len(g_tag) > 0 {
		kf.static_tags = repr.SortingTagsFromString(g_tag)
	}

	enct := conf.String("encoding", "json")
	if len(enct) > 0 {
		kf.enctype = schemas.SendEncodingFromString(enct)
	}

	kf.db = db.(*dbs.KafkaDB)
	kf.conn = db.Connection().(sarama.AsyncProducer)

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

func (cass *KafkaMetrics) GetResolutions() [][]int {
	return cass.resolutions
}

func (kf *KafkaFlatMetrics) SetCurrentResolution(res int) {
	kf.currentResolution = res
}

func (kf *KafkaFlatMetrics) Write(stat repr.StatRepr) error {

	if kf.shutitdown {
		return nil
	}

	stat.Name.MergeMetric2Tags(kf.static_tags)
	kf.indexer.Write(*stat.Name) // to the indexer
	item := &schemas.KMetric{
		AnyMetric: schemas.AnyMetric{
			Single: &schemas.SingleMetric{
				Metric:     stat.Name.Key,
				Time:       time.Now().UnixNano(),
				Sum:        float64(stat.Sum),
				Last:       float64(stat.Last),
				Count:      stat.Count,
				Max:        float64(stat.Max),
				Min:        float64(stat.Min),
				Resolution: stat.Name.Resolution,
				Ttl:        stat.Name.Ttl,
				Id:         uint64(stat.Name.UniqueId()),
				Uid:        stat.Name.UniqueIdString(),
				Tags:       stat.Name.SortedTags(),
				MetaTags:   stat.Name.SortedMetaTags(),
			},
		},
	}
	item.SetSendEncoding(kf.enctype)

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

func (kf *KafkaFlatMetrics) RawRender(string, int64, int64, repr.SortingTags, uint32) ([]*RawRenderItem, error) {
	return []*RawRenderItem{}, errKafkaReaderNotImplimented
}
func (kf *KafkaFlatMetrics) CacheRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error) {
	return nil, errKafkaReaderNotImplimented
}
func (kf *KafkaFlatMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*TotalTimeSeries, error) {
	return nil, errKafkaReaderNotImplimented
}
