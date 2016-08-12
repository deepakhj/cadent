package series

import (
	"testing"
)

func Test_Json_TimeSeries(t *testing.T) {
	genericTestSeries(t, "json", NewDefaultOptions())
}

func Benchmark_Json_Put(b *testing.B) {
	benchmarkSeriesPut(b, "json", testDefaultByteSize)
}
func Benchmark_Json_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "json")
}
func Benchmark_Json_RawSize(b *testing.B) {
	benchmarkRawSize(b, "json", testDefaultByteSize)
}

func Benchmark_Json_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "json", testDefaultByteSize)
}

func Benchmark_Json_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "json", testDefaultByteSize)
}

func Benchmark_Json_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "json", testDefaultByteSize)
}

func Benchmark_Json_LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "json", testDefaultByteSize)
}
func Benchmark_Json_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "json", testDefaultByteSize)
}
