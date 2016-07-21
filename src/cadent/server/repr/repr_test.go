package repr

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"math"
	"testing"
	"time"
)

func TestStatAccumulatorRepr(t *testing.T) {
	// Only pass t into top-level Convey calls
	moo_nm := StatName{Key: "moo", Resolution: 1}
	ss := StatRepr{
		Name:       moo_nm,
		Sum:        5,
		Min:        1,
		Max:        JsonFloat64(math.Inf(1)),
		Count:      4,
		Time:       time.Now(),
	}
	goo_nm := StatName{Key: "goo", Resolution: 2}
	ss2 := StatRepr{
		Name:       goo_nm,
		Sum:        5,
		Min:        1,
		Max:        3,
		Count:      4,
		Time:       time.Now(),
	}

	STAT_REPR_CACHE.Add(ss)
	Convey("Global Cacher", t, func() {

		_, err := json.Marshal(ss)
		ss.Max = 3
		Convey("Should convert to Json", func() {
			So(err, ShouldEqual, nil)
		})

		Convey("Should have one elment", func() {
			So(STAT_REPR_CACHE.Len(), ShouldEqual, 1)
		})
		STAT_REPR_CACHE.Add(ss)
		Convey("Should have one elment (again)", func() {
			So(STAT_REPR_CACHE.Len(), ShouldEqual, 1)
		})
		gg := STAT_REPR_CACHE.Get(moo_nm.UniqueId())
		Convey("gotten element should exist", func() {
			So(gg, ShouldNotEqual, nil)
		})

	})

	Convey("Local 1  Cacher", t, func() {
		sc := NewReprCache(1)
		sc.Add(ss)
		Convey("Should have one elment", func() {
			So(sc.Len(), ShouldEqual, 1)
		})
		sc.Add(ss2)
		Convey("Should have one elment (again)", func() {
			So(sc.Len(), ShouldEqual, 1)
		})
	})

	Convey("Local 2 Cacher", t, func() {
		sc := NewReprCache(2)
		sc.Add(ss)
		Convey("Should have one elment", func() {
			So(sc.Len(), ShouldEqual, 1)
		})
		sc.Add(ss)
		Convey("Should have one elment again", func() {
			So(sc.Len(), ShouldEqual, 1)
		})
		sc.Add(ss2)
		Convey("Should have two elment", func() {
			So(sc.Len(), ShouldEqual, 2)
		})
		sc.Add(ss2)
		Convey("Should have two elment (again)", func() {
			So(sc.Len(), ShouldEqual, 2)
		})

		g_st := sc.Get(goo_nm.UniqueId())
		Convey("Gotten element should have 2 items", func() {
			So(g_st.Len(), ShouldEqual, 2)
			So(g_st.Reprs[0].Name.Key, ShouldEqual, "goo")
		})

		g_st = sc.Pop()
		Convey("Pop Gotten element should have 2 items", func() {
			So(sc.Len(), ShouldEqual, 1)
			So(g_st.Len(), ShouldEqual, 2)
			So(g_st.Reprs[0].Name.Key, ShouldEqual, "moo")
		})
		g_st = sc.Delete(moo_nm.UniqueId())
		Convey("Delete missing element", func() {
			So(g_st, ShouldEqual, nil)
		})
		g_st = sc.Delete(goo_nm.UniqueId())
		Convey("Delete Gotten element should have 2 items", func() {
			So(sc.Len(), ShouldEqual, 0)
			So(g_st.Len(), ShouldEqual, 2)
			So(g_st.Reprs[0].Name.Key, ShouldEqual, "goo")
		})
	})
}
