package series

import (
	"testing"
)

func Test_Gob(t *testing.T) {
	genericTestSeries(t, "binary", NewDefaultOptions())
}

func Benchmark_________________________Gob_Put(b *testing.B) {
	benchmarkSeriesPut(b, "binary", testDefaultByteSize)
}

func Benchmark_Gob____________________Random_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "binary")
}
func Benchmark_Gob_________Float_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kNonRandom(b, "binary")
}

func Benchmark_Gob___________Int_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kRandomInt(b, "binary")
}

func Benchmark_Gob_____________________Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "binary", testDefaultByteSize)
}

func Benchmark_Gob___________SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "binary", testDefaultByteSize)
}

func Benchmark_Gob_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "binary", testDefaultByteSize)
}

func Benchmark_Gob______________Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "binary", testDefaultByteSize)
}

func Benchmark_Gob_______________Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "binary", testDefaultByteSize)
}

func Benchmark_Gob_________________Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "binary", testDefaultByteSize)
}

func Benchmark_Gob_________________LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "binary", testDefaultByteSize)
}

func Benchmark_Gob______________________Reading(b *testing.B) {
	benchmarkSeriesReading(b, "binary", testDefaultByteSize)
}
