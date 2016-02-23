/*
   Simple make of new objects
*/

package readers

import (
	"fmt"
)

func NewReader(name string) (Reader, error) {
	switch {
	case name == "cassandra":
		return NewCassandraReader(), nil
	default:
		return nil, fmt.Errorf("Invalid reader `%s`", name)
	}
}
