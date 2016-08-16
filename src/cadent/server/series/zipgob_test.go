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

func Test_ZipGob_TimeSeries(t *testing.T) {
	genericTestSeries(t, "zipgob", NewDefaultOptions())
}

func Benchmark_ZipGob_Put(b *testing.B) {
	benchmarkSeriesPut(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "zipgob")
}

func Benchmark_ZipGob_Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "zipgob", testDefaultByteSize)
}

func Benchmark_ZipGob_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "zipgob", testDefaultByteSize)
}
