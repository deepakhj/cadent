package series

import (
	"testing"
)

func Test_ReprSeries(t *testing.T) {
	genericTestSeries(t, "repr", NewDefaultOptions())
}

func Benchmark_ReprSeries_Put(b *testing.B) {
	benchmarkSeriesPut(b, "repr", testDefaultByteSize)
}

func Benchmark_ReprSeries_Random_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "repr")
}
func Benchmark_ReprSeries_Float_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kNonRandom(b, "repr")
}

func Benchmark_ReprSeries_Int_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kRandomInt(b, "repr")
}

func Benchmark_ReprSeries_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "repr", testDefaultByteSize)
}

func Benchmark_ReprSeries_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_ReprSeries_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_ReprSeries_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_ReprSeries_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "repr", testDefaultByteSize)
}
