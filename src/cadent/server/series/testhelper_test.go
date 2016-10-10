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

/*
	some helper generics for the testing ond benchmarking of the verios timeseries bits
*/

package series

import (
	"bytes"
	"cadent/server/repr"
	"compress/flate"
	"compress/lzw"
	"compress/zlib"
	"fmt"
	"github.com/DataDog/zstd"
	"github.com/golang/snappy"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

var testDefaultByteSize = 8192

func init() {
	rand.Seed(time.Now().UnixNano())
}

func dummyStat() (*repr.StatRepr, int64) {
	stat := &repr.StatRepr{
		Sum:   rand.Float64(),
		Min:   rand.Float64(),
		Max:   rand.Float64(),
		Last:  rand.Float64(),
		Count: 123123123,
	}
	n := time.Now().UnixNano()
	stat.Time = n
	return stat, n
}

func dummyStatSingleVal() (*repr.StatRepr, int64) {
	stat := &repr.StatRepr{
		Sum:   rand.Float64(),
		Count: 1,
	}
	n := time.Now().UnixNano()
	stat.Time = n
	return stat, n
}

func dummyStatInt() (*repr.StatRepr, int64) {
	stat := &repr.StatRepr{
		Sum:   float64(rand.Int63n(100)),
		Min:   float64(rand.Int63n(100)),
		Max:   float64(rand.Int63n(100)),
		Last:  float64(rand.Int63n(100)),
		Count: rand.Int63n(100),
	}
	n := time.Now().UnixNano()
	stat.Time = n
	return stat, n
}

func getSeries(stat *repr.StatRepr, num_s int, randomize bool) ([]int64, []*repr.StatRepr, error) {
	times := make([]int64, 0)
	stats := make([]*repr.StatRepr, 0)
	var err error
	for i := 0; i < num_s; i++ {

		if randomize {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", rand.Int31n(10)+1))
			stat.Time = stat.Time + int64(dd.Seconds())
			stat.Max = rand.Float64() * 20000.0
			stat.Min = rand.Float64() * 20000.0
			stat.Last = rand.Float64() * 20000.0
			stat.Sum = float64(stat.Sum) + float64(i)
			stat.Count = rand.Int63n(1000)
		} else {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", 1+i))
			stat.Time = stat.Time + int64(dd.Seconds())
			stat.Max += float64(1 + i)
			stat.Min += float64(1 + i)
			stat.Last += float64(1 + i)
			stat.Sum += float64(1 + i)
			stat.Count += int64(1 + i)
		}
		times = append(times, stat.Time)
		stats = append(stats, stat.Copy())
	}
	return times, stats, err
}

func addStats(ser TimeSeries, stat *repr.StatRepr, num_s int, randomize bool) ([]int64, []*repr.StatRepr, error) {
	times := make([]int64, 0)
	stats := make([]*repr.StatRepr, 0)
	var err error
	for i := 0; i < num_s; i++ {

		if randomize {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", rand.Int31n(10)+1))
			stat.Time = stat.Time + int64(dd.Seconds())
			stat.Max = rand.Float64() * 20000.0
			stat.Min = rand.Float64() * 20000.0
			stat.Last = rand.Float64() * 20000.0
			stat.Sum = float64(stat.Sum) + float64(i)
			stat.Count = rand.Int63n(1000)
		} else {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", 1+i))
			stat.Time = stat.Time + int64(dd.Seconds())
			stat.Max += float64(1 + i)
			stat.Min += float64(1 + i)
			stat.Last += float64(1 + i)
			stat.Sum += float64(1 + i)
			stat.Count += int64(1 + i)
		}
		times = append(times, stat.Time)
		stats = append(stats, stat.Copy())
		err = ser.AddStat(stat.Copy())
		if err != nil {
			return times, stats, err
		}
	}
	return times, stats, nil
}

func addSingleValStat(ser TimeSeries, stat *repr.StatRepr, num_s int, randomize bool) ([]int64, []*repr.StatRepr, error) {
	times := make([]int64, 0)
	stats := make([]*repr.StatRepr, 0)
	var err error
	for i := 0; i < num_s; i++ {

		if randomize {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", rand.Int31n(10)+1))
			stat.Time = stat.Time + int64(dd.Seconds())
			stat.Sum = float64(stat.Sum) + float64(i)
		} else {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", 1+i))
			stat.Time = stat.Time + int64(dd.Seconds())
			stat.Sum += float64(1 + i)
		}
		times = append(times, stat.Time)
		stats = append(stats, stat.Copy())
		err = ser.AddStat(stat)
		if err != nil {
			return times, stats, err
		}
	}
	return times, stats, nil
}

func _benchReset(b *testing.B) {
	runtime.GC()
	b.ResetTimer()
	b.ReportAllocs()
}

func benchmarkRawSize(b *testing.B, stype string, n_stat int) {

	stat, n := dummyStat()
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64bit numbers
	runs := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	_benchReset(b)
	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addStats(ser, stat, n_stat, true)
		bss, _ := ser.MarshalBinary()
		pre_len += int64(len(bss))
		b_per_stat += int64(int64(len(bss)) / int64(n_stat))
		runs++
	}
	b.Logf("Raw Size for %v stats: %v Bytes per stat: %v", n_stat, pre_len/runs, b_per_stat/runs)

}

func benchmarkRawSizeSingleStat(b *testing.B, stype string, n_stat int) {
	stat, n := dummyStatSingleVal()
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64bit numbers
	runs := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	_benchReset(b)

	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addSingleValStat(ser, stat, n_stat, true)
		bss, _ := ser.MarshalBinary()
		pre_len += int64(len(bss))
		b_per_stat += int64(int64(len(bss)) / int64(n_stat))
		runs++
	}
	b.Logf("Single Val (%v runs): Raw Size for %v stats: %v Bytes per stat: %v", runs, n_stat, pre_len/runs, b_per_stat/runs)

}

func benchmarkNonRandomRawSizeSingleStat(b *testing.B, stype string, n_stat int) {
	stat, n := dummyStatSingleVal()
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64bit numbers
	runs := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	_benchReset(b)

	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addSingleValStat(ser, stat, n_stat, false)
		bss, _ := ser.MarshalBinary()
		pre_len += int64(len(bss))
		b_per_stat += int64(int64(len(bss)) / int64(n_stat))
		runs++
	}
	b.Logf("NonRandom Single Val: Raw Size for %v stats: %v Bytes per stat: %v", n_stat, pre_len/runs, b_per_stat/runs)

}

func benchmarkFlateCompress(b *testing.B, stype string, n_stat int) {
	stat, n := dummyStat()
	runs := int64(0)
	comp_len := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64byte numbers
	_benchReset(b)

	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addStats(ser, stat, n_stat, true)
		bss, _ := ser.MarshalBinary()
		c_bss := new(bytes.Buffer)
		zipper, _ := flate.NewWriter(c_bss, flate.BestSpeed)
		zipper.Write(bss)
		zipper.Flush()

		pre_len += int64(len(bss))
		c_len := int64(c_bss.Len())
		comp_len += int64(c_len)
		b_per_stat += int64(c_len / int64(n_stat))
		runs++
	}
	b.Logf("Flate: Average PreComp size: %v, Post: %v Bytes per stat: %v", pre_len/runs, comp_len/runs, b_per_stat/runs)
}

func benchmarkZipCompress(b *testing.B, stype string, n_stat int) {

	stat, n := dummyStat()
	runs := int64(0)
	comp_len := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64byte numbers
	_benchReset(b)

	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addStats(ser, stat, n_stat, true)
		bss, _ := ser.MarshalBinary()
		c_bss := new(bytes.Buffer)
		zipper := zlib.NewWriter(c_bss)
		zipper.Write(bss)
		zipper.Flush()

		pre_len += int64(len(bss))
		c_len := int64(c_bss.Len())
		comp_len += int64(c_len)
		b_per_stat += int64(c_len / int64(n_stat))
		runs++
	}
	b.Logf("Zip: Average PreComp size: %v, Post: %v Bytes per stat: %v", pre_len/runs, comp_len/runs, b_per_stat/runs)
}
func benchmarkSnappyCompress(b *testing.B, stype string, n_stat int) {

	stat, n := dummyStat()
	runs := int64(0)
	comp_len := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64byte numbers
	_benchReset(b)

	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addStats(ser, stat, n_stat, true)
		bss, _ := ser.MarshalBinary()
		c_bss := snappy.Encode(nil, bss)

		pre_len += int64(len(bss))
		c_len := int64(len(c_bss))

		comp_len += int64(c_len)
		b_per_stat += int64(c_len / int64(n_stat))
		runs++
	}
	b.Logf("Snappy: Average PreComp size: %v, Post: %v Bytes per stat: %v", pre_len/runs, comp_len/runs, b_per_stat/runs)

}

func benchmarkZStdCompress(b *testing.B, stype string, n_stat int) {

	stat, n := dummyStat()
	runs := int64(0)
	comp_len := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64byte numbers
	_benchReset(b)

	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addStats(ser, stat, n_stat, true)
		bss, _ := ser.MarshalBinary()
		c_bss := new(bytes.Buffer)

		zipper := zstd.NewWriterLevel(c_bss, zstd.BestCompression)
		zipper.Write(bss)
		zipper.Close()

		pre_len += int64(len(bss))
		c_len := int64(c_bss.Len())
		comp_len += int64(c_len)
		b_per_stat += int64(c_len / int64(n_stat))
		runs++
	}
	b.Logf("Zstd: Average PreComp size: %v, Post: %v Bytes per stat: %v", pre_len/runs, comp_len/runs, b_per_stat/runs)

}

func benchmarkLZWCompress(b *testing.B, stype string, n_stat int) {

	stat, n := dummyStat()
	runs := int64(0)
	comp_len := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64byte numbers
	_benchReset(b)

	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addStats(ser, stat, n_stat, true)
		bss, _ := ser.MarshalBinary()
		c_bss := new(bytes.Buffer)
		zipper := lzw.NewWriter(c_bss, lzw.MSB, 8)
		zipper.Write(bss)
		zipper.Close()

		pre_len += int64(len(bss))
		c_len := int64(c_bss.Len())
		comp_len += int64(c_len)
		b_per_stat += int64(c_len / int64(n_stat))
		runs++
	}
	b.Logf("LZW: Average PreComp size: %v, Post: %v Bytes per stat: %v", pre_len/runs, comp_len/runs, b_per_stat/runs)
}

func benchmarkSeriesReading(b *testing.B, stype string, n_stat int) {

	stat, n := dummyStat()

	runs := int64(0)
	reads := int64(0)
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64bit numbers

	_benchReset(b)

	for i := 0; i < b.N; i++ {

		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		addStats(ser, stat, n_stat, true)
		it, err := ser.Iter()
		if err != nil {
			b.Fatalf("Error: %v", err)
		}
		for it.Next() {
			it.Values()
			reads++
		}
		if it.Error() != nil {
			b.Fatalf("Error: %v", it.Error())
		}
		runs++
	}
	b.Logf("Reads per run %v (so basically multiple the ns/op by this number)", reads/runs)

}

func benchmarkSeriesPut(b *testing.B, stype string, n_stat int) {
	stat, n := dummyStat()
	b.SetBytes(int64(8 * 8)) //8 64bit numbers
	_benchReset(b)
	for i := 0; i < b.N; i++ {
		ser, err := NewTimeSeries(stype, n, NewDefaultOptions())
		if err != nil {
			b.Fatalf("ERROR: %v", err)
		}
		ser.AddStat(stat)
	}
}

func benchmarkSeriesPut8kRandom(b *testing.B, stype string) {

	// see how much a 8kb blob holds
	runs := int64(0)
	stat_ct := int64(0)
	max_size := 8 * 1024
	b.SetBytes(int64(max_size))

	_benchReset(b)
	for i := 0; i < b.N; i++ {

		stat, n := dummyStat()
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		for true {
			stat_ct++
			_, _, err := addStats(ser, stat, 1, true)
			if err != nil {
				b.Logf("ERROR: %v", err)
			}
			if ser.Len() >= max_size {
				break
			}
		}
		runs++
	}
	b.Logf("Random Max Stats in Buffer for %d bytes is %d (%d per stat)", max_size, stat_ct/runs, int64(max_size)*runs/stat_ct)

}

func benchmarkSeriesPut8kNonRandom(b *testing.B, stype string) {

	// see how much a 8kb blob holds for "not-so-random" but incremental stats
	runs := int64(0)
	stat_ct := int64(0)
	max_size := 8 * 1024
	b.SetBytes(int64(max_size))

	_benchReset(b)
	for i := 0; i < b.N; i++ {

		stat, n := dummyStat()
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		for true {
			stat_ct++
			_, _, err := addStats(ser, stat, 1, false)
			if err != nil {
				b.Logf("ERROR: %v", err)
			}
			if ser.Len() >= max_size {
				break
			}
		}
		runs++
	}
	b.Logf("Incremental Max Stats in Buffer for %d bytes is %d (%d per stat)", max_size, stat_ct/runs, int64(max_size)*runs/stat_ct)

}

func benchmarkSeriesPut8kRandomInt(b *testing.B, stype string) {

	// see how much a 8kb blob holds for "not-so-random" but incremental stats
	runs := int64(0)
	stat_ct := int64(0)
	max_size := 8 * 1024
	b.SetBytes(int64(max_size))

	_benchReset(b)
	for i := 0; i < b.N; i++ {

		stat, n := dummyStatInt()
		ser, _ := NewTimeSeries(stype, n, NewDefaultOptions())
		for true {
			stat_ct++
			_, _, err := addStats(ser, stat, 1, false)
			if err != nil {
				b.Logf("ERROR: %v", err)
			}
			if ser.Len() >= max_size {
				break
			}
		}
		runs++
	}
	b.Logf("Incremental Max Stats in Buffer for %d bytes is %d (%d per stat)", max_size, stat_ct/runs, int64(max_size)*runs/stat_ct)

}

func genericTestSeries(t *testing.T, stype string, options *Options) {
	highres_opts := options
	highres_opts.HighTimeResolution = true
	lowres_opts := *options
	lowres_opts.HighTimeResolution = false

	opts := []*Options{highres_opts, &lowres_opts}

	for _, opt := range opts {
		nm := "highres"
		if opt == &lowres_opts {
			nm = "lowres"
		}
		Convey("Series Type : "+stype+" : "+nm, t, func() {
			stat, n := dummyStat()

			ser, _ := NewTimeSeries(stype, n, opt)
			n_stats := 10
			times, stats, err := addStats(ser, stat, n_stats, true)
			So(err, ShouldEqual, nil)

			s_time_test := n
			e_time_test := times[len(times)-1]
			if !ser.HighResolution() {
				tt, _ := splitNano(s_time_test)
				s_time_test = int64(tt)

				tt, _ = splitNano(e_time_test)
				e_time_test = int64(tt)

				s_time := ser.StartTime()
				tt, _ = splitNano(s_time)

				So(tt, ShouldEqual, s_time_test)

				e_time := ser.LastTime()
				tt, _ = splitNano(e_time)

				So(tt, ShouldEqual, e_time_test)

			} else {

				So(ser.StartTime(), ShouldEqual, s_time_test)
				So(ser.LastTime(), ShouldEqual, e_time_test)
			}

			it, err := ser.Iter()
			if err != nil {
				t.Fatalf("Iterator Error: %v", err)
			}
			idx := 0
			for it.Next() {
				to, mi, mx, ls, su, ct := it.Values()
				// just the second portion test
				test_to := time.Unix(0, to)
				test_lt := time.Unix(0, times[idx])
				//t.Logf("VALS: %v %v %v %v %v %v %v", to, mi, mx, fi, ls, su, ct)
				if ser.HighResolution() {
					So(test_lt.UnixNano(), ShouldEqual, test_to.UnixNano())
				} else {
					So(test_lt.Unix(), ShouldEqual, test_to.Unix())
				}
				So(stats[idx].Max, ShouldEqual, mx)
				So(stats[idx].Last, ShouldEqual, ls)
				So(stats[idx].Count, ShouldEqual, ct)
				So(stats[idx].Min, ShouldEqual, mi)
				So(stats[idx].Sum, ShouldEqual, su)

				//t.Logf("%d: Time ok: %v", idx, times[idx] == to)
				r := it.ReprValue()

				So(stats[idx].Sum, ShouldEqual, r.Sum)
				So(stats[idx].Min, ShouldEqual, r.Min)
				So(stats[idx].Max, ShouldEqual, r.Max)

				//t.Logf("BIT Repr: %v", r)
				idx++
			}
			if it.Error() != nil {
				t.Fatalf("Iter Error: %v", it.Error())
			}
			So(idx, ShouldEqual, n_stats)

			// compression, iterator R/W test
			bss := ser.Bytes()
			if err != nil {
				t.Fatalf("ERROR: %v", err)
			}
			t.Logf("Data size: %v", len(bss))

			Convey("Series Type: "+stype+" - Multi Iterator - "+nm, func() {
				run := 0
				max_run := 10
				dd, _ := dummyStat()
				for {
					ser.AddStat(dd)
					it, _ := ser.Iter()
					idx := 0
					for it.Next() {
						idx++

					}
					So(idx, ShouldEqual, n_stats+run+1)
					run++
					if run > max_run {
						break
					}
				}
			})

			Convey("Series Type: "+stype+" - 'Smart' Single Point - "+nm, func() {

				n_dd, nn := dummyStatSingleVal()
				nser, _ := NewTimeSeries(stype, nn, opt)
				n_times, o_stats, terr := addSingleValStat(nser, n_dd, n_stats, true)
				So(terr, ShouldEqual, nil)
				nit, _ := nser.Iter()
				idx := 0
				for nit.Next() {
					to, _, _, _, su, ct := nit.Values()

					test_to := time.Unix(0, to)
					test_lt := time.Unix(0, n_times[idx])
					if ser.HighResolution() {
						So(test_lt.UnixNano(), ShouldEqual, test_to.UnixNano())
					} else {
						So(test_lt.Unix(), ShouldEqual, test_to.Unix())
					}
					So(o_stats[idx].Sum, ShouldEqual, su)
					So(o_stats[idx].Count, ShouldEqual, ct)
					idx++
				}
				So(idx, ShouldEqual, n_stats)
				if nit.Error() != nil {
					t.Fatalf("Iter Error: %v", nit.Error())
				}
			})

			Convey("Series Type: "+stype+" - Snappy/Iterator - "+nm, func() {
				// Snappy tests
				c_bss := snappy.Encode(nil, bss)
				t.Logf("Snappy Data size: %v", len(c_bss))

				outs := make([]byte, 0)
				dec, err := snappy.Decode(outs, c_bss)
				t.Logf("Snappy Decode Data size: %v", len(dec))
				if err != nil {
					t.Fatalf("Error: %v", err)
				}

				n_iter, err := NewIter(stype, dec)
				if err != nil {
					t.Fatalf("Error: %v", err)
				}
				idx = 0
				for n_iter.Next() {
					to, mi, mx, ls, su, ct := n_iter.Values()
					test_to := time.Unix(0, to)
					test_lt := time.Unix(0, times[idx])
					if ser.HighResolution() {
						So(test_lt.UnixNano(), ShouldEqual, test_to.UnixNano())
					} else {
						So(test_lt.Unix(), ShouldEqual, test_to.Unix())
					}
					So(stats[idx].Max, ShouldEqual, mx)
					So(stats[idx].Last, ShouldEqual, ls)
					So(stats[idx].Count, ShouldEqual, ct)
					So(stats[idx].Min, ShouldEqual, mi)
					So(stats[idx].Sum, ShouldEqual, su)
					idx++
				}
			})
			bss = ser.Bytes()

			Convey("Series Type: "+stype+" - Zip/Iterator - "+nm, func() {
				// zip tests
				c_bss := new(bytes.Buffer)
				zipper := zlib.NewWriter(c_bss)
				zipper.Write(bss)
				zipper.Close()

				t.Logf("Zip Data size: %v", c_bss.Len())

				outs := new(bytes.Buffer)
				rdr, err := zlib.NewReader(c_bss)
				if err != nil {
					t.Fatalf("ERROR: %v", err)
				}
				io.Copy(outs, rdr)
				rdr.Close()
				t.Logf("Zip Decode Data size: %v", outs.Len())
				if err != nil {
					t.Fatalf("Error: %v", err)
				}

				n_iter, err := NewIter(stype, outs.Bytes())
				if err != nil {
					t.Fatalf("Error: %v", err)
				}
				idx = 0
				for n_iter.Next() {
					to, mi, mx, ls, su, ct := n_iter.Values()
					test_to := time.Unix(0, to)
					test_lt := time.Unix(0, times[idx])
					if ser.HighResolution() {
						So(test_lt.UnixNano(), ShouldEqual, test_to.UnixNano())
					} else {
						So(test_lt.Unix(), ShouldEqual, test_to.Unix())
					}
					So(stats[idx].Max, ShouldEqual, mx)
					So(stats[idx].Last, ShouldEqual, ls)
					So(stats[idx].Count, ShouldEqual, ct)
					So(stats[idx].Min, ShouldEqual, mi)
					So(stats[idx].Sum, ShouldEqual, su)
					idx++
				}
			})

			Convey("Series Type: "+stype+" - LZW/Iterator - "+nm, func() {
				// zip tests
				c_bss := new(bytes.Buffer)
				zipper := lzw.NewWriter(c_bss, lzw.MSB, 8)
				zipper.Write(bss)
				zipper.Close()

				t.Logf("LZW Data size: %v", c_bss.Len())

				outs := new(bytes.Buffer)
				rdr := lzw.NewReader(c_bss, lzw.MSB, 8)
				if err != nil {
					t.Fatalf("ERROR: %v", err)
				}
				io.Copy(outs, rdr)
				rdr.Close()
				t.Logf("LZW Decode Data size: %v", outs.Len())
				if err != nil {
					t.Fatalf("Error: %v", err)
				}

				n_iter, err := NewIter(stype, outs.Bytes())
				if err != nil {
					t.Fatalf("Error: %v", err)
				}
				idx = 0
				for n_iter.Next() {
					to, mi, mx, ls, su, ct := n_iter.Values()
					test_to := time.Unix(0, to)
					test_lt := time.Unix(0, times[idx])
					if ser.HighResolution() {
						So(test_lt.UnixNano(), ShouldEqual, test_to.UnixNano())
					} else {
						So(test_lt.Unix(), ShouldEqual, test_to.Unix())
					}
					So(stats[idx].Max, ShouldEqual, mx)
					So(stats[idx].Last, ShouldEqual, ls)
					So(stats[idx].Count, ShouldEqual, ct)
					So(stats[idx].Min, ShouldEqual, mi)
					So(stats[idx].Sum, ShouldEqual, su)
					idx++
				}
			})

		})
	}
}
