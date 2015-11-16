// a simple atomic stat counter/rate
package stats

import (
	"time"
)

type StatCount struct {
	Name       string
	TotalCount AtomicInt
	TickCount  AtomicInt
}

func (stat *StatCount) Up(val uint64) {
	stat.TotalCount.Add(1)
	stat.TickCount.Add(1)
}
func (stat *StatCount) ResetTick() {
	stat.TickCount.Set(0)
}
func (stat *StatCount) Rate(duration time.Duration) float32 {
	if stat.TickCount == 0 {
		return 0
	}
	return float32(stat.TickCount) / float32(duration/time.Second)
}

func (stat *StatCount) TotalRate(duration time.Duration) float32 {
	if stat.TotalCount == 0 {
		return 0
	}
	return float32(stat.TotalCount) / float32(duration/time.Second)
}
func (stat *StatCount) Reset() {
	stat.TickCount.Set(0)
	stat.TotalCount.Set(0)
}
