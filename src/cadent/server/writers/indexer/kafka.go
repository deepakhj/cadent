/*
	THe Kafka Index writer



*/

package indexer

import (
	"cadent/server/stats"
	"cadent/server/writers/dbs"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"time"
)

/** basic data type **/

type KafkaPath struct {
	Type     string   `json:"type"`
	Path     string   `json:"path"`
	Segments []string `json:"segments"`
	Time     int64    `json:"time"`

	encoded []byte
	err     error
}

func (kp *KafkaPath) ensureEncoded() {
	if kp.encoded == nil && kp.err == nil {
		kp.encoded, kp.err = json.Marshal(kp)
	}
}

func (kp *KafkaPath) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KafkaPath) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

/****************** Interfaces *********************/
type KafkaIndexer struct {
	db   *dbs.KafkaDB
	conn sarama.AsyncProducer

	log *logging.Logger
}

func NewKafkaIndexer() *KafkaIndexer {
	kf := new(KafkaIndexer)
	kf.log = logging.MustGetLogger("writers.indexer.kafka")
	return kf
}

func (kf *KafkaIndexer) Config(conf map[string]interface{}) error {
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

func (kf *KafkaIndexer) Write(skey string) error {

	item := &KafkaPath{
		Type:     "index",
		Path:     skey,
		Segments: strings.Split(skey, "."),
		Time:     time.Now().UnixNano(),
	}

	stats.StatsdClientSlow.Incr("writer.kafka.indexer.writes", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.IndexTopic(),
		Key:   sarama.StringEncoder(skey), // hash on metric key
		Value: item,
	}

	return nil
}

/**** READER ***/
// just to match the interface, as there's no way to do this really
func (kf *KafkaIndexer) Find(metric string) (MetricFindItems, error) {
	return MetricFindItems{}, fmt.Errorf("KAFKA FIND CANNOT BE IMPLIMENTED")
}

func (kf *KafkaIndexer) Expand(metric string) (MetricExpandItem, error) {
	return MetricExpandItem{}, fmt.Errorf("KAFKA EXPAND CANNOT BE IMPLIMENTED")
}
