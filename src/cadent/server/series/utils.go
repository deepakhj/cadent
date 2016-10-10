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
	Interface bits for time series
*/

package series

import (
	"cadent/server/repr"
	"errors"
	"time"
)

var errResolutionTooSmall = errors.New("Resolution cannot be <= 0")

// make the "second" and "nanosecond" parts
func splitNano(t int64) (uint32, uint32) {
	// not "good way" of splitting a Nano-time is available so we need to
	// convert things to "time" and grab the resulting bits

	// already in seconds
	if t <= 2147483647 {
		return uint32(t), 0
	}
	tt := time.Unix(0, t)
	return uint32(tt.Unix()), uint32(tt.Nanosecond())
}

// remake a "nano-time"
func combineSecNano(ts uint32, tns uint32) int64 {
	// not "good way" of splitting a Nano-time is available so we need to
	// convert things to "time" and grab the resulting bits
	tt := time.Unix(int64(ts), int64(tns))
	return tt.UnixNano()
}

// see if all the floats are the same
func sameFloatVals(min float64, max float64, last float64, sum float64) bool {
	return min == max && min == last && min == sum
}

// merge and resample 2 timeseries
// it will use the Series TYPE and OPTIONS from the First timeseries in the list
func MergeAndResample(ts1 TimeSeries, ts2 TimeSeries, step uint32) (TimeSeries, error) {

	if step <= 0 {
		return nil, errResolutionTooSmall
	}

	// make sure the start/ends are nicely divisible by the Step
	start := ts1.StartTime()
	if ts2.StartTime() < start {
		start = ts2.StartTime()
	}

	// times for series are in nanseconds, the step is in seconds
	// so upconv the step
	nano_step := int64(step) * int64(time.Second)

	left := start % nano_step
	if left != 0 {
		start = start + nano_step - left
	}

	end := ts1.LastTime()
	if ts2.LastTime() > end {
		end = ts2.LastTime()
	}

	endTime := (end - 1) - ((end - 1) % nano_step) + nano_step

	iter1, err := ts1.Iter()
	if err != nil {
		return nil, err
	}

	iter2, err := ts2.Iter()
	if err != nil {
		return nil, err
	}

	opts := NewDefaultOptions()
	opts.HighTimeResolution = ts1.HighResolution()
	switch ts1.(type) {
	case *GorillaTimeSeries:
		opts.NumValues = int64(ts1.(*GorillaTimeSeries).numValues)
	case *CodecTimeSeries:
		opts.Handler = ts1.(*CodecTimeSeries).HandlerName()
	}
	new_s, err := NewTimeSeries(ts1.Name(), start, opts)
	if err != nil {
		return nil, err
	}

	var s1 *repr.StatRepr = nil
	var s2 *repr.StatRepr = nil

	for t := int64(0); t <= endTime; t += nano_step {
		dp := &repr.StatRepr{}
		// loop through the orig data until we hit a time > then the current one
		for iter1.Next() {
			// need to merge the last point in if we have it
			if s1 != nil {
				dp.Merge(s1)
				s1 = nil
			}
			s_repr := iter1.ReprValue()
			if s_repr.Time > t {
				s1 = s_repr
				break
			}
			dp.Merge(s1)

		}
		for iter2.Next() {
			// need to merge the last point in if we have it
			if s2 != nil {
				dp.Merge(s2)
				s1 = nil
			}
			s_repr := iter2.ReprValue()
			if s_repr.Time > t {
				s2 = s_repr
				break
			}
			dp.Merge(s2)
		}

		if dp.Time != 0 {
			new_s.AddStat(dp)
		}
	}
	return new_s, nil

}
