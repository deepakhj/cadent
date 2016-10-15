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
   Simple make of new objects
*/

package metrics

import (
	"errors"
	"fmt"
)

// cache configs are required for some metric writers
var errMetricsCacheRequired = errors.New("Cache object is required")

// /cached/series endpoint cannot have multi targets
var errMultiTargetsNotAllowed = errors.New("Multiple Targets are not allowed")

// somehow a nil name
var errNameIsNil = errors.New("Name object cannot be nil")

// somehow a nil series
var errSeriesIsNil = errors.New("Name object cannot be nil")

func NewWriterMetrics(name string) (Metrics, error) {
	switch {
	case name == "mysql":
		return NewMySQLMetrics(), nil
	case name == "mysql-triggered":
		return NewMySQLTriggeredMetrics(), nil
	case name == "mysql-flat":
		return NewMySQLFlatMetrics(), nil
	case name == "file":
		return NewCSVFileMetrics(), nil
	case name == "cassandra":
		return NewCassandraMetrics(), nil
	case name == "cassandra-triggered":
		return NewCassandraTriggerMetrics(), nil
	case name == "cassandra-flat":
		return NewCassandraFlatMetrics(), nil
	case name == "elasticsearch-flat" || name == "elastic-flat":
		return NewElasticSearchFlatMetrics(), nil
	case name == "whisper" || name == "carbon" || name == "graphite":
		return NewWhisperMetrics(), nil
	case name == "kafka":
		return NewKafkaMetrics(), nil
	case name == "kafka-flat":
		return NewKafkaFlatMetrics(), nil
	default:
		return nil, fmt.Errorf("Invalid metrics `%s`", name)
	}
}

func ResolutionsNeeded(name string) (WritersNeeded, error) {
	switch {
	case name == "mysql":
		return AllResolutions, nil
	case name == "mysql-flat":
		return AllResolutions, nil
	case name == "file":
		return AllResolutions, nil
	case name == "cassandra":
		return AllResolutions, nil
	case name == "cassandra-flat":
		return AllResolutions, nil
	case name == "elasticsearch-flat" || name == "elastic-flat":
		return AllResolutions, nil
	case name == "kafka" || name == "kafka-flat":
		return AllResolutions, nil
	case name == "whisper" || name == "carbon" || name == "graphite" || name == "cassandra-triggered" || name == "mysql-triggered":
		return FirstResolution, nil
	default:
		return AllResolutions, fmt.Errorf("Invalid metrics `%s`", name)
	}
}
