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

   configuration elements
*/

package injectors

import (
	"cadent/server/utils/options"
	"cadent/server/writers/injectors/kafka"
	"fmt"
)

type InjectorConfig struct {
	Name        string          `toml:"name" json:"name,omitempty"`
	Driver      string          `toml:"driver" json:"driver,omitempty"`
	DSN         string          `toml:"dsn"  json:"dsn,omitempty"`
	Accumulator string          `toml:"accumulator"  json:"accumulator,omitempty"`
	Options     options.Options `toml:"options"  json:"options,omitempty"`
}

func (c *InjectorConfig) New() (Injector, error) {
	c.Options.Set("dsn", c.DSN)

	switch c.Driver {
	case "kafka":
		kf := kafka.New(c.Name)
		err := kf.Config(c.Options)
		return kf, err
	default:
		return nil, fmt.Errorf("Invalid injector driver %s", c.Driver)
	}
}
