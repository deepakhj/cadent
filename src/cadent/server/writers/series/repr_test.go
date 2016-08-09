package series

import (
	"testing"
)

func Test_Repr(t *testing.T) {
	genericTestSeries(t, "repr", NewDefaultOptions())
}

func Benchmark_Repr_Put(b *testing.B) {
	benchmarkSeriesPut(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Random_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "repr")
}
func Benchmark_Repr_Float_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kNonRandom(b, "repr")
}

func Benchmark_Repr_Int_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kRandomInt(b, "repr")
}

func Benchmark_Repr_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "repr", testDefaultByteSize)
}
