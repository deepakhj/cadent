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

// a simple atomic stat counter/rate
package stats

import (
	"statsd"
	"strings"
	"time"
)

//statsd client singleton for "fast" timers (i.e. sampling rates in the 1% range)
var StatsdClient statsd.Statsd = nil

// statsd client singleton for "raw" (no sampling rates) for slow items
var StatsdClientSlow statsd.Statsd = nil

var NAME_SANITIZER *strings.Replacer

func SanitizeName(name string) string {
	return NAME_SANITIZER.Replace(name)
}

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
	NAME_SANITIZER = strings.NewReplacer(
		"..", ".",
		",", "_",
		"=", "_",
		"*", "_",
		"(", "_",
		")", "_",
		"{", "_",
		"}", "_",
		"^", "_",
		":", "_",
		"$", "_",
		" ", "_",
		"!", "_",
		"@", "_",
		"#", "_",
		"&", "_",
		"%", "_",
		"~", "_",
		"`", "_",
		"'", "_",
		">", "_",
		"<", "_",
		"?", "_",
		"/", "_",
		"]", "_",
		"[", "_",
		"|", "_",
		"\\", "_",
		";", "_",
	)
}
