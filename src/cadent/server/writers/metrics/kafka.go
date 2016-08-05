/*
	THe Kafka Metrics writer



*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
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
	Resolution uint32           `json:"resolution"`
	Id         repr.StatId      `json:"id"`
	TTL        uint32           `json:"ttl"`
	Tags       repr.SortingTags `json:"tags"` // key1=value1,key2=value2...

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
	static_tags repr.SortingTags

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

	g_tag, ok := conf["tags"]
	if ok {
		t_tags := strings.Split(g_tag.(string), ",")
		for _, tag := range t_tags {
			spl := strings.Split(tag, "=")
			if len(spl) == 2 {
				kf.static_tags = append(kf.static_tags, spl)
			}
		}
	}
	return nil
}

func (kf *KafkaMetrics) Start() {
	//noop
}

func (kf *KafkaMetrics) Stop() {
	shutdown.AddToShutdown()
	defer shutdown.ReleaseFromShutdown()
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

	kf.indexer.Write(stat.Name) // to the indexer
	stat.Name.MergeTags(kf.static_tags)
	item := &KafkaMetric{
		Type:       "metric",
		Metric:     stat.Name.Key,
		Time:       time.Now().UnixNano(),
		Sum:        stat.Sum,
		First:      stat.First,
		Last:       stat.Last,
		Mean:       stat.Mean,
		Count:      stat.Count,
		Max:        stat.Max,
		Min:        stat.Min,
		Resolution: stat.Name.Resolution,
		TTL:        stat.Name.TTL,
		Id:         stat.Name.UniqueId(),
		Tags:       stat.Name.SortedTags(),
	}

	stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.IndexTopic(),
		Key:   sarama.StringEncoder(stat.UniqueId()), // hash on unique id
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
