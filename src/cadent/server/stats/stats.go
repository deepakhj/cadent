// a simple atomic stat counter/rate
package stats

import (
	"statsd"
	"time"
)

//statsd client singleton for "fast" timers (i.e. sampling rates in the 1% range)
var StatsdClient statsd.Statsd = nil

// statsd client singleton for "raw" (no sampling rates) for slow items
var StatsdClientSlow statsd.Statsd = nil

//a handy "defer" function for timers, in Nano seconds
func StatsdNanoTimeFunc(statname string, start time.Time) {

	// XXXX
	//return // BIG Performance HIT here for very fast functions

	elapsed := time.Since(start)
	StatsdClient.Timing(statname, int64(elapsed))
}

//a handy "defer" function for timers, in Nano seconds
func StatsdSlowNanoTimeFunc(statname string, start time.Time) {
	elapsed := time.Since(start)
	StatsdClientSlow.Timing(statname, int64(elapsed))
}

// set to noop statds client initially
func init() {
	if StatsdClient == nil {
		StatsdClient = new(statsd.StatsdNoop)
		StatsdClientSlow = new(statsd.StatsdNoop)
	}
}
