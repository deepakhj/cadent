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

func TestProtbuf_________________________TimeSeries(t *testing.T) {
	genericTestSeries(t, "protobuf", NewDefaultOptions())
}

func Benchmark_ProtoBuf_________________________Put(b *testing.B) {
	benchmarkSeriesPut(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf__________________________8k(b *testing.B) {
	benchmarkSeriesPut8kRandom(b, "protobuf")
}

func Benchmark_ProtoBuf____________________Raw_Size(b *testing.B) {
	benchmarkRawSize(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf__________SingleVal_Raw_Size(b *testing.B) {
	benchmarkRawSizeSingleStat(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtBuf_NonRandom_SingleVal_Raw_Size(b *testing.B) {
	benchmarkNonRandomRawSizeSingleStat(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf_______________ZStd_Compress(b *testing.B) {
	benchmarkZStdCompress(b, "protobuf", testDefaultByteSize)
}
func Benchmark_ProtoBuf_____________Snappy_Compress(b *testing.B) {
	benchmarkSnappyCompress(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf______________Flate_Compress(b *testing.B) {
	benchmarkFlateCompress(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf________________Zip_Compress(b *testing.B) {
	benchmarkZipCompress(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf________________LZW_Compress(b *testing.B) {
	benchmarkLZWCompress(b, "protobuf", testDefaultByteSize)
}

func Benchmark_ProtoBuf____________________Reading(b *testing.B) {
	benchmarkSeriesReading(b, "protobuf", testDefaultByteSize)
}
