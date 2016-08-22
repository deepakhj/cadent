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
	The File write

	will dump to an appended file
	stat\tsum\tmin\tmax\tcount\tlast\tresoltion\ttime


	OPTIONS: For `Config`

		prefix: filename prefix if any (_1s, _10s)
		max_file_size: Rotate the file if this size is met (default 100MB)
		rotate_every: Check to rotate the file every interval (default 10s)
*/

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/repr"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/indexer"
	"errors"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"os"
	"sync"
	"time"
)

var errNoFilePointer = errors.New("Cannot write point, no file pointer")
var errFileReaderNotImplemented = errors.New("FILE READER NOT IMPLMENTED")

/****************** Interfaces *********************/
type FileMetrics struct {
	fp                *os.File
	filename          string
	prefix            string
	indexer           indexer.Indexer
	resolutions       [][]int
	currentResolution int

	max_file_size int64 // file rotation
	write_lock    sync.Mutex
	rotate_check  time.Duration
	shutdown      *broadcast.Broadcaster
	startstop     utils.StartStop

	log *logging.Logger
}

// Make a new RotateWriter. Return nil if error occurs during setup.
func NewFileMetrics() *FileMetrics {
	fc := new(FileMetrics)
	fc.shutdown = broadcast.New(1)
	fc.log = logging.MustGetLogger("writers.file")
	return fc
}

// TODO
func (fi *FileMetrics) Stop() {
	fi.startstop.Stop(func() {
		shutdown.AddToShutdown()

		fi.shutdown.Send(true)
	})
}

func (fi *FileMetrics) Start() {
	fi.startstop.Start(func() {
		go fi.PeriodicRotate()
	})
}

func (fi *FileMetrics) SetIndexer(idx indexer.Indexer) error {
	fi.indexer = idx
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (fi *FileMetrics) SetResolutions(res [][]int) int {
	fi.resolutions = res
	return len(res) // need as many writers as bins
}

func (fi *FileMetrics) SetCurrentResolution(res int) {
	fi.currentResolution = res
}

func (fi *FileMetrics) Config(conf map[string]interface{}) error {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (/path/to/file) needed for FileWriter")
	}
	dsn := gots.(string)
	fi.filename = dsn

	_wr_buffer := conf["max_file_size"]
	if _wr_buffer == nil {
		fi.max_file_size = 100 * 1024 * 1024
	} else {
		// toml things generic ints are int64
		fi.max_file_size = _wr_buffer.(int64)
	}
	// file prefix
	_pref := conf["prefix"]
	if _pref == nil {
		fi.prefix = ""
	} else {
		fi.prefix = _pref.(string)
	}

	fi.rotate_check = time.Duration(10 * time.Second)
	_rotate := conf["rotate_every"]
	if _rotate != nil {
		fi.rotate_check = _rotate.(time.Duration)
	}
	fi.fp = nil
	fi.Rotate()

	return nil
}

func (fi *FileMetrics) Filename() string {
	return fi.filename + fi.prefix
}

func (fi *FileMetrics) PeriodicRotate() (err error) {
	shuts := fi.shutdown.Listen()
	ticks := time.NewTicker(fi.rotate_check)
	for {
		select {
		case <-shuts.Ch:
			shuts.Close()
			fi.log.Warning("Shutting down file writer...")
			if fi.fp != nil {
				fi.fp.Close()
				fi.fp = nil
			}
			shutdown.ReleaseFromShutdown()
			return
		case <-ticks.C:
			err := fi.Rotate()
			if err != nil {
				fi.log.Error("File Rotate Error: %v", err)
			}
		}
	}
}

// Perform the actual act of rotating and reopening file.
func (fi *FileMetrics) Rotate() (err error) {
	fi.write_lock.Lock()
	defer fi.write_lock.Unlock()

	f_name := fi.Filename()
	// Close existing file if open
	if fi.fp != nil {
		// check the size and rotate if too big
		info, err := os.Stat(f_name)
		if err == nil && info.Size() > fi.max_file_size {
			err = fi.fp.Close()
			fi.fp = nil
			if err != nil {
				return err
			}
			err = os.Rename(f_name, f_name+"."+time.Now().Format("20060102150405"))
			if err != nil {
				return err
			}

			// Create a new file.
			fi.fp, err = os.Create(f_name)
			fi.log.Notice("Rotated file %s", f_name)
			return err
		}
	}

	// if the file exists, it's the old one
	fi.fp, err = os.OpenFile(f_name, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err == nil {
		fi.fp.Seek(0, 2) // seek to end
		return nil
	} else {
		fi.fp, err = os.Create(f_name)
		fi.log.Notice("Started file writer %s", f_name)

		return err
	}
}

func (fi *FileMetrics) WriteLine(line string) (int, error) {
	fi.write_lock.Lock()
	defer fi.write_lock.Unlock()
	if fi.fp == nil {
		return 0, errNoFilePointer
	}
	return fi.fp.Write([]byte(line))
}

func (fi *FileMetrics) Write(stat repr.StatRepr) error {

	// stat\tuid\tsum\tmin\tmax\tlast\tcount\tresoltion\ttime\tttl

	line := fmt.Sprintf(
		"%s\t%s\t%0.6f\t%0.6f\t%0.6f\t%0.6f\t%d\t%d\t%d\t%d\n",
		stat.Name.Key, stat.Name.UniqueIdString(), stat.Sum, stat.Min, stat.Max, stat.Last, stat.Count,
		stat.Name.Resolution, stat.Time.UnixNano(), stat.Name.TTL,
	)

	fi.indexer.Write(stat.Name) // index me
	_, err := fi.WriteLine(line)

	return err
}

/**** READER ***/

func (fi *FileMetrics) Render(path string, from int64, to int64) (WhisperRenderItem, error) {
	return WhisperRenderItem{}, errFileReaderNotImplemented
}
func (fi *FileMetrics) RawRender(path string, from int64, to int64) ([]*RawRenderItem, error) {
	return nil, errFileReaderNotImplemented
}
func (fi *FileMetrics) CacheRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error) {
	return nil, errFileReaderNotImplemented
}
func (fi *FileMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*TotalTimeSeries, error) {
	return nil, errFileReaderNotImplemented
}
