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

func Test_Msgpack________________________TimeSeries(t *testing.T) {
	genericTestSeries(t, "msgpack", NewDefaultOptions())
}

func Benchmark_Msgpack__________________________Put(b *testing.B) {
	benchmarkSeriesPut(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack___________________________8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "msgpack")
}

func Benchmark_Msgpack_____________________Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack___________SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack________________ZStd_Compress(b *testing.B) {
	benchmarkZStdCompress(b, "msgpack", testDefaultByteSize)
}
func Benchmark_Msgpack______________Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_______________Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_________________Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack_________________LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "msgpack", testDefaultByteSize)
}

func Benchmark_Msgpack______________________Reading(b *testing.B) {
	benchmarkSeriesReading(b, "msgpack", testDefaultByteSize)
}
