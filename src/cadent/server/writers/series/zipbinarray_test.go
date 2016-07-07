package series

import (
	"cadent/server/repr"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"testing"
	"time"
)

func TestZipSimpleBinaryTimeSeries(t *testing.T) {

	Convey("ZipSimpleTimeSeries", t, func() {
		stat := repr.StatRepr{
			Key:        "goo",
			StatKey:    "goo",
			Sum:        5,
			Min:        1,
			Max:        repr.JsonFloat64(rand.Float64()),
			Last:       repr.JsonFloat64(rand.Float64()),
			Count:      4123123123123,
			Resolution: 2,
		}
		n := time.Now()
		stat.Time = n

		ser := NewZipSimpleBinaryTimeSeries(n.UnixNano())
		times := make([]int64, 0)
		stats := make([]*repr.StatRepr, 0)
		var err error
		n_stats := 10
		for i := 0; i < n_stats; i++ {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", rand.Int31n(10)))
			stat.Time = stat.Time.Add(dd)
			stat.Max = repr.JsonFloat64(rand.Float64())
			stat.Sum = repr.JsonFloat64(float64(stat.Sum) + float64(i))
			times = append(times, stat.Time.UnixNano())
			stats = append(stats, stat.Copy())
			err = ser.AddStat(&stat)
			if err != nil {
				t.Logf("ERROR: %v", err)

			}
		}
		So(err, ShouldEqual, nil)
		So(ser.StartTime(), ShouldEqual, n.UnixNano())
		So(ser.LastTime(), ShouldEqual, times[len(times)-1])

		it, err := ser.Iter()
		t.Logf("Error: %v", err)
		idx := 0
		for it.Next() {
			to, mi, mx, fi, ls, su, ct := it.Values()
			So(times[idx], ShouldEqual, to)
			So(stats[idx].Max, ShouldEqual, mx)
			So(stats[idx].Last, ShouldEqual, ls)
			So(stats[idx].Count, ShouldEqual, ct)
			So(stats[idx].First, ShouldEqual, fi)
			So(stats[idx].Min, ShouldEqual, mi)
			So(stats[idx].Sum, ShouldEqual, su)

			//t.Logf("%d: Time ok: %v", idx, times[idx] == to)
			r := it.ReprValue()

			So(stats[idx].Sum, ShouldEqual, r.Sum)
			So(stats[idx].Min, ShouldEqual, r.Min)
			So(stats[idx].Max, ShouldEqual, r.Max)

			//t.Logf("BIT Repr: %v", r)
			idx++
		}
		So(idx, ShouldEqual, n_stats)

		t.Logf("Error: %v", it.Error())
		t.Logf("Data Len: %v", ser.Len())

		// see how much a 8kb blob holds
		max_size := 8 * 1024
		ser_size := NewZipSimpleBinaryTimeSeries(n.UnixNano())
		on_tick := 0
		for true {
			on_tick++
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", on_tick))
			stat.Time = stat.Time.Add(dd)
			stat.Max = repr.JsonFloat64(rand.Float64())
			stat.Sum = repr.JsonFloat64(float64(stat.Sum) + float64(on_tick))
			times = append(times, stat.Time.UnixNano())
			stats = append(stats, stat.Copy())
			err = ser_size.AddStat(&stat)
			if err != nil {
				t.Logf("ERROR: %v", err)
			}
			if ser_size.Len() >= max_size {
				t.Logf("Max Stats in Buffer for %d bytes is %d", max_size, on_tick)
				break
			}
		}
	})

}

func BenchmarkBinaryZipSeriesPut(b *testing.B) {
	b.ResetTimer()
	stat := repr.StatRepr{
		Sum:        5,
		Min:        1,
		Max:        repr.JsonFloat64(rand.Float64()),
		Last:       repr.JsonFloat64(rand.Float64()),
		Count:      4123123123123,
		Resolution: 2,
	}
	n := time.Now()
	stat.Time = n

	for i := 0; i < b.N; i++ {
		ser := NewSimpleJsonTimeSeries(n.UnixNano())
		ser.AddStat(&stat)
	}
}
