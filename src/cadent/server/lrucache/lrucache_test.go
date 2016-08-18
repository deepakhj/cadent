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

package lrucache

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

type TValue string

func (v TValue) Size() int {
	return len(v)
}
func (v TValue) ToString() string {
	return string(v)
}

func TestLRUCache(t *testing.T) {

	var strs = []string{
		"moooooooooooo",
		"poooooooooooo",
		"goooooooooooo",
		"toooooooooooo",
		"yoooooooooooo",
		"uoooooooooooo",
		"ioooooooooooo",
	}
	base_s := uint64(len(strs[0]))
	size := uint64(base_s * 4)
	lru := NewLRUCache(size)

	insize := uint64(0)
	for _, st := range strs {
		insize += uint64(len(st))
	}

	Convey("LRUcache should", t, func() {
		Convey("have a capacity", func() {
			So(lru.GetCapacity(), ShouldEqual, size)
		})

		Convey("accept some Keys", func() {
			for _, st := range strs {
				tv := TValue(st)
				lru.Set(st, tv)
			}
			// "updateinplace"
			lru.Set(strs[6], TValue(strs[6]))
			So(lru.size, ShouldEqual, size)
			So(lru.size < insize, ShouldBeTrue)
		})

		Convey("get some Keys", func() {
			gotct := 0
			for _, st := range strs {
				got, have := lru.Get(st)
				if have {
					So(got.ToString(), ShouldEqual, st)
					gotct += 1
				}
			}
			So(gotct, ShouldEqual, 4)
		})

		Convey("delete some Keys", func() {
			lru.Delete(strs[3])
			So(len(lru.Keys()), ShouldEqual, 3)
			_, have := lru.Get(strs[3])
			So(have, ShouldNotBeNil)
		})

		Convey("expand capacity", func() {
			lru.SetCapacity(insize)
			So(lru.GetCapacity(), ShouldEqual, insize)
			for _, st := range strs {
				tv := TValue(st)
				lru.SetIfAbsent(st, tv)
			}
			So(len(lru.Items()), ShouldEqual, len(strs))
		})

		Convey("Stats for coverage", func() {
			lru.StatsJSON()
		})

		Convey("be cleared", func() {
			lru.Clear()
			So(lru.GetCapacity(), ShouldEqual, insize)
			So(len(lru.Items()), ShouldEqual, 0)
		})
	})

}
