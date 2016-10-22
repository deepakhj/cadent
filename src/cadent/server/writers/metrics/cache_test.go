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
	"testing"
	"time"
)

func TestCacher(t *testing.T) {

	nm := repr.StatName{Key: "moo.goo.org", Resolution: 1, Ttl: 1000}
	rep := repr.StatRepr{
		Count: 1,
		Sum:   100,
		Name:  &nm,
	}
	tmp, _ := GetCacherSingleton("myniftyname", "chunk")
	c := tmp.(*CacherChunk)

	Convey("CacherChunk should function basically", t, func() {
		So(c, ShouldNotEqual, nil)

		c.SetName("monkey")
		So(c.Name, ShouldEqual, "monkey")

		c.SetSeriesEncoding("gob")

		err := c.Add(&nm, &rep)

		So(err, ShouldEqual, nil)

		So(len(c.curSlice), ShouldEqual, 1)
		So(c.curChunk.Len(), ShouldEqual, 1)

	})

	Convey("CacherChunk should emit log messages", t, func() {

		c.SetMaxChunks(2)
		c.SetLogTimeWindow(2)
		c.Start()
		lchan := c.GetLogChan()

		for i := int64(0); i < 100; i++ {
			repCp := rep
			repCp.Time = time.Now().UnixNano() + i*int64(10*time.Second)
			err := c.Add(&nm, &repCp)
			So(err, ShouldEqual, nil)
		}
		// myniftyname was already inited above
		So(len(c.curSlice[nm.UniqueId()]), ShouldEqual, 101)
		So(c.curChunk.Len(), ShouldEqual, 1)
		So(c.curChunk.ts[nm.UniqueId()].Series.Count(), ShouldEqual, 101)

		gots := <-lchan.Ch
		g := gots.(*CacheChunkLog)
		So(g.SequenceId, ShouldEqual, 0)
		So(len(g.Slice), ShouldEqual, 1)
		So(len(g.Slice[nm.UniqueId()]), ShouldEqual, 101)
		So(len(c.curSlice), ShouldEqual, 0)
		lchan.Close()

	})
}
