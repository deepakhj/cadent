package metrics

import (
	"cadent/server/repr"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestFileWriterAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls

	Convey("a dummy file writer", t, func() {
		file, _ := ioutil.TempFile("/tmp", "tt")
		f_name := file.Name()
		file.Close()
		fw, _ := NewMetrics("file")
		conf := make(map[string]interface{})

		st := repr.StatRepr{
			StatKey:    "goo",
			Sum:        5,
			Min:        1,
			Max:        3,
			Count:      4,
			Time:       time.Now(),
			Resolution: 2,
		}
		conf["max_file_size"] = int64(200)
		conf["rotate_every"] = time.Duration(time.Second)
		ok := fw.Config(conf)

		Convey("Should config fail", func() {
			So(ok, ShouldNotEqual, nil)
		})
		conf["dsn"] = f_name
		ok = fw.Config(conf)

		Convey("Should config", func() {
			So(ok, ShouldEqual, nil)
		})
		Convey("Should write", func() {
			err := fw.Write(st)
			So(err, ShouldEqual, nil)
		})

		//rotate should do it's thing
		for i := 0; i < 1000; i++ {
			fw.Write(st)
		}
		time.Sleep(time.Second)
		file.Close()
		os.Remove(file.Name())

	})

}
