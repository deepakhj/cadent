package lrucache

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"regexp"
)

type TValue string

func (v TValue) Size(){
	return len(v)
}
func (v TValue) ToString(){
	return string(v)
}

func TestLRUCache(t *testing.T) {

	lru := NewLRUCache(5)

	var strs = []string{
		"mooooooooooo",
		"pooooooooooo",
		"go"
	}

	Convey("LRUcache should have capactiy", t, func() {
		So(lru.GetCapacity(), ShouldEqual, 5)

		Convey("LRUcache should accept some Keys", func() {

		}
	})


	conf["key_index"] = 10
	Convey("Graphite Runner should not parser this with a bad key index", t, func() {

		gr, _ := NewGraphiteRunner(conf)
		key, _, err := gr.ProcessLine(good_line)
		So(key, ShouldEqual, "")
		So(err, ShouldNotEqual, nil)
	})

}
