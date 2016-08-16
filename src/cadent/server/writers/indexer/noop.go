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
  simple "no op" indexer that does nothing
*/

package indexer

import (
	"cadent/server/repr"
	"fmt"
)

type NoopIndexer struct{}

func NewNoopIndexer() *NoopIndexer {
	return new(NoopIndexer)
}

func (my *NoopIndexer) Config(conf map[string]interface{}) error {
	return nil
}
func (my *NoopIndexer) Name() string                     { return "noop-indexer" }
func (my *NoopIndexer) Stop()                            {}
func (my *NoopIndexer) Start()                           {}
func (my *NoopIndexer) Delete(name *repr.StatName) error { return nil }

func (my *NoopIndexer) Write(metric repr.StatName) error {
	return nil
}
func (my *NoopIndexer) Find(metric string) (MetricFindItems, error) {
	return MetricFindItems{}, fmt.Errorf("Noop indexer cannot find anything")
}

func (my *NoopIndexer) Expand(metric string) (MetricExpandItem, error) {
	return MetricExpandItem{}, fmt.Errorf("Noop indexer cannot expand anything")
}
