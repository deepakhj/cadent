/*
   Simple make of new objects
*/

package metrics

import (
	"fmt"
)

func NewMetrics(name string) (Metrics, error) {
	switch {
	case name == "mysql":
		return NewMySQLMetrics(), nil
	case name == "file":
		return NewFileMetrics(), nil
	case name == "cassandra":
		return NewCassandraMetrics(), nil
	case name == "cassandra-flat":
		return NewCassandraFlatMetrics(), nil
	case name == "whisper" || name == "carbon" || name == "graphite":
		return NewWhisperMetrics(), nil
	case name == "kafka":
		return NewKafkaMetrics(), nil
	default:
		return nil, fmt.Errorf("Invalid metrics `%s`", name)
	}
}

func ResolutionsNeeded(name string) (WritersNeeded, error) {
	switch {
	case name == "mysql":
		return AllResolutions, nil
	case name == "file":
		return AllResolutions, nil
	case name == "cassandra":
		return AllResolutions, nil
	case name == "cassandra-flat":
		return AllResolutions, nil
	case name == "kafka":
		return AllResolutions, nil
	case name == "whisper" || name == "carbon" || name == "graphite":
		return FirstResolution, nil
	default:
		return AllResolutions, fmt.Errorf("Invalid metrics `%s`", name)
	}
}
