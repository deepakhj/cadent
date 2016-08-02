package series

import (
	"testing"
)

func TestProtbufArrayTimeSeries(t *testing.T) {
	genericTestSeries(t, "protobuf")
}

func Benchmark_ProtoBuf_Put(b *testing.B) {
	benchmarkSeriesPut(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "protobuf")
}

func Benchmark_ProtoBuf_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtBuf_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBufSnappyCompress(b *testing.B) {
	benchmarkSnappyCompress(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBufFlateCompress(b *testing.B) {
	benchmarkFlateCompress(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBufZipCompress(b *testing.B) {
	benchmarkZipCompress(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBufReading(b *testing.B) {
	benchmarkSeriesReading(b, "protobuf", testDefaultByteSize)
}
