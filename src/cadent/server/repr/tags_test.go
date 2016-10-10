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

	"sort"
	"testing"
)

func TestTagsRepr(t *testing.T) {

	Convey("Tags Parsing", t, func() {
		t_str := "moo=goo loo=baz"
		t_str_1 := "moo=goo.loo=baz"
		t_str_2 := "moo=goo,loo=baz"
		t_str_3 := "moo_is_goo,loo_is_baz"
		outs := SortingTags{
			&Tag{Name: "moo", Value: "goo"},
			&Tag{Name: "loo", Value: "baz"},
		}
		Convey("spaced tags should parse", func() {
			tags := SortingTagsFromString(t_str)
			So(tags, ShouldResemble, outs)
		})
		Convey("dots tags should parse", func() {
			tags := SortingTagsFromString(t_str_1)
			So(tags, ShouldResemble, outs)
		})
		Convey("comma tags should parse", func() {
			tags := SortingTagsFromString(t_str_2)
			So(tags, ShouldResemble, outs)
		})
		Convey("_is_ tags should parse", func() {
			tags := SortingTagsFromString(t_str_3)
			So(tags, ShouldResemble, outs)
		})

		arr_str := []string{"moo=goo", "loo=baz"}
		Convey("array tags should parse", func() {
			tags := SortingTagsFromArray(arr_str)
			So(tags, ShouldResemble, outs)
		})
	})

	Convey("Tags Merging", t, func() {
		outs := SortingTags{
			&Tag{Name: "moo", Value: "goo"},
			&Tag{Name: "loo", Value: "baz"},
		}

		outs_m := SortingTags{
			&Tag{Name: "foo", Value: "bar"},
			&Tag{Name: "loo", Value: "MOOOO"},
		}

		outs_ok := SortingTags{
			&Tag{Name: "foo", Value: "bar"},
			&Tag{Name: "loo", Value: "MOOOO"},
			&Tag{Name: "moo", Value: "goo"},
		}

		Convey("should merge properly", func() {
			tags := outs.Merge(outs_m)
			sort.Sort(tags)
			So(tags, ShouldResemble, outs_ok)
		})

		Convey("should merge empties properly", func() {
			tags := outs.Merge(SortingTags{})
			sort.Sort(tags)
			So(tags, ShouldResemble, outs)
		})

	})

}
