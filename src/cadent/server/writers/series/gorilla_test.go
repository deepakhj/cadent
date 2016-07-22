package series

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestGorillaTimeSeries(t *testing.T) {

	Convey("GorillaSeries", t, func() {
		stat, n := dummyStat()

		ser := NewGoriallaTimeSeries(n.UnixNano(), NewOptions(6, true))
		var err error
		n_stats := 10
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
		So(it.(*GorillaIter).numValues, ShouldEqual, 6)
		idx := 0
		for it.Next() {

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
		So(it.Error(), ShouldEqual, nil)
		So(idx, ShouldEqual, n_stats)
	})

}

func Benchmark_GorillaSeriesPut(b *testing.B) {
	benchmarkSeriesPut(b, "gorilla", 1024)
}

func Benchmark_GorillaSeriesPut_1ValueSecond(b *testing.B) {
	b.ResetTimer()
	stat, n := dummyStat()
	ops := NewOptions(1, false)
	b.SetBytes(int64(8 * 8)) //8 64bit numbers
	for i := 0; i < b.N; i++ {
		ser, err := NewTimeSeries("gorilla", n.UnixNano(), ops)
		if err != nil {
			b.Fatalf("ERROR: %v", err)
		}
		ser.AddStat(stat)
	}
}

func Benchmark_GorillaSeries_Random_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "gorilla")
}
func Benchmark_GorillaSeries_Float_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kNonRandom(b, "gorilla")
}
func Benchmark_GorillaSeries_Int_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kRandomInt(b, "gorilla")
}

func Benchmark_GorillaSeries_RawSize(b *testing.B) {
	benchmarkRawSize(b, "gorilla", 1024)
}

func Benchmark_GorillaSeries_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "gorilla", 1024)
}

func Benchmark_GorillaSeries_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "gorilla", 1024)
}

func Benchmark_GorillaSeries_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "gorilla", 1024)
}

func Benchmark_GorillaSeries_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "gorilla", 1024)
}

func Benchmark_Gorilla_Series_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "gorilla", 1024)
}
