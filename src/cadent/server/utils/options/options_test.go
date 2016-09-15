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

/*
 little helper to get "options" from a map[string]interface{}
*/

package options

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestUtilsOptions(t *testing.T) {
	// Only pass t into top-level Convey calls
	Convey("Setting and Getting Options", t, func() {

		ops := New()
		ops["int64"] = 123
		ops["string"] = "abc"
		ops["bool"] = true
		ops["dur"] = "2s"
		ops["float"] = 123123.23

		So(ops.Bool("bool", false), ShouldEqual, true)
		So(ops.Int64("int64", 343), ShouldEqual, 123)
		So(ops.Float64("float", .9), ShouldEqual, 123123.23)
		So(ops.Float64("MOO", .9), ShouldEqual, 0.9)
		So(ops.Bool("MOO", false), ShouldEqual, false)

		_, err := ops.BoolRequired("MOO")
		So(err, ShouldNotEqual, nil)

		_, err = ops.Float64Required("float")
		So(err, ShouldEqual, nil)

		_, err = ops.Float64Required("floatsdf")
		So(err, ShouldNotEqual, nil)

		_, err = ops.Int64Required("int64")
		So(err, ShouldEqual, nil)

		_, err = ops.Int64Required("int64sdfsdf")
		So(err, ShouldNotEqual, nil)

		dd := ops.Duration("dur", time.Duration(0))
		So(dd, ShouldEqual, time.Duration(2)*time.Second)

		_, err = ops.Float64Required("int64sdfsdf")
		So(err, ShouldNotEqual, nil)

		ops.Set("monkey", 123456)
		So(ops.Int64("monkey", 3), ShouldEqual, 123456)

	})
}
