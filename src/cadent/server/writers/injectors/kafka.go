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
*/

package injectors

import (
	"cadent/server/utils"
	"cadent/server/writers"
	"cadent/server/writers/schemas"
	"github.com/bsm/sarama-cluster"
	"gopkg.in/op/go-logging.v1"
	"log"
)

/****************** Data writers *********************/
type Kafka struct {
	Name string

	Topic         string
	ConsumerGroup string
	Brokers       string
	StartOffset   string
	EncodingType  schemas.KafkaEncodingType

	Cluster *cluster.Consumer
	Config  *cluster.Config

	writer writers.Writer

	log       *logging.Logger
	startstop utils.StartStop
}

func NewKafka(name string) *Kafka {
	kf := new(Kafka)
	kf.Name = name

	// defaults
	kf.Topic = "cadent"
	kf.ConsumerGroup = "cadent-" + kf.Name
	kf.EncodingType = schemas.KAFKA_ENCODE_MSGP

}

func (kf *Kafka) Config(map[string]interface{}) error {

	return nil
}

func (kf *Kafka) Start() error {
	var err error
	kf.startstop.Start(func() {

	})
	return err
}

func (kf *Kafka) Stop() error {
	var err error
	kf.startstop.Stop(func() {

	})
	return err
}

func (kf *Kafka) SetWriter(writer writers.Writer) error {
	kf.writer = writer
	return nil
}

func (kf *Kafka) consume() {
	k.wg.Add(1)
	messageChan := kf.consumer.Messages()
	for msg := range messageChan {
		if LogLevel < 2 {
			log.Debug("kafka-mdm received message: Topic %s, Partition: %d, Offset: %d, Key: %x", msg.Topic, msg.Partition, msg.Offset, msg.Key)
		}
		k.In.Handle(msg.Value)
		k.consumer.MarkOffset(msg, "")
	}
	log.Info("kafka-mdm consumer ended.")
	k.wg.Done()
}
