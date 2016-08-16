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

func Test_Json_TimeSeries(t *testing.T) {
	genericTestSeries(t, "json", NewDefaultOptions())
}

func Benchmark_Json_Put(b *testing.B) {
	benchmarkSeriesPut(b, "json", testDefaultByteSize)
}
func Benchmark_Json_8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "json")
}
func Benchmark_Json_RawSize(b *testing.B) {
	benchmarkRawSize(b, "json", testDefaultByteSize)
}

func Benchmark_Json_Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "json", testDefaultByteSize)
}

func Benchmark_Json_Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "json", testDefaultByteSize)
}

func Benchmark_Json_Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "json", testDefaultByteSize)
}

func Benchmark_Json_LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "json", testDefaultByteSize)
}
func Benchmark_Json_Reading(b *testing.B) {
	benchmarkSeriesReading(b, "json", testDefaultByteSize)
}
