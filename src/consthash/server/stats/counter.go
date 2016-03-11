// a simple atomic stat counter/rate
package stats

import (
	"time"
)

type StatCount struct {
	Name       string
	TotalCount *AtomicInt
	TickCount  *AtomicInt
}

func NewStatCount(name string) StatCount {
	return StatCount{
		Name:       name,
		TotalCount: NewAtomic(name + "-TotalCount"),
		TickCount:  NewAtomic(name + "-TickCount"),
	}
}
func (stat *StatCount) Up(val uint64) {
	stat.TotalCount.Add(int64(val))
	stat.TickCount.Add(int64(val))
}
func (stat *StatCount) ResetTick() {
	stat.TickCount.Set(int64(0))
}
func (stat *StatCount) Rate(duration time.Duration) float32 {
	if stat.TickCount.Equal(int64(0)) {
		return 0
	}
	return float32(stat.TickCount.Get()) / float32(duration/time.Second)
}

func (stat *StatCount) TotalRate(duration time.Duration) float32 {
	if stat.TotalCount.Equal(int64(0)) {
		return 0.0
	}
	return float32(stat.TotalCount.Get()) / float32(duration/time.Second)
}
func (stat *StatCount) Reset() {
	stat.TickCount.Set(int64(0))
	stat.TotalCount.Set(int64(0))
}
