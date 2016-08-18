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
	"cadent/server/writers/indexer"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestFileWriterAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls
	Convey("a dummy file writer", t, func() {
		file, _ := ioutil.TempFile("/tmp", "cadent_file_test")
		f_name := file.Name()
		file.Close()
		fw, _ := NewMetrics("file")

		idx, _ := indexer.NewIndexer("noop")
		conf := make(map[string]interface{})
		conf["max_file_size"] = int64(200)

		ok := fw.Config(conf)

		Convey("Config Should config fail", func() {
			So(ok, ShouldNotEqual, nil)
		})

		conf["dsn"] = f_name
		conf["rotate_every"] = time.Duration(time.Second)
		ok = fw.Config(conf)
		fw.SetIndexer(idx)

		Convey("Config Should not fail", func() {
			So(ok, ShouldEqual, nil)
		})

		st := repr.StatRepr{
			Time:  time.Now(),
			Name:  repr.StatName{Key: "goo", Resolution: 2},
			Sum:   5,
			Min:   1,
			Max:   3,
			Count: 4,
		}

		Convey("Should write", func() {
			err := fw.Write(st)
			So(err, ShouldEqual, nil)
		})

		//rotate should do it's thing
		for i := 0; i < 1000; i++ {
			fw.Write(st)
		}

		time.Sleep(time.Second)
		fw.Stop()
		os.Remove(file.Name())
		fw = nil

	})

}
