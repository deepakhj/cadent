package series

import (
	"testing"
)

func Test_GobSeries(t *testing.T) {
	genericTestSeries(t, "binary", NewDefaultOptions())
}

func Benchmark_GobSeries_Put(b *testing.B) {
	benchmarkSeriesPut(b, "binary", testDefaultByteSize)
}

func Benchmark_GobSeries_Random_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "binary")
}
func Benchmark_GobSeries_Float_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kNonRandom(b, "binary")
}

func Benchmark_GobSeries_Int_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kRandomInt(b, "binary")
}

func Benchmark_GobSeries_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "binary", testDefaultByteSize)
}

func Benchmark_GobSeries_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "binary", testDefaultByteSize)
}

func Benchmark_GobSeries_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "binary", testDefaultByteSize)
}

func Benchmark_GobSeries_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "binary", testDefaultByteSize)
}

func Benchmark_GobSeries_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "binary", testDefaultByteSize)
}

func Benchmark_GobSeries_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "binary", testDefaultByteSize)
}

func Benchmark_GobSeries_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "binary", testDefaultByteSize)
}
