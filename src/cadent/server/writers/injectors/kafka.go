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

package injectors

import (
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers"
	"cadent/server/writers/schemas"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"github.com/onsi/ginkgo/config"
	"gopkg.in/op/go-logging.v1"
	"strings"
	"sync"
	"time"
)

const ErrorBadOffsetName = errors.New("starting_offset should be `oldest` or `newest`")

/****************** Data readers *********************/
type Kafka struct {
	Name string

	Topic         string
	ConsumerGroup string
	Brokers       string
	StartOffset   string
	EncodingType  schemas.KafkaEncodingType
	MessageType   schemas.KafkaMessageType

	Cluster  *cluster.Consumer
	Config   *cluster.Config
	consumer *cluster.Consumer

	writer writers.Writer

	log       *logging.Logger
	startstop utils.StartStop

	noticeStop  chan bool
	consumeStop chan bool
	doneWg      sync.WaitGroup
}

func NewKafka(name string) *Kafka {
	kf := new(Kafka)
	kf.Name = name
	kf.noticeStop = make(chan bool, 1)
	kf.consumeStop = make(chan bool, 1)
	kf.log = logging.MultiLogger("kafka.injestor")

	return kf
}

func (kf *Kafka) Config(conf options.Options) (err error) {
	kf.Topic = conf.String("topic", "cadent")
	kf.ConsumerGroup = conf.String("group", "cadent-"+kf.Name)
	kf.EncodingType = schemas.KafkaEncodingFromString(conf.String("encoding", "msgpack"))
	kf.MessageType = schemas.KafkaMessageFromString(conf.String("message_type", "series"))
	kf.StartOffset = conf.String("starting_offset", "newest")
	kf.Brokers, err = conf.StringRequired("dsn")

	kf.Config = cluster.NewConfig()

	kf.Config.Group.Return.Notifications = true
	kf.Config.ChannelBufferSize = int(conf.Int64("channel_buffer_size", 1000000))
	kf.Config.Consumer.Fetch.Min = int(conf.Int64("consumer_fetch_min", 1024000))
	kf.Config.Consumer.Fetch.Default = int(conf.Int64("consumer_fetch_default", 4096000))
	kf.Config.Consumer.MaxWaitTime = conf.Duration("consumer_max_wait_time", time.Second*time.Duration(1))
	kf.Config.Consumer.MaxProcessingTime = conf.Duration("consumer_max_processing_time", time.Second*time.Duration(1))
	kf.Config.Net.MaxOpenRequests = int(conf.Int64("max_open_requests", 128))
	kf.Config.Config.Version = sarama.V0_10_0_0

	switch kf.StartOffset {
	case "oldest":
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "newest":
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		return ErrorBadOffsetName
	}

	err = kf.Config.Validate()
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	kf.consumer, err = cluster.NewConsumer(
		strings.Split(kf.Brokers, ","),
		kf.ConsumerGroup,
		strings.Split(kf.Topic, ","),
		kf.Config,
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
	})
	return err
}

func (kf *Kafka) Stop() error {
	var err error
	kf.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
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
				kf.log.Warning("Released %d partitions on topic: %s", kf.Topic)

			} else {
				for topic, partitions := range msg.Current {
					kf.log.Notice("Current partitions: %s: %v", topic, partitions)
				}
			}
		case <-kf.noticeStop:
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
			kf.log.Debug("Got message from %s: partition: %d, offset: %d", msg.Topic, msg.Partition, msg.Offset)
			new_obj := schemas.MetricTypeFromString(kf.MessageType)
			err := new_obj.Decode(msg.Value)
			if err != nil {
				kf.log.Errorf("Counld not process incoming message: %v", err)
			}
			//
			// Do "write" and verify
			//
			kf.consumer.MarkOffset(msg, "")
		}
	}
}
