/*
   Simple make of new objects
*/

package writers

import (
	"fmt"
)

func NewWriter(name string) (Writer, error) {
	switch {
	case name == "mysql":
		return NewMySQLWriter(), nil
	case name == "file":
		return NewFileWriter(), nil
	default:
		return nil, fmt.Errorf("Invalid writer `%s`", name)
	}
}
