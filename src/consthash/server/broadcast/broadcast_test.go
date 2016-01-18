package broadcast

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestBroadcasterer(t *testing.T) {

	bcast := New(1)
	var listeners []*Listener

	lister := func(list *Listener){
		for {
			select{
			case gots := <-list.Ch:
				t.Logf("Got broadcast %s: %v", list.id, gots)
				list.Close()
				return
			}
		}
	}
	for i := 0; i < 10; i++{
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
			So(func(){bcast.Send(true)}, ShouldPanic)
		})

	})
}