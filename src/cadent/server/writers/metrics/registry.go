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
	case name == "whisper":
		return NewWhisperMetrics(), nil
	default:
		return nil, fmt.Errorf("Invalid metrics `%s`", name)
	}
}
