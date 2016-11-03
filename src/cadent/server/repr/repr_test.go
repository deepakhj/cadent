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
	"encoding/json"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"hash/fnv"
	"strconv"
	"testing"
	"time"
)

func TestStatAccumulatorRepr(t *testing.T) {
	// Only pass t into top-level Convey calls
	moo_nm := StatName{Key: "moo", Resolution: 1}
	ss := StatRepr{
		Name:  &moo_nm,
		Sum:   5,
		Min:   1,
		Max:   88,
		Count: 4,
		Time:  time.Now().UnixNano(),
	}

	tgs := []*Tag{{Name: "nameZ", Value: "value1"}, {Name: "nameA", Value: "value2"}}

	goo_nm := StatName{Key: "goo", Resolution: 2, Tags: SortingTags(tgs)}
	ss2 := StatRepr{
		Name:  &goo_nm,
		Sum:   5,
		Min:   1,
		Max:   3,
		Count: 4,
		Time:  time.Now().UnixNano(),
	}

	Convey("Stat Names Copy", t, func() {
		cp := moo_nm.Copy()
		cp.SetKey("monkey")

		So(cp.Key, ShouldNotEqual, moo_nm.Key)

	})

	Convey("Stat Copy", t, func() {
		cp := ss.Copy()

		So(cp.Name.Key, ShouldEqual, ss.Name.Key)
		So(cp.Count, ShouldEqual, ss.Count)
		So(cp.Time, ShouldEqual, ss.Time)
		So(cp.Max, ShouldEqual, ss.Max)
		So(cp.Min, ShouldEqual, ss.Min)
		So(cp.Last, ShouldEqual, ss.Last)

		So(&cp, ShouldNotEqual, &ss)

	})

	Convey("Stat Names", t, func() {
		Convey("tags should sort", func() {
			org := goo_nm.Tags.String()
			So(org, ShouldEqual, "nameZ=value1 nameA=value2")
			tags_str := goo_nm.SortedTags().String()
			So(tags_str, ShouldEqual, "nameA=value2 nameZ=value1")
		})
		Convey("unique IDs should be correct", func() {
			buf := fnv.New64a()
			fmt.Fprintf(buf, "%s:%s", goo_nm.Key, goo_nm.SortedTags())
			u_1 := buf.Sum64()

			So(goo_nm.UniqueId(), ShouldEqual, u_1) // fnv64a(key+:+sortedNames(tags))
			So(goo_nm.UniqueIdString(), ShouldEqual, strconv.FormatUint(uint64(u_1), 36))

			buf = fnv.New64a()
			fmt.Fprintf(buf, "%s:%s", moo_nm.Key, moo_nm.SortedTags())
			u_2 := buf.Sum64()

			So(moo_nm.UniqueId(), ShouldEqual, u_2)
			So(moo_nm.UniqueIdString(), ShouldEqual, strconv.FormatUint(uint64(u_2), 36))
		})
		Convey("Graphite names should be correct", func() {
			So(goo_nm.Name(), ShouldEqual, "goo.nameA=value2.nameZ=value1")
			So(moo_nm.Name(), ShouldEqual, "moo")
		})
	})

	STAT_REPR_CACHE.Add(ss)
	Convey("Global Cacher", t, func() {

		_, err := json.Marshal(&ss)
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
