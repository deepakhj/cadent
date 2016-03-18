/*
   Simple make of new objects
*/

package indexer

import (
	"fmt"
)

func NewIndexer(name string) (Indexer, error) {
	switch {
	case name == "cassandra":
		return NewCassandraIndexer(), nil
	case name == "whisper":
		return NewWhisperIndexer(), nil
	default:
		return nil, fmt.Errorf("Invalid indexer `%s`", name)
	}
}
