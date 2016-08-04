package series

import (
	"testing"
)

func TestMsgPackArrayTimeSeries(t *testing.T) {
	genericTestSeries(t, "msgpack")
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
