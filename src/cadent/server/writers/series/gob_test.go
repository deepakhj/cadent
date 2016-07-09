package series

import (
	"testing"
)

func Test_GobSeries(t *testing.T) {
	genericTestSeries(t, "binary")
}

func Benchmark_GobSeriesPut(b *testing.B) {
	benchmarkSeriesPut(b, "binary", 1024)
}

func Benchmark_GobSeries8k(b *testing.B) {
	benchmarkSeriesPut8k(b, "binary")
}

func Benchmark_GobSeriesRawSize(b *testing.B) {
	benchmarkRawSize(b, "binary", 1024)
}

func Benchmark_GobSeriesSnappyCompress(b *testing.B) {
	benchmarkSnappyCompress(b, "binary", 1024)
}

func Benchmark_GobSeriesFlateCompress(b *testing.B) {
	benchmarkFlateCompress(b, "binary", 1024)
}

func Benchmark_GobSeriesZipCompress(b *testing.B) {
	benchmarkZipCompress(b, "binary", 1024)
}

func Benchmark_GobSeriesReading(b *testing.B) {
	benchmarkSeriesReading(b, "binary", 1024)
}
