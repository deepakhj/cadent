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

func Test_Gorilla_Series(t *testing.T) {
	genericTestSeries(t, "gorilla", NewDefaultOptions())
}

func Benchmark_GorillaPut(b *testing.B) {
	benchmarkSeriesPut(b, "gorilla", testDefaultByteSize)
}

func Benchmark_GorillaPut_1Value_LowRes(b *testing.B) {
	b.ResetTimer()
	stat, n := dummyStat()
	ops := NewOptions(1, false)
	b.SetBytes(int64(8 * 8)) //8 64bit numbers
	for i := 0; i < b.N; i++ {
		ser, err := NewTimeSeries("gorilla", n.UnixNano(), ops)
		if err != nil {
			b.Fatalf("ERROR: %v", err)
		}
		ser.AddStat(stat)
	}
}

func Benchmark_Gorilla_Random_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "gorilla")
}

func Benchmark_Gorilla_Float_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kNonRandom(b, "gorilla")
}

func Benchmark_Gorilla_Int_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kRandomInt(b, "gorilla")
}

func Benchmark_Gorilla_RawSize(b *testing.B) {
	benchmarkRawSize(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "gorilla", testDefaultByteSize)
}

func Benchmark_GorrillaSeries_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "gorilla", testDefaultByteSize)
}

func Benchmark_Gorilla_Series_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "gorilla", testDefaultByteSize)
}
