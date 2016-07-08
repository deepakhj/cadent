package series

import (
	"bytes"
	"cadent/server/repr"
	"compress/flate"
	"fmt"
	"github.com/golang/snappy"
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"testing"
	"time"
)

func TestGorillaTimeSeries(t *testing.T) {

	Convey("GorillaSeries", t, func() {
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

		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		var err error
		n_stats := 10
		times, stats, err := addStats(ser, stat, n_stats)

		So(err, ShouldEqual, nil)
		So(ser.StartTime(), ShouldEqual, n.UnixNano())
		So(ser.LastTime(), ShouldEqual, times[len(times)-1])

		it, err := ser.Iter()
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
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
		ser_size := NewMultiGoriallaTimeSeries(n.UnixNano())
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

func BenchmarkGorillaSeriesPut(b *testing.B) {
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
		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		ser.AddStat(&stat)
	}
}

func BenchmarkGorillaSeriesSnappyCompress(b *testing.B) {
	b.ResetTimer()
	stat := repr.StatRepr{
		Sum:   repr.JsonFloat64(rand.Float64()),
		Min:   repr.JsonFloat64(rand.Float64()),
		Max:   repr.JsonFloat64(rand.Float64()),
		Last:  repr.JsonFloat64(rand.Float64()),
		Count: 4123123123123,
	}
	n := time.Now()
	stat.Time = n

	n_stat := 1024
	for i := 0; i < b.N; i++ {
		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		addStats(ser, stat, n_stat)
		data, _ := ser.MarshalBinary()
		pre_l := len(data)
		c_bss := snappy.Encode(nil, data)
		b.Logf("Data Len: Pre: %v bytes/Stat %v, Comp: %v bytes/stat %v", pre_l, len(c_bss)/n_stat, len(c_bss), len(c_bss)/n_stat)
	}
}

func BenchmarkGorillaSeriesFlateCompress(b *testing.B) {
	b.ResetTimer()
	stat := repr.StatRepr{
		Sum:   repr.JsonFloat64(rand.Float64()),
		Min:   repr.JsonFloat64(rand.Float64()),
		Max:   repr.JsonFloat64(rand.Float64()),
		Last:  repr.JsonFloat64(rand.Float64()),
		Count: 4123123123123,
	}
	n := time.Now()
	stat.Time = n

	n_stat := 1024
	for i := 0; i < b.N; i++ {
		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		addStats(ser, stat, n_stat)
		bss, _ := ser.MarshalBinary()
		c_bss := new(bytes.Buffer)
		zipper, _ := flate.NewWriter(c_bss, flate.BestSpeed)
		pre_l := len(bss)
		zipper.Write(bss)
		zipper.Flush()
		b.Logf("Flate Comp Data Len: Pre: %v,  %v bytes per stat %v", pre_l, c_bss.Len(), c_bss.Len()/n_stat)
	}
}
