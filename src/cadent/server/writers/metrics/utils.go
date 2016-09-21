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

/*
   Writers/Readers of stats

   We are attempting to mimic the Graphite API json blobs throughout the process here
   such that we can hook this in directly to either graphite-api/web

   NOTE: this is not a full graphite DSL, just paths and metrics, we leave the fancy functions inside
   graphite-api/web to work their magic .. one day we'll impliment the full DSL, but until then ..

   Currently just implementing /find /expand and /render (json only) for graphite-api
*/

package metrics

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

/****************** Helpers *********************/
// parses things like "-23h" etc
// or pure date strings of the form 2015-07-01T20:10:30.781Z
// only does "second" precision which is all graphite can do currently
func ParseTime(st string) (int64, error) {
	st = strings.TrimSpace(strings.ToLower(st))

	// first see if it's a "date string"
	// of the form 2015-07-01T20:10:30.781Z
	_time, err := time.Parse("2015-07-01T20:10:30.781Z", st)
	if err == nil {
		return _time.Unix(), nil
	}

	// or Mon Jan 2 15:04:05 -0700 MST 2006
	_time, err = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006", st)
	if err == nil {
		return _time.Unix(), nil
	}

	unix_t := int64(time.Now().Unix())
	if st == "now" {
		return unix_t, nil
	}

	if strings.HasSuffix(st, "s") || strings.HasSuffix(st, "sec") || strings.HasSuffix(st, "second") {
		items := strings.Split(st, "s")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i, nil
	}

	if strings.HasSuffix(st, "m") || strings.HasSuffix(st, "min") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60, nil
	}

	if strings.HasSuffix(st, "h") || strings.HasSuffix(st, "hour") {
		items := strings.Split(st, "h")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60, nil
	}

	if strings.HasSuffix(st, "d") || strings.HasSuffix(st, "day") {
		items := strings.Split(st, "d")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60*24, nil
	}
	if strings.HasSuffix(st, "mon") || strings.HasSuffix(st, "month") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60*24*30, nil
	}
	if strings.HasSuffix(st, "y") || strings.HasSuffix(st, "year") {
		items := strings.Split(st, "y")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60*24*365, nil
	}

	// if it's an int already, we're good
	i, err := strconv.ParseInt(st, 10, 64)
	if err == nil {
		return i, nil
	}

	return 0, fmt.Errorf("Time `%s` could not be parsed :: %v", st, err)

}

// based on a start/end int and a step, determine just how many points we
// should be returning
func PointsInInterval(start int64, end int64, step int64) int64 {
	if step <= 0 {
		return math.MaxInt64 // basically "way too many"
	}
	return (end - start) / step
}

// based on the resolution attempt to round start/end nicely by the resolutions
func TruncateTimeTo(num int64, mod int) int64 {
	_mods := int(math.Mod(float64(num), float64(mod)))
	if _mods < mod/2 {
		return num - int64(_mods)
	}
	return num + int64(mod-_mods)
}
