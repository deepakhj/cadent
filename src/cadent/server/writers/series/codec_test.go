package series

import (
	"testing"
)

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

func Benchmark_Binc_LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "binc", testDefaultByteSize)
}
func Benchmark_Binc_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "binc", testDefaultByteSize)
}

/* cbor */

func Test_Cbor_TimeSeries(t *testing.T) {
	opts := NewDefaultOptions()
	opts.Handler = "cbor"
	genericTestSeries(t, "cbor", opts)
}

func Benchmark_Cbor_Put(b *testing.B) {
	benchmarkSeriesPut(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "cbor")
}

func Benchmark_Cbor_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "cbor", testDefaultByteSize)
}
