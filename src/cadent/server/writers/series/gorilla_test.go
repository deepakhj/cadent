package series

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestGorillaTimeSeries(t *testing.T) {

	Convey("GorillaSeries", t, func() {
		stat, n := dummyStat()

		ser := NewGoriallaTimeSeries(n.UnixNano())
		var err error
		n_stats := 256
		times, stats, err := addStats(ser, stat, n_stats, true)

		So(err, ShouldEqual, nil)
		if ser.fullResolution {
			So(ser.StartTime(), ShouldEqual, n.UnixNano())
			So(ser.LastTime(), ShouldEqual, times[len(times)-1])
		} else {
			tt := time.Unix(0, ser.StartTime())
			So(tt.Unix(), ShouldEqual, n.Unix())
			lt := time.Unix(0, ser.LastTime())
			test_lt := time.Unix(0, times[len(times)-1])
			So(lt.Unix(), ShouldEqual, test_lt.Unix())
		}
		it, err := ser.Iter()
		So(err, ShouldEqual, nil)
		idx := 0
		for it.Next() {

			/*to, mi,_,_,_,_,_:= it.Values()
			t_second := time.Unix(0, times[idx]).Unix()
			t_o_second := time.Unix(0, to).Unix()
			t.Logf("%d: Data Comp: init: %v, diff: %v, init: %v, diff: %v", idx, t_second, t_o_second - t_second, stats[idx].Min, mi - float64(stats[idx].Min))
			idx++
			continue*/

			to, mi, mx, fi, ls, su, ct := it.Values()
			t_orig := times[idx]
			if !ser.fullResolution {
				t_orig = time.Unix(0, times[idx]).Unix()
				to = time.Unix(0, to).Unix()

			}
			So(t_orig, ShouldEqual, to)
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
	})

}

func Benchmark_GorillaSeriesPut(b *testing.B) {
	benchmarkSeriesPut(b, "gorilla", 1024)
}

func Benchmark_GorillaSeriesPut8k(b *testing.B) {
	benchmarkSeriesPut8k(b, "gorilla")
}

func Benchmark_GorillaSeriesRawSize(b *testing.B) {
	benchmarkRawSize(b, "gorilla", 1024)
}

func Benchmark_GorillaSeriesSnappyCompress(b *testing.B) {
	benchmarkSnappyCompress(b, "gorilla", 1024)
}

func Benchmark_GorillaSeriesFlateCompress(b *testing.B) {
	benchmarkFlateCompress(b, "gorilla", 1024)
}

func Benchmark_GorillaSeriesZipCompress(b *testing.B) {
	benchmarkZipCompress(b, "gorilla", 1024)
}

func Benchmark_GorillaSeriesReading(b *testing.B) {
	benchmarkSeriesReading(b, "gorilla", 1024)
}
