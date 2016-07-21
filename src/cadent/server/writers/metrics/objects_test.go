package metrics

import (
	. "github.com/smartystreets/goconvey/convey"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestWriterObjects(t *testing.T) {

	n_sts1 := 20
	n_sts2 := 8
	t_list1 := make([]RawDataPoint, n_sts1)
	t_list2 := make([]RawDataPoint, n_sts2)
	t_list3 := make([]RawDataPoint, n_sts2)
	t_list4 := make([]RawDataPoint, n_sts2)

	step_1 := uint32(10)
	t_start := uint32(time.Now().Unix())

	for idx := range t_list1 {
		t_list1[idx] = RawDataPoint{
			Time: t_start + uint32(idx)*step_1,
			Mean: rand.Float64(),
		}
	}

	t_end1 := t_list1[len(t_list1)-1].Time + step_1

	for idx := range t_list2 {
		t_list2[idx] = RawDataPoint{
			Time: t_start + uint32(idx)*step_1,
			Mean: rand.Float64(),
		}
	}
	t_end2 := t_list2[len(t_list2)-1].Time + step_1

	// random steps
	for idx := range t_list3 {
		t_list3[idx] = RawDataPoint{
			Time: t_start + uint32(idx),
			Mean: float64(rand.Int63n(100)),
			Min:  float64(idx),
			Max:  float64(idx + 100),
		}
	}
	t_end3 := t_list3[len(t_list3)-1].Time + 1

	// random steps
	for idx := range t_list4 {
		t_list4[idx] = RawDataPoint{
			Time: t_start + 10*uint32(idx) + uint32(rand.Int63n(2)+1.0),
			Mean: float64(rand.Int63n(100)),
			Min:  float64(idx),
			Max:  float64(idx + 100),
		}
	}
	t_end4 := t_list4[len(t_list4)-1].Time + 10

	Convey("Raw Data Item Quantize", t, func() {

		// this should "expand" in size to fill in the end points
		rl := &RawRenderItem{
			Data:  t_list2,
			Start: t_start,
			Step:  step_1,
			End:   t_end1,
		}

		rl.Quantize()

		So(rl.Len(), ShouldBeGreaterThan, len(t_list2))
		So(math.IsNaN(rl.Data[rl.Len()-1].Mean), ShouldEqual, true)

		// truncate
		rl.TrunctateTo(t_start, t_end2)
		So(rl.Len(), ShouldEqual, len(t_list2))
	})

	Convey("Raw Data Item Resample", t, func() {

		// this should "expand" in size to fill in the end points
		rl := &RawRenderItem{
			Data:  t_list3,
			Start: t_start,
			End:   t_end3,
		}

		for _, d := range rl.Data {
			t.Logf("Pre Data: %v", d)
		}
		rl.Resample(2)
		for _, d := range rl.Data {
			t.Logf("Data: %v", d)
		}
		So(rl.Len(), ShouldEqual, len(t_list3)/2)

		rl = &RawRenderItem{
			Data:  t_list4,
			Start: t_start,
			End:   t_end4,
		}

		for _, d := range rl.Data {
			t.Logf("Pre Data: %v", d)
		}
		rl.Resample(20)
		for _, d := range rl.Data {
			t.Logf("Data: %v", d)
		}
		// can be the same or +/-1 depending on the start time and time divisor
		t_l := len(t_list4)
		So(rl.Len(), ShouldBeIn, []int{t_l/2 - 1, t_l / 2, t_l/2 + 1})

		rl.Resample(5)
		for _, d := range rl.Data {
			t.Logf("RE2 Data: %v", d)
		}
	})

	Convey("Raw Data item tests", t, func() {

		r_list1 := &RawRenderItem{
			Data:  t_list1,
			Start: t_start,
			Step:  step_1,
			End:   t_end1,
		}

		r_list2 := &RawRenderItem{
			Data:  t_list2,
			Start: t_start,
			Step:  step_1,
			End:   t_end2,
		}

		r_list1.Merge(r_list2)
		So(r_list1.Len(), ShouldEqual, r_list2.Len())

		for idx, data := range r_list1.Data {
			So(data.Mean, ShouldEqual, t_list1[idx].Mean)
		}

		// reset
		r_list1 = &RawRenderItem{
			Data:  t_list1,
			Start: t_start,
			Step:  step_1,
			End:   t_end1,
		}

		r_list2.Merge(r_list1)
		So(r_list2.Len(), ShouldEqual, r_list1.Len())

		for idx, data := range r_list2.Data {
			if idx < n_sts2 {
				So(data.Mean, ShouldEqual, t_list2[idx].Mean)
			} else {
				So(data.Mean, ShouldEqual, t_list1[idx].Mean)
			}
		}

	})

}
