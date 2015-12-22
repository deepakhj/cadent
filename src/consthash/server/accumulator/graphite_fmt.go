/*
Dump the line graphite expects to get
*/

package accumulator

import (
	"fmt"
	"time"
)

const GRAPHITE_FMT_NAME = "graphite_formater"

type GraphiteFormater struct{}

func (g *GraphiteFormater) Type() string { return GRAPHITE_FMT_NAME }
func (g *GraphiteFormater) ToString(key string, val float64, tstamp int32, stats_type string, tags map[string]string) string {
	if tstamp <= 0 {
		tstamp = int32(time.Now().Unix())
	}
	return fmt.Sprintf("%s %f %d", key, val, tstamp)
}
