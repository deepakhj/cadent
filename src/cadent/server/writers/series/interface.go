/*
	Interface bits for time series
*/

package series

import (
	"cadent/server/repr"
)

type TimeSeries interface {
	UnmarshalBinary([]byte) error
	AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error
	AddStat(*repr.StatRepr) error
	Iter() (TimeSeriesIter, error)
	MarshalBinary() ([]byte, error)
	Len() int
	StartTime() int64
	LastTime() int64
}

type TimeSeriesIter interface {
	Next() bool
	Values() (int64, float64, float64, float64, float64, float64, int64)
	ReprValue() *repr.StatRepr
	Error() error
}
