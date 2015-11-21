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

func NewWriterConn(protocal string, host string, timeout time.Duration) (net.Conn, error) {
	if protocal == "tcp" || protocal == "udp" || protocal == "unix" {
		conn, err := net.DialTimeout(protocal, host, timeout)
		return conn, err
	} else if protocal == "http" || protocal == "https" {
		return NewWrtiterHttpConn(protocal, host, timeout)
	}
	return nil, fmt.Errorf("Invalid connection protocal")
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
func NewWrtiterHttpConn(protocal string, host string, timeout time.Duration) (*WrtiterHttpConn, error) {
	// the "host" may be url + path
	w := new(WrtiterHttpConn)
	w.Proto = protocal
	w.Host = host
	w.Timeout = timeout
	// the input may have a PATH as well, but we may need to host as some point
	w.url, _ = url.Parse(protocal + "://" + host)
	w.Method = "POST"
	w.tr = &http.Transport{}
	w.client = &http.Client{
		Transport: w.tr,
		Timeout:   timeout,
	}
	return w, nil
}

func (w *WrtiterHttpConn) Write(b []byte) (n int, err error) {
	defer w.tr.CloseIdleConnections()
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
