/*
   Writers/Readers of stats

   We are attempting to mimic the Graphite API json blobs throughout the process here
   such that we can hook this in directly to either graphite-api/web

   NOTE: this is not a full graphite DSL, just paths and metrics, we leave the fancy functions inside
   graphite-api/web to work their magic .. one day we'll impliment the full DSL, but until then ..

   Currently just implimenting /find /expand and /render (json only) for graphite-api
*/

package readers

import (
	"fmt"
	golanglog "log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

/****************** Helpers *********************/
// parses things like "-23h" etc
// only does "second" precision which is all graphite can do currently
func ParseTime(st string) (int64, error) {
	st = strings.Trim(strings.ToLower(st), " \n\t")
	unix_t := int64(time.Now().Unix())
	if st == "now" {
		return unix_t, nil
	}

	if strings.HasSuffix(st, "s") || strings.HasSuffix(st, "sec") || strings.HasSuffix(st, "second") {
		items := strings.Split(st, "s")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i, nil
	}

	if strings.HasSuffix(st, "m") || strings.HasSuffix(st, "min") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60, nil
	}

	if strings.HasSuffix(st, "h") || strings.HasSuffix(st, "hour") {
		items := strings.Split(st, "h")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60, nil
	}

	if strings.HasSuffix(st, "d") || strings.HasSuffix(st, "day") {
		items := strings.Split(st, "d")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60*24, nil
	}
	if strings.HasSuffix(st, "mon") || strings.HasSuffix(st, "month") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60*24*30, nil
	}
	if strings.HasSuffix(st, "y") || strings.HasSuffix(st, "year") {
		items := strings.Split(st, "y")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60*24*365, nil
	}

	// if it's an int already, we're good
	i, err := strconv.ParseInt(st, 10, 64)
	if err == nil {
		return i, nil
	}

	return 0, fmt.Errorf("Time `%s` could not be parsed :: %v", st, err)

}

/** HTTP loggers **/

// mock struct to be a writer interface
type statusWriter struct {
	http.ResponseWriter
	status int
	length int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	w.length = len(b)
	return w.ResponseWriter.Write(b)
}

// WriteLog Logs the Http Status for a request into fileHandler and returns a httphandler function which is a wrapper to log the requests.
func WriteLog(handle http.Handler, fileHandler *os.File) http.HandlerFunc {
	logger := golanglog.New(fileHandler, "", 0)
	return func(w http.ResponseWriter, request *http.Request) {
		start := time.Now()
		writer := statusWriter{w, 0, 0}
		handle.ServeHTTP(&writer, request)
		end := time.Now()
		latency := end.Sub(start)
		statusCode := writer.status
		length := writer.length
		if request.URL.RawQuery != "" {
			logger.Printf(
				"%v %s %s \"%s %s%s%s %s\" %d %d \"%s\" %v",
				end.Format("2006/01/02 15:04:05"),
				request.Host,
				request.RemoteAddr,
				request.Method,
				request.URL.Path, "?",
				request.URL.RawQuery,
				request.Proto,
				statusCode,
				length, request.Header.Get("User-Agent"),
				latency,
			)
		} else {
			logger.Printf(
				"%v %s %s \"%s %s %s\" %d %d \"%s\" %v",
				end.Format("2006/01/02 15:04:05"),
				request.Host,
				request.RemoteAddr,
				request.Method,
				request.URL.Path,
				request.Proto,
				statusCode,
				length,
				request.Header.Get("User-Agent"),
				latency,
			)
		}
	}
}
