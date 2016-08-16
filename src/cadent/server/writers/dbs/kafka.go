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
	The Kafka write

	For kafka we simple write to 2 message types (they can be in different topics if desired)
	cadent_index
	cadent_data

	the index topic

	OPTIONS: For `Config`

	index_topic: topic for index message (default: cadent)
	metric_topic: topic for data messages (default: cadent)

	# some kafka options
	compress: "snappy|gzip|none"
	max_retry: 10
	ack_type: "all|local" (all = all replicas ack, default local)
	flush_time: flush produced messages ever tick (default 1s)


*/

package dbs

import (
	"cadent/server/stats"
	"fmt"
	"github.com/Shopify/sarama"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"time"
)

const (
	DEFAULT_KAFKA_BATCH_SIZE  = 1024
	DEFAULT_KAFKA_INDEX_TOPIC = "cadent"
	DEFAULT_KAFKA_DATA_TOPIC  = "cadent"
	DEFAULT_KAFKA_RETRY       = 10
	DEFAULT_KAFKA_ACK         = "local"
	DEFAULT_KAFKA_FLUSH       = "1s"
	DEFAULT_KAFKA_COMPRESSION = "snappy"
)

/****************** Interfaces *********************/
type KafkaDB struct {
	conn         sarama.AsyncProducer
	index_topic  string
	data_topic   string
	batch_count  int64
	table_prefix string

	log *logging.Logger
}

func NewKafkaDB() *KafkaDB {
	kf := new(KafkaDB)
	kf.log = logging.MustGetLogger("writers.kafka")
	return kf
}

func (kf *KafkaDB) Config(conf map[string]interface{}) error {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (kafkahost1,kafkahost2...) is needed for kafka config")
	}
	dsn := gots.(string)
	var err error

	if err != nil {
		return err
	}
	_table := conf["index_topic"]
	if _table == nil {
		kf.index_topic = DEFAULT_KAFKA_INDEX_TOPIC
	} else {
		kf.index_topic = _table.(string)
	}

	_ptable := conf["metric_topic"]
	if _ptable == nil {
		kf.data_topic = DEFAULT_KAFKA_DATA_TOPIC
	} else {
		kf.data_topic = _ptable.(string)
	}

	// batch count
	_batch := conf["batch_count"]
	if _batch == nil {
		kf.batch_count = DEFAULT_KAFKA_BATCH_SIZE
	} else {
		kf.batch_count = _batch.(int64)
	}

	// file prefix
	_retry := conf["max_retry"]
	_rt := DEFAULT_KAFKA_RETRY
	if _retry != nil {
		_rt = int(_retry.(int64))
	}

	// file prefix
	_ack := conf["ack_type"]
	_ak := DEFAULT_KAFKA_ACK
	if _ack != nil {
		_ak = _ack.(string)
	}

	_flush := DEFAULT_KAFKA_FLUSH
	_fl := conf["flush_time"]
	if _fl != nil {
		_flush = _fl.(string)
	}

	dur, err := time.ParseDuration(_flush)
	if err != nil {
		return fmt.Errorf("Invalid Kafka Flush time: %v", err)
	}

	_comp := DEFAULT_KAFKA_COMPRESSION
	_cp := conf["compression"]
	if _cp != nil {
		_comp = _cp.(string)
	}

	brokerList := strings.Split(dsn, ",")

	config := sarama.NewConfig()

	config.Producer.Retry.Max = _rt

	switch _ak {
	case "all":
		config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	default:
		config.Producer.RequiredAcks = sarama.WaitForLocal
	}

	switch _comp {
	case "snappy":
		config.Producer.Compression = sarama.CompressionSnappy
	case "gzip":
		config.Producer.Compression = sarama.CompressionGZIP
	default:
		config.Producer.Compression = sarama.CompressionNone
	}

	config.Producer.Flush.Frequency = dur // Flush batches every 500ms

	// On the broker side, you may want to change the following settings to get
	// stronger consistency guarantees:
	// - For your broker, set `unclean.leader.election.enable` to false
	// - For the topic, you could increase `min.insync.replicas`.

	kf.log.Notice("Connecting async kafka producer: %s", brokerList)
	producer, err := sarama.NewAsyncProducer(brokerList, config)
	if err != nil {
		return fmt.Errorf("Failed to start Kafka producer: %v", err)
	}

	// We will just log to STDOUT if we're not able to produce messages.
	// Note: messages will only be returned here after all retry attempts are exhausted.
	go func() {
		for err := range producer.Errors() {
			stats.StatsdClientSlow.Incr("writer.kafka.write-failures", 1)
			kf.log.Error("Failed to write message: %v", err)
		}
	}()

	go func() {
		for ret := range producer.Successes() {
			stats.StatsdClient.Incr(fmt.Sprintf("writer.kafka.%s.write-success", ret.Topic), 1)
		}
	}()
	kf.conn = producer
	return nil
}

func (kf *KafkaDB) IndexTopic() string {
	return kf.index_topic
}

func (kf *KafkaDB) DataTopic() string {
	return kf.data_topic
}

func (kf *KafkaDB) Connection() DBConn {
	return kf.conn
}
