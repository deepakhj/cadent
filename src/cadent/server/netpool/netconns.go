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

package netpool

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

func NewWriterConn(protocol string, host string, timeout time.Duration) (net.Conn, error) {
	if protocol == "tcp" || protocol == "udp" || protocol == "unix" {
		conn, err := net.DialTimeout(protocol, host, timeout)
		return conn, err
	} else if protocol == "http" || protocol == "https" {
		return NewWriterHttpConn(protocol, host, timeout)
	}
	return nil, fmt.Errorf("Invalid connection protocol")
}

/** the net.Conn interface for ease

Read(b []byte) (n int, err error)
Write(b []byte) (n int, err error)
Close() error
LocalAddr() Addr
RemoteAddr() Addr
SetDeadline(t time.Time) error
SetReadDeadline(t time.Time) error
SetWriteDeadline(t time.Time) error

*/

/************************/
/****** HTTP CONN *******/
/************************/

// a special http "writter" that acts like a net.Conn for writing
type WrtiterHttpConn struct {
	Timeout time.Duration
	Proto   string
	Host    string
	Method  string

	url    *url.URL
	tr     *http.Transport
	client *http.Client

	t int
}

// net.Addr interfaceer
type HTTPAddr struct {
	Name string
}

func (ha HTTPAddr) Network() string {
	return "tcp"
}
func (ha HTTPAddr) String() string {
	return ha.Name
}

// basically delgate the dial timeout
func NewWriterHttpConn(protocol string, host string, timeout time.Duration) (*WrtiterHttpConn, error) {
	// the "host" may be url + path
	w := new(WrtiterHttpConn)
	w.Proto = protocol
	w.Host = host
	w.Timeout = timeout
	// the input may have a PATH as well, but we may need to host as some point
	w.url, _ = url.Parse(protocol + "://" + host)
	w.Method = "POST"
	w.tr = &http.Transport{}
	w.client = &http.Client{
		Transport: w.tr,
		Timeout:   timeout,
	}
	// start the closer
	go w.closeIdle()
	return w, nil
}

// periodically close idle cons to avoid leakage
func (w *WrtiterHttpConn) closeIdle() {
	w.tr.CloseIdleConnections()
	time.Sleep(time.Duration(1 * time.Second))
	w.closeIdle()
}

func (w *WrtiterHttpConn) Write(b []byte) (n int, err error) {
	reader := bytes.NewReader(b)
	req, err := http.NewRequest(w.Method, w.url.Scheme+"://"+w.url.Host+w.url.Path, reader)
	//log.Debug(w.url.Scheme+"://"+w.url.Host+w.url.Path)
	if err != nil {
		log.Error("Request failed to construct: %s", err)
		return 0, err
	}

	res, err := w.client.Do(req)

	if err != nil {
		log.Error("post request failed: `%s`", err)
		return 0, err
	}
	io.Copy(ioutil.Discard, res.Body)
	res.Body.Close()
	return len(b), nil
}

func (w *WrtiterHttpConn) LocalAddr() net.Addr {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return addrs[0]
	}
	return nil
}
func (w *WrtiterHttpConn) RemoteAddr() net.Addr {
	return HTTPAddr{
		Name: w.url.Host,
	}
}

func (w *WrtiterHttpConn) Close() (err error) {
	return nil
}

/** noops for this */
func (w *WrtiterHttpConn) Read(b []byte) (n int, err error) {
	// not reading
	return 0, nil
}

func (w *WrtiterHttpConn) SetDeadline(t time.Time) error {
	return nil
}

func (w *WrtiterHttpConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (w *WrtiterHttpConn) SetWriteDeadline(t time.Time) error {
	return nil
}
