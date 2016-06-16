/*
	The File write

	will dump to an appended file
	stat\tsum\tmean\tmin\tmax\tcount\tresoltion\ttime


	OPTIONS: For `Config`

		prefix: filename prefix if any (_1s, _10s)
		max_file_size: Rotate the file if this size is met (default 100MB)
		rotate_every: Check to rotate the file every interval (default 10s)
*/

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/repr"
	"cadent/server/writers/indexer"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"os"
	"sync"
	"time"
)

/****************** Interfaces *********************/
type FileMetrics struct {
	fp          *os.File
	filename    string
	prefix      string
	indexer     indexer.Indexer
	resolutions [][]int

	max_file_size int64 // file rotation
	write_lock    sync.Mutex
	rotate_check  time.Duration
	shutdown      *broadcast.Broadcaster

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
	fi.shutdown.Send(true)
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

	go fi.PeriodicRotate()

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
			break
		case <-ticks.C:
			err := fi.Rotate()
			if err != nil {
				fi.log.Error("File Rotate Error: %v", err)
			}
		}
	}
	return nil
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

	return
}

func (fi *FileMetrics) WriteLine(line string) (int, error) {
	fi.write_lock.Lock()
	defer fi.write_lock.Unlock()
	if fi.fp == nil {
		return 0, fmt.Errorf("Cannot write point, no file pointer")
	}
	return fi.fp.Write([]byte(line))
}

func (fi *FileMetrics) Write(stat repr.StatRepr) error {

	// stat\tsum\tmean\tmin\tmax\tcount\tresoltion\ttime\tttl

	line := fmt.Sprintf(
		"%s\t%0.6f\t%0.6f\t%0.6f\t%0.6f\t%d\t%0.2f\t%d\t%d\n",
		stat.Key, stat.Sum, stat.Mean, stat.Min, stat.Max, stat.Count,
		stat.Resolution, stat.Time.UnixNano(), stat.TTL,
	)

	fi.indexer.Write(stat.Key) // index me
	_, err := fi.WriteLine(line)

	return err
}

/**** READER ***/
// XXX TODO
func (my *FileMetrics) Render(path string, from string, to string) (WhisperRenderItem, error) {
	return WhisperRenderItem{}, fmt.Errorf("FILE READER NOT YET DONE")
}
func (my *FileMetrics) RawRender(path string, from string, to string) ([]*RawRenderItem, error) {
	return []*RawRenderItem{}, fmt.Errorf("FILE READER NOT YET DONE")
}
