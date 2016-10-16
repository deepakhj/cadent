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
   Kafka injector

   It's not a good idea, for obvious reasons, to inject these messages into a kafka writer on the same topic
*/

package kafka

import (
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers"
	"cadent/server/writers/schemas"
	"errors"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"gopkg.in/op/go-logging.v1"
	"strings"
	"sync"
	"time"
)

var ErrorBadOffsetName = errors.New("starting_offset should be `oldest` or `newest`")
var ErrorBadMessageType = errors.New("Message type not supported by this kafka worker")
var ErrorAccumulatorNotDefined = errors.New("The accumulator is not set.")

type KafkaConfig struct {
	Topic         string `toml:"topic" json:"topic"`
	ConsumerGroup string `toml:"consumer_group" json:"consumer-group"`
	EncodingType  string `toml:"encoding" json:"encoding"`
	MessageType   string `toml:"message_type" json:"message-type"`
	StartOffset   string `toml:"starting_offset" json:"starting-offset"`
	Brokers       string `toml:"dsn" json:"dsn"`
}

/****************** Data readers *********************/
type Kafka struct {
	Name string

	Topic         string
	ConsumerGroup string
	Brokers       string
	StartOffset   string
	EncodingType  schemas.SendEncoding
	MessageType   schemas.MessageType
	KafkaWorker   Worker

	Cluster       *cluster.Consumer
	ClusterConfig *cluster.Config
	consumer      *cluster.Consumer

	writer writers.Writer

	log       *logging.Logger
	startstop utils.StartStop

	IsReady bool

	noticeStop  chan bool
	consumeStop chan bool
	markStop    chan bool
	doneWg      sync.WaitGroup

	markDoneChan chan *sarama.ConsumerMessage
}

func New(name string) *Kafka {
	kf := new(Kafka)
	kf.Name = name
	kf.noticeStop = make(chan bool, 1)
	kf.consumeStop = make(chan bool, 1)
	kf.markStop = make(chan bool, 1)
	kf.markDoneChan = make(chan *sarama.ConsumerMessage, 128)
	kf.log = logging.MustGetLogger("kafka.injestor")

	return kf
}

func (kf *Kafka) Config(conf options.Options) (err error) {
	kf.Topic = conf.String("topic", "cadent")
	kf.ConsumerGroup = conf.String("consumer_group", "cadent-"+kf.Name)
	kf.EncodingType = schemas.SendEncodingFromString(conf.String("encoding", "msgpack"))
	kf.MessageType = schemas.MetricTypeFromString(conf.String("message_type", "series"))
	kf.StartOffset = conf.String("starting_offset", "newest")
	kf.Brokers, err = conf.StringRequired("dsn")
	if err != nil {
		return err
	}

	//worker based on message type of course
	switch kf.MessageType {
	case schemas.MSG_RAW:
		kf.KafkaWorker = new(RawMetricWorker)
	case schemas.MSG_UNPROCESSED:
		kf.KafkaWorker = new(UnProcessedMetricWorker)
	case schemas.MSG_ANY:
		kf.KafkaWorker = new(AnyMetricWorker)
	default:
		return ErrorBadMessageType

	}

	kf.ClusterConfig = cluster.NewConfig()

	kf.ClusterConfig.Group.Return.Notifications = true
	kf.ClusterConfig.ChannelBufferSize = int(conf.Int64("channel_buffer_size", 1000000))
	kf.ClusterConfig.Consumer.Fetch.Min = int32(conf.Int64("consumer_fetch_min", 1))
	kf.ClusterConfig.Consumer.Fetch.Default = int32(conf.Int64("consumer_fetch_default", 4096000))
	kf.ClusterConfig.Consumer.MaxWaitTime = conf.Duration("consumer_max_wait_time", time.Second*time.Duration(1))
	kf.ClusterConfig.Consumer.MaxProcessingTime = conf.Duration("consumer_max_processing_time", time.Second*time.Duration(1))
	kf.ClusterConfig.Net.MaxOpenRequests = int(conf.Int64("max_open_requests", 128))
	kf.ClusterConfig.Config.Version = sarama.V0_10_0_0

	switch kf.StartOffset {
	case "oldest":
		kf.ClusterConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "newest":
		kf.ClusterConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		return ErrorBadOffsetName
	}

	err = kf.ClusterConfig.Validate()
	if err != nil {
		return err
	}

	kf.consumer, err = cluster.NewConsumer(
		strings.Split(kf.Brokers, ","),
		kf.ConsumerGroup,
		strings.Split(kf.Topic, ","),
		kf.ClusterConfig,
	)
	if err != nil {
		return err
	}

	return nil
}

func (kf *Kafka) Start() error {
	var err error
	kf.startstop.Start(func() {
		go kf.onNotify()
		go kf.onConsume()
		go kf.onMark()
	})
	return err
}

func (kf *Kafka) Stop() error {
	var err error
	kf.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		kf.log.Notice("Kafka shutting down")
		err = kf.consumer.CommitOffsets()
		if err != nil {
			kf.log.Errorf("Kafka stop fail: %s", err)
			return
		}
		err = kf.consumer.Close()
		if err != nil {
			kf.log.Errorf("Kafka stop fail: %s", err)
			return
		}
		kf.noticeStop <- true
		kf.consumeStop <- true
		kf.markStop <- true
		kf.log.Notice("Kafka stopped")
	})
	return err
}

func (kf *Kafka) SetWriter(writer writers.Writer) error {
	kf.writer = writer
	return nil
}

func (kf *Kafka) onNotify() {
	notsChan := kf.consumer.Notifications()

	kf.doneWg.Add(1)
	defer kf.doneWg.Done()

	for {
		select {
		case msg := <-notsChan:
			if msg == nil {
				continue
			}
			if len(msg.Claimed) > 0 {

				for topic, partitions := range msg.Claimed {
					kf.log.Info("Claimed %d partitions on topic: %s", len(partitions), topic)
				}
			}
			if len(msg.Released) > 0 {
				for topic, partitions := range msg.Released {
					kf.log.Info("Released %d partitions on topic: %s", len(partitions), topic)
				}
			}

			if len(msg.Current) == 0 {
				kf.log.Warning("Released all partitions on topic: %s", kf.Topic)
				kf.IsReady = false

			} else {
				for topic, partitions := range msg.Current {
					kf.log.Notice("Current partitions: %s: %v", topic, partitions)
				}
				kf.IsReady = true

			}
		case <-kf.noticeStop:
			kf.log.Notice("Shutting down kafka notifier")
			return
		}
	}
}

func (kf *Kafka) onConsume() {
	kf.doneWg.Add(1)
	defer kf.doneWg.Done()

	dataChan := kf.consumer.Messages()
	for {
		select {
		case <-kf.consumeStop:
			kf.log.Notice("Consumer stopped")
			return
		case msg, more := <-dataChan:

			if !more {
				kf.log.Notice("Consumer stopped")
				return
			}
			//kf.log.Debug("Got message from %s: partition: %d, offset: %d", msg.Topic, msg.Partition, msg.Offset)
			new_obj := schemas.KMetricObjectFromType(kf.MessageType)
			new_obj.SetSendEncoding(kf.EncodingType)

			err := new_obj.Decode(msg.Value)

			// we have commit offsets for decode fails otherwise we'll never move
			if err != nil {
				kf.log.Errorf("Could not process incoming message: %v", err)
				schemas.PutPool(new_obj) // put it back in the syc pool
				kf.markDoneChan <- msg
				continue
			}

			err = kf.KafkaWorker.DoWork(new_obj)
			if err != nil {
				kf.log.Errorf("Error working on message: %v", err)
				schemas.PutPool(new_obj) // put it back in the syc pool

				continue
			}
			schemas.PutPool(new_obj) // put it back in the syc pool

			kf.markDoneChan <- msg
		}
	}
}

func (kf *Kafka) onMark() {
	kf.doneWg.Add(1)
	defer kf.doneWg.Done()
	for {
		select {
		case <-kf.markStop:
			kf.log.Notice("Marking stopped")
			return
		case msg, more := <-kf.markDoneChan:
			if !more {
				kf.log.Notice("Mark offset Consumer stopped")
				return
			}
			//kf.log.Debug("Marking offset: %s: %d", msg.Topic, msg.Offset)
			kf.consumer.MarkOffset(msg, "")
		}
	}
}
