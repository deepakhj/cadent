/*
	THe Kafka Index writer



*/

package indexer

import (
	"cadent/server/repr"
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
	Id       repr.StatId `json:"id"`
	Type     string      `json:"type"`
	Path     string      `json:"path"`
	Segments []string    `json:"segments"`
	SentTime int64       `json:"senttime"`
	Tags     [][]string  `json:"tags"`

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
	db          *dbs.KafkaDB
	conn        sarama.AsyncProducer
	write_index bool // if false, we skip the index writing message as well, the stat metric itself has the key in it

	log *logging.Logger
}

func NewKafkaIndexer() *KafkaIndexer {
	kf := new(KafkaIndexer)
	kf.log = logging.MustGetLogger("writers.indexer.kafka")
	kf.write_index = true
	return kf
}

func (kf *KafkaIndexer) Stop() {
	if err := kf.conn.Close(); err != nil {
		kf.log.Error("Failed to shut down producer cleanly %v", err)
	}
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

	_wr := conf["write_index"]
	if _wr != nil {
		kf.write_index = _wr.(bool)
	}

	kf.db = db.(*dbs.KafkaDB)
	kf.conn = db.Connection().(sarama.AsyncProducer)

	return nil
}

func (kf *KafkaIndexer) Write(skey repr.StatName) error {
	// noop if not writing indexes
	if !kf.write_index {
		return nil
	}

	item := &KafkaPath{
		Type:     "index",
		Id:       skey.UniqueId(),
		Path:     skey.Key,
		Segments: strings.Split(skey.Key, "."),
		Tags:     skey.SortedTags(),
		SentTime: time.Now().UnixNano(),
	}

	stats.StatsdClientSlow.Incr("writer.kafka.indexer.writes", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.IndexTopic(),
		Key:   sarama.StringEncoder(skey.Key), // hash on metric key
		Value: item,
	}

	return nil
}

// send a "delete message" to the mix
func (kf *KafkaIndexer) Delete(skey *repr.StatName) error {
	// noop if not writing indexes
	if !kf.write_index {
		return nil
	}

	item := &KafkaPath{
		Type:     "delete-index",
		Id:       skey.UniqueId(),
		Path:     skey.Key,
		Segments: strings.Split(skey.Key, "."),
		Tags:     skey.SortedTags(),
		SentTime: time.Now().UnixNano(),
	}

	stats.StatsdClientSlow.Incr("writer.kafka.indexer.delete", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.IndexTopic(),
		Key:   sarama.StringEncoder(skey.Key), // hash on metric key
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
