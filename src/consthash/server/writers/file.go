/*
	The File write

	will dump to an appended file
	stat\tsum\tmean\tmin\tmax\tcount\tresoltion\ttime


	OPTIONS: For `Config`

		prefix: filename prefix if any (_1s, _10s)
		max_file_size: Rotate the file if this size is met (default 100MB)
		rotate_every: Check to rotate the file every interval (default 10s)
*/

package writers

import (
	"consthash/server/repr"
	"fmt"
	logging "github.com/op/go-logging"
	"os"
	"sync"
	"time"
)

/****************** Interfaces *********************/
type FileWriter struct {
	fp       *os.File
	filename string
	prefix   string

	max_file_size int64 // file rotation
	write_lock    sync.Mutex
	rotate_check  time.Duration

	log *logging.Logger
}

// Make a new RotateWriter. Return nil if error occurs during setup.
func NewFileWriter() *FileWriter {
	fc := new(FileWriter)
	fc.log = logging.MustGetLogger("writers.file")
	return fc
}

func (fi *FileWriter) Config(conf map[string]interface{}) error {
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

func (fi *FileWriter) Filename() string {
	return fi.filename + fi.prefix
}

func (fi *FileWriter) PeriodicRotate() (err error) {
	for {
		time.Sleep(fi.rotate_check)
		err := fi.Rotate()
		if err != nil {
			fi.log.Error("File Rotate Error: %v", err)
		}
	}
	return
}

// Perform the actual act of rotating and reopening file.
func (fi *FileWriter) Rotate() (err error) {
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

func (fi *FileWriter) WriteLine(line string) (int, error) {
	fi.write_lock.Lock()
	defer fi.write_lock.Unlock()
	return fi.fp.Write([]byte(line))
}

func (fi *FileWriter) Write(stat repr.StatRepr) error {

	//	stat\tsum\tmean\tmin\tmax\tcount\tresoltion\ttime

	line := fmt.Sprintf(
		"%s\t%0.6f\t%0.6f\t%0.6f\t%0.6f\t%d\t%0.2f\t%d\t%d\n",
		stat.Key, stat.Sum, stat.Mean, stat.Min, stat.Max, stat.Count, stat.Resolution, stat.Time.UnixNano(), stat.TTL,
	)

	_, err := fi.WriteLine(line)

	return err
}
