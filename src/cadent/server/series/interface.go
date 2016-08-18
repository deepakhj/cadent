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
)

type TimeSeries interface {
	UnmarshalBinary([]byte) error
	AddPoint(t int64, min float64, max float64, last float64, sum float64, count int64) error
	AddStat(*repr.StatRepr) error

	//NOTE these two methods render the write buffer dead, and thus the "time series" complete
	// use "IterClone" or "ByteClone" to get the current
	// read buffer and not effect the write buffer
	Iter() (TimeSeriesIter, error)
	MarshalBinary() ([]byte, error)

	// grab the current buffer, which has the side effect of having to "re-write" a new buffer
	// in the internal object from the clone, so there is a penalty for this
	Bytes() []byte

	// num points in the mix
	Count() int

	Len() int
	StartTime() int64
	LastTime() int64
	HighResolution() bool
	Name() string
}

type TimeSeriesIter interface {
	Next() bool
	Values() (int64, float64, float64, float64, float64, int64)
	ReprValue() *repr.StatRepr
	Error() error
}
