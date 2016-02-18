/*
   Writers of stats
*/

package writers

import (
	"consthash/server/repr"
)

/****************** Data Wrtiers *********************/
type Writer interface {
	Config(map[string]interface{}) error
	Write(repr.StatRepr) error
}
