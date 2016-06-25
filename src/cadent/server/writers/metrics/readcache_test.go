package metrics

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
	"cadent/server/repr"
	"math/rand"
	"sort"
)

func TestWriterReadCache(t *testing.T) {

	stat := repr.StatRepr{
		Key:        "goo",
		StatKey:    "goo",
		Sum:        5,
		Mean: 10,
		Min:        1,
		Max:        3,
		Last: 2,
		Count:      4,
		Resolution: 2,
	}

	Convey("ReadCacheItem", t, func() {

		m_items := 10
		c_item := NewReadCacheItem(m_items)

		t_start := time.Now()

		for i := 0; i<m_items + 10;i++{
			s := stat.Copy()
			s.Mean = repr.JsonFloat64(rand.Float64())
			// a random time testing the sorts
			s.Time = t_start.Add(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
			c_item.Add(s)
		}
		Convey("ReadCacheItems should be 10 long", func() {
			So(c_item.Data.Len(), ShouldEqual, 10)
			So(c_item.StartTime.Before(c_item.EndTime), ShouldEqual, true)
		})
		t.Logf("CacheItems: Start: %s", c_item.StartTime)
		t.Logf("CacheItems: End: %s", c_item.EndTime)

		data := c_item.Get(time.Time{}, time.Now().Add(time.Duration(time.Second * time.Duration(10000))))
		Convey("ReadCacheItems Get should be in order", func() {
			So(sort.IsSorted(SortedStats(data)), ShouldEqual, true)
		})
		t.Logf("Cache Get: %v", data)




		Convey("ReadCacheItems Small cache", func() {
			small_cache := NewReadCacheItem(m_items)
			for i := 0; i<5;i++{
				s := stat.Copy()
				s.Mean = repr.JsonFloat64(rand.Float64())
				// a random time testing the sorts
				s.Time = t_start.Add(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
				small_cache.Add(s)
			}
			So(small_cache.count, ShouldEqual, 5)
			data := small_cache.Get(time.Time{}, time.Now().Add(time.Duration(time.Second * time.Duration(10000))))
			So(sort.IsSorted(SortedStats(data)), ShouldEqual, true)
			So(len(data), ShouldEqual, 5)

		})
	})

}
