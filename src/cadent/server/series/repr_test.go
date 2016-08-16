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

func Test_Repr(t *testing.T) {
	genericTestSeries(t, "repr", NewDefaultOptions())
}

func Benchmark_Repr_Put(b *testing.B) {
	benchmarkSeriesPut(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Random_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "repr")
}
func Benchmark_Repr_Float_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kNonRandom(b, "repr")
}

func Benchmark_Repr_Int_Incremental_8k(b *testing.B) {
	benchmarkSeriesPut8kRandomInt(b, "repr")
}

func Benchmark_Repr_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "repr", testDefaultByteSize)
}

func Benchmark_Repr_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "repr", testDefaultByteSize)
}
