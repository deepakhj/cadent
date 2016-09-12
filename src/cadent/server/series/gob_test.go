/*
Copyright 2016 Under Armour, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

func Benchmark_GOB_______________ZStd_Compress(b *testing.B) {
	benchmarkZStdCompress(b, "binary", testDefaultByteSize)
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
