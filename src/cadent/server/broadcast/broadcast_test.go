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

package broadcast

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestBroadcasterer(t *testing.T) {

	bcast := New(1)
	var listeners []*Listener

	lister := func(list *Listener) {
		for {
			select {
			case gots := <-list.Ch:
				t.Logf("Got broadcast %d: %v", list.id, gots)
				list.Close()
				return
			}
		}
	}
	for i := 0; i < 10; i++ {
		list := bcast.Listen()
		listeners = append(listeners, list)
		go lister(list)
	}
	Convey("Broadcaster should", t, func() {

		Convey("have 10 listeners", func() {
			So(len(bcast.listeners), ShouldEqual, 10)
		})
		Convey("Should broadcast", func() {
			bcast.Send(true)
			So(len(bcast.listeners), ShouldEqual, 10)
		})
		Convey("Should close and send shoud fail", func() {
			bcast.Close()
			So(func() { bcast.Send(true) }, ShouldPanic)
		})

	})
}
