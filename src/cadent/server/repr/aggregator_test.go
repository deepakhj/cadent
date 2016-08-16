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

package repr

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestStatReprAggregator(t *testing.T) {
	// Only pass t into top-level Convey calls
	t_time := time.Now()
	ss := StatRepr{
		Name:  StatName{Key: "moo", Resolution: 1},
		Sum:   5,
		Min:   1,
		Max:   3,
		Count: 4,
		Time:  t_time,
	}
	ssM := StatRepr{
		Name: StatName{Key: "moo", Resolution: 1},

		Sum:   5,
		Min:   0,
		Max:   8,
		Count: 4,
		Time:  t_time,
	}
	ss2 := StatRepr{
		Name: StatName{Key: "goo", Resolution: 2},

		Sum:   5,
		Min:   1,
		Max:   3,
		Count: 4,
		Time:  t_time,
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
		m_key := sc.MapKey(ss.Name.UniqueIdString(), t_time)
		Convey("Should have been aggrigated", func() {
			gots := sc.Items[m_key]
			So(gots.Sum, ShouldEqual, 10)
			So(gots.Min, ShouldEqual, 1)
			So(gots.Count, ShouldEqual, 8)
		})
		sc.Add(ssM)
		Convey("Should have been aggrigated Again", func() {
			gots := sc.Items[m_key]
			So(gots.Sum, ShouldEqual, 15)
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
