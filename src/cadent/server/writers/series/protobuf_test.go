package series

import (
	"testing"
)

func TestProtbufArrayTimeSeries(t *testing.T) {
	genericTestSeries(t, "protobuf")
}

func Benchmark_ProtoBufPut(b *testing.B) {
	benchmarkSeriesPut(b, "protobuf", 1024)
}

func Benchmark_ProtoBuf8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "protobuf")
}

func Benchmark_ProtoBufRawSize(b *testing.B) {
	benchmarkRawSize(b, "protobuf", 1024)
}

func Benchmark_ProtoBufSnappyCompress(b *testing.B) {
	benchmarkSnappyCompress(b, "protobuf", 1024)
}

func Benchmark_ProtoBufFlateCompress(b *testing.B) {
	benchmarkFlateCompress(b, "protobuf", 1024)
}

func Benchmark_ProtoBufZipCompress(b *testing.B) {
	benchmarkZipCompress(b, "protobuf", 1024)
}

func Benchmark_ProtoBufReading(b *testing.B) {
	benchmarkSeriesReading(b, "protobuf", 1024)
}
