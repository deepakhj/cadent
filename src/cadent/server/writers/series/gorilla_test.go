package series

import (
	"bytes"
	"cadent/server/repr"
	"compress/flate"
	"fmt"
	"github.com/golang/snappy"
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"testing"
	"time"
)

func TestGorillaTimeSeries(t *testing.T) {

	Convey("GorillaSeries", t, func() {
		stat, n := dummyStat()


		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		var err error
		n_stats := 256
		times, stats, err := addStats(ser, stat, n_stats)

		So(err, ShouldEqual, nil)
		if ser.fullResolution {
			So(ser.StartTime(), ShouldEqual, n.UnixNano())
			So(ser.LastTime(), ShouldEqual, times[len(times) - 1])
		}else{
			tt := time.Unix(0, ser.StartTime())
			So(tt.Unix(), ShouldEqual, n.Unix())
			lt := time.Unix(0, ser.LastTime())
			test_lt := time.Unix(0, times[len(times) - 1])
			So(lt.Unix(), ShouldEqual, test_lt.Unix())
		}
		it, err := ser.Iter()
		So(err, ShouldEqual, nil)
		idx := 0
		for it.Next() {

			/*to, mi,_,_,_,_,_:= it.Values()
			t_second := time.Unix(0, times[idx]).Unix()
			t_o_second := time.Unix(0, to).Unix()
			t.Logf("%d: Data Comp: init: %v, diff: %v, init: %v, diff: %v", idx, t_second, t_o_second - t_second, stats[idx].Min, mi - float64(stats[idx].Min))
			idx++
			continue*/

			to, mi, mx, fi, ls, su, ct := it.Values()
			t_orig := times[idx]
			if !ser.fullResolution {
				t_orig = time.Unix(0, times[idx]).Unix()
				to = time.Unix(0, to).Unix()

			}
			So(t_orig, ShouldEqual, to)
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

		t.Logf("Error: %v", it.Error())
		t.Logf("Data Len: %v", ser.Len())
		So(idx, ShouldEqual, n_stats)

		// see how much a 8kb blob holds
		max_size := 8 * 1024
		ser_size := NewMultiGoriallaTimeSeries(n.UnixNano())
		on_tick := 0
		for true {
			on_tick++
			dd, _ := time.ParseDuration(fmt.Sprintf("%ds", on_tick))
			stat.Time = stat.Time.Add(dd)
			stat.Max = repr.JsonFloat64(rand.Float64())
			stat.Sum = repr.JsonFloat64(float64(stat.Sum) + float64(on_tick))
			times = append(times, stat.Time.UnixNano())
			stats = append(stats, stat.Copy())
			err = ser_size.AddStat(&stat)
			if err != nil {
				t.Logf("ERROR: %v", err)
			}
			if ser_size.Len() >= max_size {
				t.Logf("Max Stats in Buffer for %d bytes is %d", max_size, on_tick)
				break
			}
		}
	})

}

func BenchmarkGorillaSeriesPut(b *testing.B) {
	b.ResetTimer()
	stat, n := dummyStat()

	b.SetBytes(stat.ByteSize())
	for i := 0; i < b.N; i++ {
		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		ser.AddStat(&stat)
	}
}

func BenchmarkGorillaSeriesBaseCompress(b *testing.B) {
	b.ResetTimer()
	stat, n := dummyStat()


	n_stat := 1024
	for i := 0; i < b.N; i++ {
		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		addStats(ser, stat, n_stat)
		data, _ := ser.MarshalBinary()
		pre_l := len(data)
		b.Logf("Base Data Len: Pre: %v bytes per stat %v", pre_l, pre_l/n_stat)
	}
}

func BenchmarkGorillaSeriesRead(b *testing.B) {
	stat, n := dummyStat()

	n_stat := 2048
	ser := NewMultiGoriallaTimeSeries(n.UnixNano())
	addStats(ser, stat, n_stat)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.Logf("Bytes: %d", ser.Len())
		it, _ := ser.Iter()
		var j int
		for it.Next() {
			j++
		}
		b.Logf("Reads: %d", j)
		b.Logf("Error: %v", it.Error())
	}
}
func BenchmarkGorillaSeriesSnappyCompress(b *testing.B) {
	b.ResetTimer()
	stat, n := dummyStat()


	n_stat := 1024
	for i := 0; i < b.N; i++ {
		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		addStats(ser, stat, n_stat)
		data, _ := ser.MarshalBinary()
		pre_l := len(data)
		c_bss := snappy.Encode(nil, data)
		b.Logf("Data Len: Pre: %v bytes/Stat %v, Comp: %v bytes/stat %v", pre_l, len(c_bss)/n_stat, len(c_bss), len(c_bss)/n_stat)
	}
}

func BenchmarkGorillaSeriesFlateCompress(b *testing.B) {
	b.ResetTimer()
	stat, n := dummyStat()


	n_stat := 1024

	for i := 0; i < b.N; i++ {
		ser := NewMultiGoriallaTimeSeries(n.UnixNano())
		addStats(ser, stat, n_stat)
		bss, _ := ser.MarshalBinary()
		c_bss := new(bytes.Buffer)
		zipper, _ := flate.NewWriter(c_bss, flate.BestSpeed)
		pre_l := len(bss)
		zipper.Write(bss)
		zipper.Flush()
		b.Logf("Flate Comp Data Len: Pre: %v,  %v bytes per stat %v", pre_l, c_bss.Len(), c_bss.Len()/n_stat)
	}
}
