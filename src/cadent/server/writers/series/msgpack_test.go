package series

import (
	"testing"
)

func Test_MsgPack_TimeSeries(t *testing.T) {
	genericTestSeries(t, "msgpack", NewDefaultOptions())
}

func Benchmark_MsgPack_Put(b *testing.B) {
	benchmarkSeriesPut(b, "msgpack", testDefaultByteSize)
}

func Benchmark_MsgPack_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "msgpack")
}

func Benchmark_MsgPack_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "msgpack", testDefaultByteSize)
}

func Benchmark_MsgPack_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "msgpack", testDefaultByteSize)
}

func Benchmark_MsgPack_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "msgpack", testDefaultByteSize)
}

func Benchmark_MsgPack_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_MsgPack_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_MsgPack_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_MsgPack_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "msgpack", testDefaultByteSize)
}

/* binc */

func Test_Binc_TimeSeries(t *testing.T) {
	opts := NewDefaultOptions()
	opts.Handler = "binc"
	genericTestSeries(t, "binc", opts)
}

func Benchmark_Binc_Put(b *testing.B) {
	benchmarkSeriesPut(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "binc")
}

func Benchmark_Binc_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "binc", testDefaultByteSize)
}
