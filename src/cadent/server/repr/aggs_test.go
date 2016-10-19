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
)

func TestAggsRepr(t *testing.T) {

	test_list := make(map[string]uint32, 0)

	test_list["stats.gauges.moo.go"] = LAST
	test_list["stats.moo.moo.gauge"] = LAST
	test_list["stats.moo.moo.abs"] = LAST
	test_list["stats.timers.moo.go"] = MEAN
	test_list["stats.timers.moo.max_90"] = MAX
	test_list["stats.timers.moo.upper_90"] = MAX
	test_list["stats.timers.moo.lower"] = MIN
	test_list["stats.timers.moo.min"] = MIN
	test_list["stats.timers.moo.min_90"] = MIN
	test_list["stats.timers.moo.lower_90"] = MIN
	test_list["stats.timers.moo.upper_90"] = MAX
	test_list["stats.timers.moo.count_90"] = SUM
	test_list["stats.timers.moo.count"] = SUM
	test_list["stats.timers.moo.max"] = MAX
	test_list["stats.timers.moo.mean"] = MEAN
	test_list["stats.set.house"] = SUM
	test_list["stats.sets.house"] = SUM
	test_list["moo.goo.count"] = SUM
	test_list["moo.goo.errors"] = SUM
	test_list["moo.goo.ok"] = SUM
	test_list["moo.goo.upper"] = MAX
	test_list["moo.goo.inserts"] = SUM
	test_list["moo.goo.insert"] = SUM
	test_list["moo.goo.updated"] = SUM
	test_list["moo.goo.updates"] = SUM
	test_list["moo.goo.requests"] = SUM
	test_list["moo.goo.request"] = SUM
	test_list["moo.goo.set"] = SUM
	test_list["moo.goo.sets"] = SUM

	Convey("Aggregate Key Parsing", t, func() {

		for str, ff := range test_list {
			match := GuessReprValueFromKey(str)
			t.Logf("string: %s -> %v (want: %v)", str, match, ff)
			So(match, ShouldEqual, ff)
		}

	})

	tag_list := make(map[string]uint32, 0)

	tag_list["go"] = MEAN
	tag_list["gauge"] = LAST
	tag_list["abs"] = LAST
	tag_list["max_90"] = MAX
	tag_list["upper_90"] = MAX
	tag_list["lower"] = MIN
	tag_list["min"] = MIN
	tag_list["min_90"] = MIN
	tag_list["lower_90"] = MIN
	tag_list["upper_90"] = MAX
	tag_list["count_90"] = SUM
	tag_list["count"] = SUM
	tag_list["max"] = MAX
	tag_list["mean"] = MEAN
	tag_list["count"] = SUM
	tag_list["errors"] = SUM
	tag_list["ok"] = SUM
	tag_list["upper"] = MAX
	tag_list["inserts"] = SUM
	tag_list["insert"] = SUM
	tag_list["updated"] = SUM
	tag_list["updates"] = SUM
	tag_list["requests"] = SUM
	tag_list["request"] = SUM
	tag_list["set"] = SUM
	tag_list["sets"] = SUM

	Convey("Aggregate Tag Parsing", t, func() {

		for str, ff := range tag_list {
			match := AggTypeFromTag(str)
			t.Logf("string: %s -> %v (want: %v)", str, match, ff)
			So(match, ShouldEqual, ff)
		}

	})

}
