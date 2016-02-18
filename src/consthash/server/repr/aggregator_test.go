package repr

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestStatReprAggregator(t *testing.T) {
	// Only pass t into top-level Convey calls

	ss := StatRepr{
		StatKey:    "moo",
		Key:        "moo",
		Sum:        5,
		Min:        1,
		Max:        3,
		Count:      4,
		Time:       time.Now(),
		Resolution: 1,
	}
	ssM := StatRepr{
		StatKey:    "moo",
		Key:        "moo",
		Sum:        5,
		Min:        0,
		Max:        8,
		Count:      4,
		Time:       time.Now(),
		Resolution: 1,
	}
	ss2 := StatRepr{
		Key:        "goo",
		StatKey:    "goo",
		Sum:        5,
		Min:        1,
		Max:        3,
		Count:      4,
		Time:       time.Now(),
		Resolution: 2,
	}

	Convey("Aggregator", t, func() {
		sc := NewAggregator(time.Duration(10 * time.Second))

		sc.Add(ss)
		Convey("Should have one elment", func() {
			So(sc.Len(), ShouldEqual, 1)
		})
		sc.Add(ss2)
		Convey("Should have two", func() {
			So(sc.Len(), ShouldEqual, 2)
		})

		sc.Add(ss)
		Convey("Should have been aggrigated", func() {
			gots := sc.Items[ss.Key]
			So(gots.Sum, ShouldEqual, 10)
			So(gots.Min, ShouldEqual, 1)
			So(gots.Count, ShouldEqual, 8)
		})
		sc.Add(ssM)
		Convey("Should have been aggrigated Again", func() {
			gots := sc.Items[ss.Key]
			So(gots.Min, ShouldEqual, 0)
			So(gots.Max, ShouldEqual, 8)
		})
	})

	Convey("MultiAggregator", t, func() {
		durs := []time.Duration{
			time.Duration(10 * time.Second),
			time.Duration(60 * time.Second),
			time.Duration(10 * 60 * time.Second),
		}
		sc := NewMulti(durs)
		Convey("Should have 3 elment", func() {
			So(sc.Len(), ShouldEqual, 3)
		})

		sc.Add(ss)
		Convey("Each Agg Should have one elment", func() {
			for _, agg := range sc.Aggs {
				So(agg.Len(), ShouldEqual, 1)
			}
		})
		sc.Add(ss2)
		Convey("Should have two", func() {
			for _, agg := range sc.Aggs {
				So(agg.Len(), ShouldEqual, 2)
			}
		})
	})

}