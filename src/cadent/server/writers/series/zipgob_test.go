package series

import (
	"testing"
)

func TestZipSimpleBinaryTimeSeries(t *testing.T) {
	genericTestSeries(t, "zipgob", NewDefaultOptions())
}

func Benchmark_ZipGobPut(b *testing.B) {
	benchmarkSeriesPut(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "zipgob")
}

func Benchmark_ZipGobRawSize(b *testing.B) {
	benchmarkRawSize(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGobSnappyCompress(b *testing.B) {
	benchmarkSnappyCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGobFlateCompress(b *testing.B) {
	benchmarkFlateCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGobZipCompress(b *testing.B) {
	benchmarkZipCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGobReading(b *testing.B) {
	benchmarkSeriesReading(b, "zipgob", testDefaultByteSize)
}
