package series

import (
	"testing"
)

func Test_Msgpack_TimeSeries(t *testing.T) {
	genericTestSeries(t, "msgpack", NewDefaultOptions())
}

func Benchmark_Msgpack_Put(b *testing.B) {
	benchmarkSeriesPut(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "msgpack")
}

func Benchmark_Msgpack_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "msgpack", testDefaultByteSize)
}
