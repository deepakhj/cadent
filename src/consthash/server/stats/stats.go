// a simple atomic stat counter/rate
package stats

import (
	"statsd"
	"time"
)

//statsd client singleton
var StatsdClient statsd.Statsd = nil

//a handy "defer" function for timers, in Nano seconds
func StatsdNanoTimeFunc(statname string, start time.Time) {

	// XXXX
	//return // BIG Performance HIT here for very fast functions

	elapsed := time.Since(start)
	StatsdClient.Timing(statname, int64(elapsed))
}

// set to noop statds client initially
func init() {
	if StatsdClient == nil {
		StatsdClient = new(statsd.StatsdNoop)
	}
}
