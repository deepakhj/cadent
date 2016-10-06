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

package main

// a little "blaster of stats" send to kafka
// we use the basic series type in schema

import (
	"cadent/server/utils/options"
	"cadent/server/writers/dbs"
	"cadent/server/writers/schemas"
	"github.com/Shopify/sarama"
	"log"
	"os"
	"time"
)

func GetKafkaProducer(broker string) sarama.AsyncProducer {
	opts := options.New()
	opts.Set("compression", "none")
	opts.Set("dsn", broker)
	k, e := dbs.NewDB("kafka", broker, &opts)
	if e != nil {
		panic(e)
	}

	return k.Connection().(sarama.AsyncProducer)
}

func UnProcessedMakeKafkaMessage() *schemas.KUnProcessedMetric {
	max := 1000
	div := 10
	len := 4
	taglen := 6
	tgs := sprinterTagList(taglen)
	item := &schemas.KUnProcessedMetric{
		UnProcessedMetric: schemas.UnProcessedMetric{
			Metric: sprinter(len),
			Time:   time.Now().UnixNano(),
			Sum:    randFloat(max, div),
			Last:   randFloat(max, div),
			Count:  randInt(max),
			Max:    randFloat(max, div),
			Min:    randFloat(max, div),
			Tags:   schemas.ToMetricTag(tgs),
		},
	}

	return item
}

func KafkaBlast(server string, topic string, rate string, buffer int, enc string) {

	conn := GetKafkaProducer(server)
	sleeper, err := time.ParseDuration(rate)
	if err != nil {
		log.Printf("Error in Rate: %s", err)
		os.Exit(1)
	}
	for {
		stat := UnProcessedMakeKafkaMessage()
		i_enc := schemas.SendEncodingFromString(enc)
		stat.SetSendEncoding(i_enc)
		conn.Input() <- &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(stat.Metric), // hash on unique id
			Value: stat,
		}
		sentLines++
		time.Sleep(sleeper)
	}
}

/**/
