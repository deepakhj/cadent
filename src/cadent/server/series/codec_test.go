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

/* binc */

func Test_Binc________________________TimeSeries(t *testing.T) {
	opts := NewDefaultOptions()
	opts.Handler = "binc"
	genericTestSeries(t, "binc", opts)
}

func Benchmark_Binc___________________________Put(b *testing.B) {
	benchmarkSeriesPut(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc____________________________8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "binc")
}

func Benchmark_Binc______________________Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc____________SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc__NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc_________________ZStd_Compress(b *testing.B) {
	benchmarkZStdCompress(b, "binc", testDefaultByteSize)
}
func Benchmark_Binc_______________Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc________________Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc__________________Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "binc", testDefaultByteSize)
}

func Benchmark_Binc__________________LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "binc", testDefaultByteSize)
}
func Benchmark_Binc_______________________Reading(b *testing.B) {
	benchmarkSeriesReading(b, "binc", testDefaultByteSize)
}

/* cbor */

func Test_Cbor_________________________TimeSeries(t *testing.T) {
	opts := NewDefaultOptions()
	opts.Handler = "cbor"
	genericTestSeries(t, "cbor", opts)
}

func Benchmark_Cbor___________________________Put(b *testing.B) {
	benchmarkSeriesPut(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor____________________________8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "cbor")
}

func Benchmark_Cbor______________________Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor____________SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor__NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_______________Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_________________ZStd_Compress(b *testing.B) {
	benchmarkZStdCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor________________Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor__________________Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor__________________LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "cbor", testDefaultByteSize)
}

func Benchmark_Cbor_______________________Reading(b *testing.B) {
	benchmarkSeriesReading(b, "cbor", testDefaultByteSize)
}
