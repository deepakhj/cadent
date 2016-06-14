/*
	THe Kafka Metrics writer



*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	logging "gopkg.in/op/go-logging.v1"
	"time"
)

/** kafka put object **/
type KafkaMetric struct {
	Type       string           `json:"type"`
	Time       int64            `json:"time"`
	Metric     string           `json:"metric"`
	Sum        repr.JsonFloat64 `json:"sum"`
	Mean       repr.JsonFloat64 `json:"mean"`
	Min        repr.JsonFloat64 `json:"min"`
	Max        repr.JsonFloat64 `json:"max"`
	Count      int64            `json:"count"`
	Last       repr.JsonFloat64 `json:"last"`
	First      repr.JsonFloat64 `json:"first"`
	Resolution float64          `json:"resolution"`
	TTL        int64            `json:"ttl"`

	encoded []byte
	err     error
}

func (kp *KafkaMetric) ensureEncoded() {
	if kp.encoded == nil && kp.err == nil {
		kp.encoded, kp.err = json.Marshal(kp)
	}
}

func (kp *KafkaMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KafkaMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

/****************** Interfaces *********************/
type KafkaMetrics struct {
	db          *dbs.KafkaDB
	conn        sarama.AsyncProducer
	indexer     indexer.Indexer
	resolutions [][]int

	batches int // number of stats to "batch" per message (default 0)
	log     *logging.Logger
}

func NewKafkaMetrics() *KafkaMetrics {
	kf := new(KafkaMetrics)
	kf.batches = 0
	kf.log = logging.MustGetLogger("writers.kafka.metrics")
	return kf
}

func (kf *KafkaMetrics) Config(conf map[string]interface{}) error {
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

	return nil
}

// TODO
func (kf *KafkaMetrics) Stop() {
	if err := kf.conn.Close(); err != nil {
		kf.log.Error("Failed to shut down producer cleanly %v", err)
	}
}

func (kf *KafkaMetrics) SetIndexer(idx indexer.Indexer) error {
	kf.indexer = idx
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (kf *KafkaMetrics) SetResolutions(res [][]int) int {
	kf.resolutions = res
	return len(res) // need as many writers as bins
}

func (kf *KafkaMetrics) Write(stat repr.StatRepr) error {

	kf.indexer.Write(stat.Key) // to the indexer
	item := &KafkaMetric{
		Type:       "metric",
		Metric:     stat.Key,
		Time:       time.Now().UnixNano(),
		Sum:        stat.Sum,
		First:      stat.First,
		Last:       stat.Last,
		Mean:       stat.Mean,
		Count:      stat.Count,
		Max:        stat.Max,
		Min:        stat.Min,
		Resolution: stat.Resolution,
		TTL:        stat.TTL,
	}

	stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.IndexTopic(),
		Key:   sarama.StringEncoder(stat.Key), // hash on metric key
		Value: item,
	}
	return nil
}

/**** READER ***/
// needed to match interface, but we obviously cannot do this
func (kf *KafkaMetrics) Render(path string, from string, to string) (WhisperRenderItem, error) {
	return WhisperRenderItem{}, fmt.Errorf("KAKFA DRIVER CANNOT DO RENDER")
}
func (kf *KafkaMetrics) RawRender(path string, from string, to string) ([]*RawRenderItem, error) {
	return []*RawRenderItem{}, fmt.Errorf("KAKFA DRIVER CANNOT DO RENDER")
}
