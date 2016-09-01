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

/** HTTP compression handlers **/

package api

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// mock struct to be a writer interface
type compressionWriter struct {
	io.Writer
	http.ResponseWriter
	http.Hijacker
	http.Flusher
	http.CloseNotifier
}

func (w *compressionWriter) WriteHeader(c int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(c)
}

func (w *compressionWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *compressionWriter) Write(b []byte) (int, error) {
	h := w.ResponseWriter.Header()
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", http.DetectContentType(b))
	}
	h.Del("Content-Length")

	return w.Writer.Write(b)
}

type flusher interface {
	Flush() error
}

func (w *compressionWriter) Flush() {
	// Flush compressed data if compressor supports it.
	if f, ok := w.Writer.(flusher); ok {
		f.Flush()
	}
	// Flush HTTP response.
	if w.Flusher != nil {
		w.Flusher.Flush()
	}
}

func CompressHandler(h http.Handler) http.Handler {
	level := gzip.DefaultCompression

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	END:
		for _, enc := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
			switch strings.TrimSpace(enc) {
			case "gzip":
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Add("Vary", "Accept-Encoding")

				gw, _ := gzip.NewWriterLevel(w, level)
				defer gw.Close()

				h, hok := w.(http.Hijacker)
				if !hok {
					h = nil
				}

				f, fok := w.(http.Flusher)
				if !fok {
					f = nil
				}

				cn, cnok := w.(http.CloseNotifier)
				if !cnok {
					cn = nil
				}

				w = &compressionWriter{
					Writer:         gw,
					ResponseWriter: w,
					Hijacker:       h,
					Flusher:        f,
					CloseNotifier:  cn,
				}

				break END
			case "deflate":
				w.Header().Set("Content-Encoding", "deflate")
				w.Header().Add("Vary", "Accept-Encoding")

				fw, _ := flate.NewWriter(w, level)
				defer fw.Close()

				h, hok := w.(http.Hijacker)
				if !hok {
					h = nil
				}

				f, fok := w.(http.Flusher)
				if !fok {
					f = nil
				}

				cn, cnok := w.(http.CloseNotifier)
				if !cnok {
					cn = nil
				}

				w = &compressionWriter{
					Writer:         fw,
					ResponseWriter: w,
					Hijacker:       h,
					Flusher:        f,
					CloseNotifier:  cn,
				}

				break END
			}
		}

		h.ServeHTTP(w, r)
	})
}
