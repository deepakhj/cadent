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
   Injectors

   configuration tests
*/

package injectors

import (
	"cadent/server/utils/options"
	"cadent/server/writers/injectors/kafka"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestInjectorConfig(t *testing.T) {
	// Only pass t into top-level Convey calls

	c := InjectorConfig{Options: options.Options{}}
	c.Name = "test"
	c.Driver = "kafka"
	c.DSN = "127.0.0.1:9092"
	c.Accumulator = "na"

	c.Options.Set("topic", "cadent")
	c.Options.Set("consumer_group", "cadent-test")
	c.Options.Set("encoding", "json")
	c.Options.Set("message_type", "raw")

	Convey("Should provide a valid kafka injector", t, func() {

		k, err := c.New()
		So(err, ShouldEqual, nil)

		So(k.(*kafka.Kafka).Brokers, ShouldEqual, "127.0.0.1:9092")
	})
}
