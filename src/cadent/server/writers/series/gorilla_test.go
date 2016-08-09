package series

import (
	"testing"
)

func Test_Gorilla_Series(t *testing.T) {
	genericTestSeries(t, "gorilla", NewDefaultOptions())
}

func Benchmark_GorillaPut(b *testing.B) {
	benchmarkSeriesPut(b, "gorilla", testDefaultByteSize)
}

func Benchmark_GorillaPut_1Value_LowRes(b *testing.B) {
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

func Benchmark_Gorilla_Random_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "gorilla")
}

func Benchmark_Gorilla_Float_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kNonRandom(b, "gorilla")
}

func Benchmark_Gorilla_Int_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kRandomInt(b, "gorilla")
}

func Benchmark_Gorilla_RawSize(b *testing.B) {
	benchmarkRawSize(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "gorilla", testDefaultByteSize)
}

func Benchmark_GorrillaSeries_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_Series_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "gorilla", testDefaultByteSize)
}
