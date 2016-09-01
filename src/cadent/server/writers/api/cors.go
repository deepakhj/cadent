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

/** HTTP (simple) Cors **/

package api

import (
	"io"
	"net/http"
)

// mock struct to be a writer interface
type corsWriter struct {
	io.Writer
	http.ResponseWriter
	http.Hijacker
	http.Flusher
	http.CloseNotifier
}

func (w *corsWriter) WriteHeader(c int) {
	w.ResponseWriter.WriteHeader(c)
}

func (w *corsWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *corsWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *corsWriter) Flush() {
	// Flush compressed data if compressor supports it.
	if f, ok := w.Writer.(flusher); ok {
		f.Flush()
	}
	// Flush HTTP response.
	if w.Flusher != nil {
		w.Flusher.Flush()
	}
}

func CorsHandler(h http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET, OPTIONS")

		hi, hok := w.(http.Hijacker)
		if !hok { /* w is not Hijacker... oh well... */
			hi = nil
		}

		cn, cnok := w.(http.CloseNotifier)
		if !cnok {
			cn = nil
		}
		f, fok := w.(http.Flusher)
		if !fok {
			f = nil
		}

		w = &corsWriter{
			Writer:         w,
			ResponseWriter: w,
			Hijacker:       hi,
			Flusher:        f,
			CloseNotifier:  cn,
		}

		h.ServeHTTP(w, r)
	})
}
