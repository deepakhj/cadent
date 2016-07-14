package metrics

import (
	"cadent/server/repr"
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"testing"
	"time"
)

func TestWriterReadCache(t *testing.T) {

	stat := repr.StatRepr{
		Key:        "goo",
		StatKey:    "goo",
		Sum:        5,
		Mean:       10,
		Min:        1,
		Max:        3,
		Last:       2,
		Count:      4,
		Resolution: 2,
	}

	r_list := []string{"moo", "goo", "loo", "hoo", "loo", "noo", "too", "elo", "houses", "trains", "acrs", "and", "other", "things"}
	Convey("ReadCacheItem", t, func() {

		m_bytes := 512
		c_item := NewReadCacheItem(m_bytes)

		t_start := time.Now()

		for i := 0; i < 1000; i++ {
			s := stat.Copy()
			s.Mean = repr.JsonFloat64(rand.Float64())
			// a random time testing the sorts
			s.Time = t_start.Add(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
			c_item.Add(s)
		}
		Convey("ReadCacheItems have proper time order", func() {
			So(c_item.StartTime.Before(c_item.EndTime), ShouldEqual, true)
		})
		t.Logf("CacheItems: Start: %s", c_item.StartTime)
		t.Logf("CacheItems: End: %s", c_item.EndTime)

		data := c_item.Get(time.Time{}, time.Now().Add(time.Duration(time.Second*time.Duration(10000))))
		t.Logf("Cache Get: %v", data)

		Convey("ReadCacheItems Small cache", func() {
			small_cache := NewReadCacheItem(m_bytes)
			for i := 0; i < 5; i++ {
				s := stat.Copy()
				s.Mean = repr.JsonFloat64(rand.Float64())
				// a random time testing the sorts
				s.Time = t_start.Add(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
				small_cache.Add(s)
			}
			So(len(small_cache.GetAll()), ShouldEqual, 5)
			data := small_cache.Get(time.Time{}, time.Now().Add(time.Duration(time.Second*time.Duration(10000))))
			So(len(data), ShouldEqual, 5)

		})
	})

	Convey("ReadCache", t, func() {

		m_bytes := 512
		m_keys := 5
		max_back := time.Second * 20
		c_item := NewReadCache(m_keys, m_bytes, max_back)

		t_start := time.Now()

		for i := 0; i < 1000; i++ {
			s := stat.Copy()
			m_idx := i % len(r_list)
			r_prefix := r_list[m_idx]
			s.Key = s.Key + "." + r_prefix
			s.Mean = repr.JsonFloat64(rand.Float64())
			// a random time testing the sorts
			s.Time = t_start.Add(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
			c_item.ActivateMetric(s.Key, nil)
			c_item.Put(s.Key, s)
		}
		t.Logf("ReadCache: Size: %d, Keys: %d", c_item.Size(), c_item.NumKeys())
		t.Logf("ReadCache: %v", c_item.lru.Items())
		Convey("ReadCache should have some space left", func() {
			So(c_item.Size(), ShouldNotEqual, 0)
		})

		Convey("ReadCache Get should return", func() {
			t_end := t_start.Add(time.Second * 2000)
			items := c_item.lru.Items()
			for _, i := range items {
				gots := c_item.GetAll(i.Key)
				t_gots := c_item.Get(i.Key, t_start, t_end)
				So(len(gots), ShouldEqual, len(t_gots))

			}
		})
	})

	Convey("ReadCache Singleton", t, func() {

		m_bytes := 512
		m_keys := 5
		max_back := time.Second * 20
		// activate the singleton
		gots := InitReadCache(m_keys, m_bytes, max_back)
		So(gots, ShouldNotEqual, nil)
		t_start := time.Now()

		for i := 0; i < 1000; i++ {
			s := stat.Copy()
			m_idx := i % len(r_list)
			r_prefix := r_list[m_idx]
			s.Key = s.Key + "." + r_prefix
			s.Mean = repr.JsonFloat64(rand.Float64())
			// a random time testing the sorts
			s.Time = t_start.Add(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
			GetReadCache().ActivateMetric(s.Key, nil)
			GetReadCache().Put(s.Key, s)
		}
		t.Logf("ReadCache Singleton: Size: %d, Keys: %d Capacity: %d", GetReadCache().Size(), GetReadCache().NumKeys(), GetReadCache().lru.GetCapacity())
		t.Logf("ReadCache Singleton: %v", GetReadCache().lru.Items())
		Convey("ReadCache Singleton should space consumed", func() {
			So(GetReadCache().Size(), ShouldNotEqual, 0)
		})

		Convey("ReadCache Singleton Get should return", func() {
			t_end := t_start.Add(time.Second * 2000)
			items := GetReadCache().lru.Items()
			for _, i := range items {
				gots := GetReadCache().GetAll(i.Key)
				t_gots := GetReadCache().Get(i.Key, t_start, t_end)
				So(len(gots), ShouldEqual, len(t_gots))

			}
		})
	})
}
