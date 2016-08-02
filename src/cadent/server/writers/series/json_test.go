package series

import (
	"testing"
)

func TestSimpleJsonTimeSeries(t *testing.T) {
	genericTestSeries(t, "json")
}

func Benchmark_JsonSeriesPut(b *testing.B) {
	benchmarkSeriesPut(b, "json", testDefaultByteSize)
}
func Benchmark_JsonSeries8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "json")
}
func Benchmark_JsonSeriesRawSize(b *testing.B) {
	benchmarkRawSize(b, "json", testDefaultByteSize)
}

func Benchmark_JsonSeriesSnappyCompress(b *testing.B) {
	benchmarkSnappyCompress(b, "json", testDefaultByteSize)
}

func Benchmark_JsonSeriesFlateCompress(b *testing.B) {
	benchmarkFlateCompress(b, "json", testDefaultByteSize)
}

func Benchmark_JsonSeriesZipCompress(b *testing.B) {
	benchmarkZipCompress(b, "json", testDefaultByteSize)
}

func Benchmark_JsonSeriesReading(b *testing.B) {
	benchmarkSeriesReading(b, "json", testDefaultByteSize)
}
