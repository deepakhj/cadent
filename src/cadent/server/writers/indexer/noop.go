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
