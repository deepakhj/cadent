package series

import (
	"testing"
)

func Test_ZipGob_TimeSeries(t *testing.T) {
	genericTestSeries(t, "zipgob", NewDefaultOptions())
}

func Benchmark_ZipGob_Put(b *testing.B) {
	benchmarkSeriesPut(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "zipgob")
}

func Benchmark_ZipGob_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "zipgob", testDefaultByteSize)
}
