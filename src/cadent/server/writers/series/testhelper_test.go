/*
	some helper generics for the testing ond benchmarking of the verios timeseries bits
*/

package series

import (
	"bytes"
	"cadent/server/repr"
	"compress/flate"
	"compress/zlib"
	"fmt"
	"github.com/golang/snappy"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func dummyStat() (*repr.StatRepr, time.Time) {
	stat := &repr.StatRepr{
		Sum:   repr.JsonFloat64(rand.Float64()),
		Min:   repr.JsonFloat64(rand.Float64()),
		Max:   repr.JsonFloat64(rand.Float64()),
		Last:  repr.JsonFloat64(rand.Float64()),
		First: repr.JsonFloat64(rand.Float64()),
		Count: 123123123,
	}
	n := time.Now()
	stat.Time = n
	return stat, n
}

func dummyStatInt() (*repr.StatRepr, time.Time) {
	stat := &repr.StatRepr{
		Sum:   repr.JsonFloat64(rand.Int63n(100)),
		Min:   repr.JsonFloat64(rand.Int63n(100)),
		Max:   repr.JsonFloat64(rand.Int63n(100)),
		Last:  repr.JsonFloat64(rand.Int63n(100)),
		First: repr.JsonFloat64(rand.Int63n(100)),
		Count: rand.Int63n(100),
	}
	n := time.Now()
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
			stat.Time = stat.Time.Add(dd)
			stat.Max = repr.JsonFloat64(rand.Float64() * 20000.0)
			stat.Min = repr.JsonFloat64(rand.Float64() * 20000.0)
			stat.First = repr.JsonFloat64(rand.Float64() * 20000.0)
			stat.Last = repr.JsonFloat64(rand.Float64() * 20000.0)
			stat.Sum = repr.JsonFloat64(float64(stat.Sum) + float64(i))
			stat.Count = rand.Int63n(1000)
		} else {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", 1+i))
			stat.Time = stat.Time.Add(dd)
			stat.Max += repr.JsonFloat64(1 + i)
			stat.Min += repr.JsonFloat64(1 + i)
			stat.First += repr.JsonFloat64(1 + i)
			stat.Last += repr.JsonFloat64(1 + i)
			stat.Sum += repr.JsonFloat64(1 + i)
			stat.Count += int64(1 + i)
		}
		times = append(times, stat.Time.UnixNano())
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
			stat.Time = stat.Time.Add(dd)
			stat.Max = repr.JsonFloat64(rand.Float64() * 20000.0)
			stat.Min = repr.JsonFloat64(rand.Float64() * 20000.0)
			stat.First = repr.JsonFloat64(rand.Float64() * 20000.0)
			stat.Last = repr.JsonFloat64(rand.Float64() * 20000.0)
			stat.Sum = repr.JsonFloat64(float64(stat.Sum) + float64(i))
			stat.Count = rand.Int63n(1000)
		} else {
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", 1+i))
			stat.Time = stat.Time.Add(dd)
			stat.Max += repr.JsonFloat64(1 + i)
			stat.Min += repr.JsonFloat64(1 + i)
			stat.First += repr.JsonFloat64(1 + i)
			stat.Last += repr.JsonFloat64(1 + i)
			stat.Sum += repr.JsonFloat64(1 + i)
			stat.Count += int64(1 + i)
		}
		times = append(times, stat.Time.UnixNano())
		stats = append(stats, stat.Copy())
		err = ser.AddStat(stat)
		if err != nil {
			return times, stats, err
		}
	}
	return times, stats, nil
}

func benchmarkRawSize(b *testing.B, stype string, n_stat int) {
	b.ResetTimer()
	stat, n := dummyStat()
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64bit numbers
	runs := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)

	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
		addStats(ser, stat, n_stat, true)
		bss, _ := ser.MarshalBinary()
		pre_len += int64(len(bss))
		b_per_stat += int64(int64(len(bss)) / int64(n_stat))
		runs++
	}
	b.Logf("Raw Size for %v stats: %v Bytes per stat: %v", n_stat, pre_len/runs, b_per_stat/runs)

}

func benchmarkFlateCompress(b *testing.B, stype string, n_stat int) {
	stat, n := dummyStat()
	runs := int64(0)
	comp_len := int64(0)
	b_per_stat := int64(0)
	pre_len := int64(0)
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64byte numbers
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
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

func benchmarkSeriesReading(b *testing.B, stype string, n_stat int) {

	stat, n := dummyStat()

	runs := int64(0)
	reads := int64(0)
	b.ResetTimer()
	b.SetBytes(int64(8 * 8 * n_stat)) //8 64bit numbers

	for i := 0; i < b.N; i++ {

		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
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
	b.ResetTimer()
	stat, n := dummyStat()
	b.SetBytes(int64(8 * 8)) //8 64bit numbers
	for i := 0; i < b.N; i++ {
		ser, err := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
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

	for i := 0; i < b.N; i++ {

		stat, n := dummyStat()
		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
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
	b.Logf("Random Max Stats in Buffer for %d bytes is %d", max_size, stat_ct/runs)

}

func benchmarkSeriesPut8kNonRandom(b *testing.B, stype string) {

	// see how much a 8kb blob holds for "not-so-random" but incremental stats
	runs := int64(0)
	stat_ct := int64(0)
	max_size := 8 * 1024
	b.SetBytes(int64(max_size))

	for i := 0; i < b.N; i++ {

		stat, n := dummyStat()
		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
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
	b.Logf("Incremental Max Stats in Buffer for %d bytes is %d", max_size, stat_ct/runs)

}

func benchmarkSeriesPut8kRandomInt(b *testing.B, stype string) {

	// see how much a 8kb blob holds for "not-so-random" but incremental stats
	runs := int64(0)
	stat_ct := int64(0)
	max_size := 8 * 1024
	b.SetBytes(int64(max_size))

	for i := 0; i < b.N; i++ {

		stat, n := dummyStatInt()
		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
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
	b.Logf("Incremental Max Stats in Buffer for %d bytes is %d", max_size, stat_ct/runs)

}

func genericTestSeries(t *testing.T, stype string) {

	Convey("Series Type: "+stype, t, func() {
		stat, n := dummyStat()

		ser, _ := NewTimeSeries(stype, n.UnixNano(), NewDefaultOptions())
		n_stats := 10
		times, stats, err := addStats(ser, stat, n_stats, true)

		So(err, ShouldEqual, nil)
		So(ser.StartTime(), ShouldEqual, n.UnixNano())
		So(ser.LastTime(), ShouldEqual, times[len(times)-1])

		it, err := ser.IterClone()
		idx := 0
		for it.Next() {
			to, mi, mx, fi, ls, su, ct := it.Values()
			So(times[idx], ShouldEqual, to)
			So(stats[idx].Max, ShouldEqual, mx)
			So(stats[idx].Last, ShouldEqual, ls)
			So(stats[idx].Count, ShouldEqual, ct)
			So(stats[idx].First, ShouldEqual, fi)
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
		So(idx, ShouldEqual, n_stats)
		if it.Error() != nil {
			t.Fatalf("Iter Error: %v", it.Error())
		}

		// compression, iterator R/W test
		bss, _ := ser.ByteClone()
		if err != nil {
			t.Fatalf("ERROR: %v", err)
		}
		t.Logf("Data size: %v", len(bss))
		Convey("Series Type: "+stype+" - Snappy/Iterator", func() {
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
				to, mi, mx, fi, ls, su, ct := n_iter.Values()
				So(times[idx], ShouldEqual, to)
				So(stats[idx].Max, ShouldEqual, mx)
				So(stats[idx].Last, ShouldEqual, ls)
				So(stats[idx].Count, ShouldEqual, ct)
				So(stats[idx].First, ShouldEqual, fi)
				So(stats[idx].Min, ShouldEqual, mi)
				So(stats[idx].Sum, ShouldEqual, su)
				idx++
			}
		})

		Convey("Series Type: "+stype+" - Zip/Iterator", func() {
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
				to, mi, mx, fi, ls, su, ct := n_iter.Values()
				So(times[idx], ShouldEqual, to)
				So(stats[idx].Max, ShouldEqual, mx)
				So(stats[idx].Last, ShouldEqual, ls)
				So(stats[idx].Count, ShouldEqual, ct)
				So(stats[idx].First, ShouldEqual, fi)
				So(stats[idx].Min, ShouldEqual, mi)
				So(stats[idx].Sum, ShouldEqual, su)
				idx++
			}
		})

	})
}
