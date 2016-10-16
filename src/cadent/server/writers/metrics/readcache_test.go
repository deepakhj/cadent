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
		Name:  &repr.StatName{Key: "goo", Resolution: 2},
		Sum:   5,
		Min:   1,
		Max:   3,
		Last:  2,
		Count: 4,
	}

	r_list := []string{"moo", "goo", "loo", "hoo", "loo", "noo", "too", "elo", "houses", "trains", "acrs", "and", "other", "things"}
	Convey("ReadCacheItem", t, func() {

		m_bytes := 512
		c_item := NewReadCacheItem(m_bytes)

		t_start := time.Now()

		for i := 0; i < 1000; i++ {
			s := stat.Copy()
			s.Sum = rand.Float64()
			// a random time testing the sorts
			s.Time = t_start.UnixNano() + int64(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
			c_item.Add(s)
		}
		Convey("ReadCacheItems have proper time order", func() {
			So(c_item.StartTime.Before(c_item.EndTime), ShouldEqual, true)
		})
		t.Logf("CacheItems: Start: %s", c_item.StartTime)
		t.Logf("CacheItems: End: %s", c_item.EndTime)

		data, _, _ := c_item.Get(time.Time{}, time.Now().Add(time.Duration(time.Second*time.Duration(10000))))
		t.Logf("Cache Get: %v", data)

		Convey("ReadCacheItems Small cache", func() {
			small_cache := NewReadCacheItem(m_bytes)
			for i := 0; i < 5; i++ {
				s := stat.Copy()
				s.Sum = rand.Float64()
				// a random time testing the sorts
				s.Time = t_start.UnixNano() + int64(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
				small_cache.Add(s)
			}
			So(len(small_cache.GetAll()), ShouldEqual, 5)
			data, _, _ := small_cache.Get(time.Time{}, time.Now().Add(time.Duration(time.Second*time.Duration(10000))))
			So(len(data), ShouldEqual, 5)

		})
	})

	Convey("ReadCache", t, func() {

		m_bytes := 512
		mx_bytes := 5 * m_bytes
		max_back := time.Second * 20
		c_item := NewReadCache(mx_bytes, m_bytes, max_back)

		t_start := time.Now()

		for i := 0; i < 1000; i++ {
			s := stat.Copy()
			m_idx := i % len(r_list)
			r_prefix := r_list[m_idx]
			s.Name.Key = s.Name.Key + "." + r_prefix
			s.Sum = rand.Float64()
			// a random time testing the sorts
			s.Time = t_start.UnixNano() + int64(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
			c_item.ActivateMetric(s.Name.Key, nil)
			c_item.Put(s.Name.Key, s)
		}
		//t.Logf("ReadCache: Size: %d, Keys: %d", c_item.Size(), c_item.NumKeys())
		//t.Logf("ReadCache: %v", c_item.lru.Items())
		Convey("ReadCache should have some space left", func() {
			So(c_item.Size(), ShouldNotEqual, 0)
		})

		Convey("ReadCache Get should return", func() {
			t_end := t_start.Add(time.Second * 2000)
			items := c_item.lru.Items()
			for _, i := range items {
				gots := c_item.GetAll(i.Key)
				t_gots, _, _ := c_item.Get(i.Key, t_start, t_end)
				So(len(gots), ShouldEqual, len(t_gots))

			}
		})
	})

	Convey("ReadCache Singleton", t, func() {

		m_bytes := 512
		mx_bytes := 5 * m_bytes
		max_back := time.Second * 20
		// activate the singleton
		gots := InitReadCache(mx_bytes, m_bytes, max_back)
		So(gots, ShouldNotEqual, nil)
		t_start := time.Now()

		var t_series RawRenderItem
		raw_nm := "moo.goo.org"
		t_series.Metric = raw_nm

		for i := 0; i < 1000; i++ {
			s := stat.Copy()
			m_idx := i % len(r_list)
			r_prefix := r_list[m_idx]
			s.Name.Key = s.Name.Key + "." + r_prefix
			s.Sum = rand.Float64()
			// a random time testing the sorts
			s.Time = t_start.UnixNano() + int64(time.Duration(time.Second * time.Duration(rand.Int63n(1000))))
			GetReadCache().ActivateMetric(s.Name.Key, nil)
			GetReadCache().Put(s.Name.Key, s)

			s2 := s.Copy()
			s.Name.Key = raw_nm
			t_series.Data = append(t_series.Data, &RawDataPoint{Sum: float64(s2.Sum), Time: uint32(s2.Time)})
		}
		t.Logf("ReadCache Singleton: Size: %d, Keys: %d Capacity: %d", GetReadCache().Size(), GetReadCache().NumKeys(), GetReadCache().lru.GetCapacity())
		//t.Logf("ReadCache Singleton: %v", GetReadCache().lru.Items())
		Convey("ReadCache Singleton should have some space", func() {
			So(GetReadCache().Size(), ShouldNotEqual, 0)
		})

		Convey("ReadCache Singleton Get should return", func() {
			t_end := t_start.Add(time.Second * 2000)
			items := GetReadCache().lru.Items()
			for _, i := range items {
				gots := GetReadCache().GetAll(i.Key)
				t_gots, _, _ := GetReadCache().Get(i.Key, t_start, t_end)
				So(len(gots), ShouldEqual, len(t_gots))

			}
		})
		Convey("ReadCache Singleton Activate From rendered", func() {

			GetReadCache().ActivateMetricFromRenderData(&t_series)
			dd := GetReadCache().GetAll(raw_nm)
			t.Logf("Orig Data: %d .. Cached Data: %d", len(t_series.Data), len(dd))
			o_data := t_series.Data[len(t_series.Data)-len(dd):]
			for idx, stat := range dd {
				So(o_data[idx].Time, ShouldEqual, stat.Time)
				So(o_data[idx].Sum, ShouldEqual, float64(stat.Sum)/float64(stat.Count))
				//t.Logf("Orig Data: %v:%v", o_data[idx].Time, o_data[idx].Sum)
				//t.Logf("Cache Data: %v:%v", stat.Time.Unix(), float64(stat.Sum)/float64(stat.Count))

			}
		})
	})
}
