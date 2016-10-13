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
   Kafka injector tester
*/

package kafka

import (
	"testing"
	//. "github.com/smartystreets/goconvey/convey"
	"cadent/server/accumulator"
	"cadent/server/repr"
	"cadent/server/test/helper"
	"cadent/server/utils/options"
	"cadent/server/writers/schemas"
	"fmt"
	"github.com/Shopify/sarama"
	"time"
)

var topic string = "cadent"
var kport string = "9092"

var enctypes []string = []string{"json", "msgpack", "protobuf"}

type TestWorker struct {
	Name string

	all_ct int
	raw_ct int
	up_ct  int
	sin_ct int
	ser_ct int
}

func (w *TestWorker) Config(options.Options) error {
	return nil
}

func (w *TestWorker) SetAccumulator(acc *accumulator.Accumulator) {
}

func (w *TestWorker) GetAccumulator() *accumulator.Accumulator {
	return nil
}

func (w *TestWorker) DoWork(metric schemas.KMessageBase) error {
	w.all_ct++
	switch metric.(type) {
	case *schemas.KMetric:
		m := metric.(*schemas.KMetric)
		r := m.Repr()
		fmt.Println("AnyMatric")
		if m.Raw != nil {
			w.raw_ct++
			fmt.Println(r)
		}
		if m.Single != nil {
			w.sin_ct++
			fmt.Println(r)
		}
		if m.Unprocessed != nil {
			w.up_ct++
			fmt.Println(r)
		}
		if m.Series != nil {
			w.ser_ct++
			fmt.Println(r)
		}
		return nil
	case *schemas.KRawMetric:
		fmt.Println("RawMetric")
		fmt.Println(metric.(*schemas.KRawMetric).Repr())
		w.raw_ct++
		return nil
	case *schemas.KUnProcessedMetric:
		fmt.Println("Unprocessed")
		fmt.Println(metric.(*schemas.KUnProcessedMetric).Repr())
		w.up_ct++
		return nil
	case *schemas.KSingleMetric:
		fmt.Println("Single")
		fmt.Println(metric.(*schemas.KSingleMetric).Repr())
		w.sin_ct++
		return nil
	case *schemas.KSeriesMetric:
		fmt.Println("Series")
		fmt.Println("metric series: " + metric.(*schemas.KSeriesMetric).Metric)
		w.ser_ct++
		return nil
	default:
		return ErrorBadMessageType
	}
}

func getConsumer(enctype string) (*Kafka, error) {
	on_ip := helper.DockerIp()

	config_opts := options.Options{}
	config_opts.Set("dsn", fmt.Sprintf("%s:%s", on_ip, kport))
	config_opts.Set("topic", topic)
	config_opts.Set("consumer_group", "cadent-test")
	config_opts.Set("starting_offset", "oldest")
	config_opts.Set("encoding", enctype)
	config_opts.Set("message_type", "any")

	kf := New("tester")
	err := kf.Config(config_opts)
	if err != nil {
		return nil, err
	}

	// set the work to the echo type
	kf.KafkaWorker = new(TestWorker)

	return kf, nil
}

func getProducer() (sarama.AsyncProducer, error) {
	config := sarama.NewConfig()
	on_ip := helper.DockerIp()

	config.Producer.Retry.Max = int(3)
	config.Producer.RequiredAcks = sarama.WaitForAll
	producer, err := sarama.NewAsyncProducer([]string{on_ip + ":" + kport}, config)

	return producer, err
}

func getMetrics(len int, useencoding schemas.SendEncoding) []schemas.KMessageBase {
	msgs := make([]schemas.KMessageBase, len)

	for idx := range msgs {

		switch idx % 3 {
		case 0:
			msgs[idx] = &schemas.KMetric{
				AnyMetric: schemas.AnyMetric{
					Raw: &schemas.RawMetric{
						Metric: "moo.goo",
						Time:   time.Now().Unix(),
						Value:  100.0 * float64(idx),
						Tags:   repr.SortingTags([]*repr.Tag{{Name: "name", Value: "val"}, {Name: "name2", Value: "val2"}}),
					},
				},
			}
		case 1:
			msgs[idx] = &schemas.KMetric{
				AnyMetric: schemas.AnyMetric{
					Unprocessed: &schemas.UnProcessedMetric{
						Metric: "moo.goo.unp",
						Time:   time.Now().Unix(),
						Sum:    1000.0,
						Min:    1.0,
						Max:    120.0,
						Count:  2 + int64(idx),
						Tags:   repr.SortingTags([]*repr.Tag{{Name: "name", Value: "val"}, {Name: "name2", Value: "val2"}}),
					},
				},
			}

		default:

			msgs[idx] = &schemas.KMetric{
				AnyMetric: schemas.AnyMetric{
					Single: &schemas.SingleMetric{
						Metric: "moo.goo.sing",
						Id:     123123,
						Uid:    "asdasd",
						Time:   time.Now().Unix(),
						Sum:    100.0,
						Min:    1.0,
						Max:    10.0,
						Count:  20 + int64(idx),
						Tags:   repr.SortingTags([]*repr.Tag{{Name: "name", Value: "val"}, {Name: "name2", Value: "val2"}}),
					},
				},
			}
		}
		msgs[idx].SetSendEncoding(useencoding)
	}
	return msgs
}

func TestKafkaInjector(t *testing.T) {

	// fire up the docker test helper
	helper.DockerUp("kafka")
	ok := helper.DockerWaitUntilReady("kafka")
	if !ok {
		t.Fatalf("Could not start the docker container for kafka")
	}

	for _, enctype := range enctypes {
		var useencoding schemas.SendEncoding = schemas.SendEncodingFromString(enctype)

		kf, err := getConsumer(enctype)
		if err != nil {
			t.Fatalf("Failed to get consumer: %v", err)
		}
		err = kf.Start()
		if err != nil {
			t.Fatalf("Failed to start: %v", err)
		}

		t.Logf("consumer: started")

		// some raw messages
		NumMessages := 10
		msgs := getMetrics(NumMessages, useencoding)
		prod, err := getProducer()
		if err != nil {
			t.Fatalf("Error on producer: %v", err)
		}

		t.Logf("testing messages")

		for {
			if kf.IsReady {

				for _, msg := range msgs {
					prod.Input() <- &sarama.ProducerMessage{
						Topic: topic,
						Key:   sarama.StringEncoder(msg.Id()),
						Value: msg,
					}
					t.Logf("Produced: %v : %v", msg.Id(), msg)
				}
				prod.Close()
				break

			}
			time.Sleep(time.Second)
		}

		t.Logf("consumer: running")

		time.Sleep(20)
		t.Logf("consumer: stopping")

		err = kf.Stop()
		if err != nil {
			t.Fatalf("Error stop kafka: %v", err)
		}
		wrk := kf.KafkaWorker.(*TestWorker)
		if wrk.all_ct != NumMessages {
			t.Fatalf("Raw metric counts do not match produced %d/%d", wrk.all_ct, NumMessages)
		}

		t.Logf("consumer: stopped")
	}

}

func Benchmark__Kafka_Encoding_JSON(b *testing.B) {
	// fire up the docker test helper
	helper.DockerUp("kafka")
	ok := helper.DockerWaitUntilReady("kafka")
	if !ok {
		b.Fatalf("Could not start the docker container for kafka")
	}

	kf, err := getConsumer("json")
	if err != nil {
		b.Fatalf("Failed to get consumer: %v", err)
	}
	err = kf.Start()
	if err != nil {
		b.Fatalf("Failed to start: %v", err)
	}

	// some raw messages
	NumMessages := 1000
	msgs := getMetrics(NumMessages, schemas.ENCODE_JSON)
	prod, err := getProducer()
	if err != nil {
		b.Fatalf("%v", err)
	}
	//wait till we are good to go
	for {
		if kf.IsReady {
			break
		}
		time.Sleep(time.Second)

	}
	b.ResetTimer()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range msgs {
			prod.Input() <- &sarama.ProducerMessage{
				Topic: topic,
				Key:   sarama.StringEncoder(msg.Id()),
				Value: msg,
			}
		}
	}
	b.StopTimer()
	prod.Close()

	err = kf.Stop()

}
