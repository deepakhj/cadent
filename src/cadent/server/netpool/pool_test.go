package netpool

import (
	"bytes"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func FreeTCPPort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
func FreeUDPPort() int {
	addr, err := net.ResolveUDPAddr("udp", "localhost:0")
	if err != nil {
		panic(err)
	}

	return addr.Port
}

// random char gen
func RandChars(length uint) string {

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)

}

func TesterTCPMockListener(inurl string) (net.Listener, error) {
	i_url, _ := url.Parse(inurl)
	net, err := net.Listen(i_url.Scheme, i_url.Host+i_url.Path)
	if err != nil {
		return nil, err
	}
	log.Debug("Mock listener %v", i_url)

	return net, nil
}

func TesterUDPMockListener(inurl string) (*net.UDPConn, error) {
	udpArr, err := net.ResolveUDPAddr("udp", inurl)
	if err != nil {
		return nil, fmt.Errorf("Error binding: %s", err)
	}
	listener, err := net.ListenUDP("udp", udpArr)
	if err != nil {
		return nil, fmt.Errorf("Error binding: %s", err)
	}
	return listener, err
}

func TesterHTTPMockListener(inurl string) error {
	Handler := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		str, _ := ioutil.ReadAll(r.Body)
		log.Debug("HTTP out: %s", string(str))
	}
	http.HandleFunc("/", Handler)

	err := http.ListenAndServe(inurl, nil)
	log.Debug("Mock Http listener %v `%v`", inurl, err)

	return err
}

func MakeTesterServer(proto string, server string) net.Listener {
	// open a few random conns
	var srvs string
	var listens net.Listener
	var err error
	srvs = fmt.Sprintf("%s://%s", proto, server)
	listens, err = TesterTCPMockListener(srvs)
	if err != nil {
		panic(err)
	}

	handelConn := func(inCon net.Conn) {
		var buf bytes.Buffer
		io.Copy(&buf, inCon)
		log.Debug("Listener Out: %s -- len: %d", string(buf.Bytes()), buf.Len())
		inCon.Close()
	}

	go func() {
		for {
			conn, err := listens.Accept()
			if err == nil {
				log.Debug("Got Listener Connection: %v", conn)
				go handelConn(conn)
			}
		}
	}()
	return listens
}

func TestPoolerConns(t *testing.T) {

	free_p := FreeTCPPort()
	list := MakeTesterServer("tcp", fmt.Sprintf("127.0.0.1:%d", free_p))
	defer list.Close()

	free_p_2 := FreeUDPPort()
	udplist, _ := TesterUDPMockListener(fmt.Sprintf("127.0.0.1:%d", free_p_2))
	defer udplist.Close()

	file, _ := ioutil.TempFile("/tmp", "cadent_file_test")
	f_name := file.Name()
	file.Close()
	defer func(){ os.Remove(f_name) }()

	sock := fmt.Sprintf("%s.sock", f_name)
	socklist := MakeTesterServer("unix", sock)
	defer socklist.Close()
	defer os.Remove(sock)

	free_p_3 := FreeTCPPort()
	go TesterHTTPMockListener(fmt.Sprintf("127.0.0.1:%d", free_p_3))

	t_out := time.Duration(10 * time.Second)
	time.Sleep(time.Duration(time.Second)) // let things start up

	Convey("TCP Pool connections should", t, func() {
		_, err := NewWriterConn("moo", "goo", t_out)

		Convey("error on bad protocal", func() {
			So(err, ShouldNotBeNil)
		})

		conn, err := NewWriterConn("tcp", fmt.Sprintf("127.0.0.1:%d", free_p), t_out)
		Convey("get a real tcp client", func() {
			So(err, ShouldBeNil)
			So(conn.RemoteAddr().String(), ShouldEqual, fmt.Sprintf("127.0.0.1:%d", free_p))
		})

		n, err := conn.Write([]byte("moo"))
		Convey("tcp client can write", func() {
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 3)
		})

		err = conn.Close()
		Convey("tcp client can close", func() {
			So(err, ShouldBeNil)
		})
	})

	Convey("HTTP Pool connections should", t, func() {
		fconn, _ := NewWriterConn("http", "127.0.0.1:1239123/", t_out)
		_, ferr := fconn.Write([]byte("moo"))
		Convey("http client should fail", func() {
			So(ferr, ShouldNotBeNil)
		})

		hconn, herr := NewWriterConn("http", fmt.Sprintf("127.0.0.1:%d/", free_p_3), t_out)

		Convey("get a real http client", func() {
			So(herr, ShouldBeNil)
			So(hconn.RemoteAddr().String(), ShouldEqual, fmt.Sprintf("127.0.0.1:%d", free_p_3))
		})

		n, err := hconn.Write([]byte("moo"))
		Convey("http client can write", func() {
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 3)
		})
		Convey("http converage", func() {
			So(hconn.SetDeadline(time.Now()), ShouldBeNil)
			So(hconn.SetWriteDeadline(time.Now()), ShouldBeNil)
			So(hconn.SetReadDeadline(time.Now()), ShouldBeNil)
			So(hconn.Close(), ShouldBeNil)
			moo := make([]byte, 1)
			_, err = hconn.Read(moo)
			So(err, ShouldBeNil)
			So(hconn.RemoteAddr(), ShouldNotBeNil)
			hconn.LocalAddr()
		})

	})

	Convey("UDP Pool connections should", t, func() {

		hconn, herr := NewWriterConn("udp", fmt.Sprintf("127.0.0.1:%d", free_p_3), t_out)

		Convey("get a real udp client", func() {
			So(herr, ShouldBeNil)
			So(hconn.RemoteAddr().String(), ShouldEqual, fmt.Sprintf("127.0.0.1:%d", free_p_3))
		})

		n, err := hconn.Write([]byte("moo"))
		Convey("udp client can write", func() {
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 3)
		})

	})

	Convey("Socket Pool connections should", t, func() {

		hconn, herr := NewWriterConn("unix", sock, t_out)

		Convey("get a real socket client", func() {
			So(herr, ShouldBeNil)
			So(hconn.RemoteAddr().String(), ShouldEqual, sock)
		})

		n, err := hconn.Write([]byte("moo"))
		Convey("socket client can write", func() {
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 3)
		})

	})
}

func ConveyPoolerTester(t *testing.T, Outpool NetpoolInterface, name string, Numcons int) {
	Convey(name+" NetPool should", t, func() {

		Convey("have the proper max connections", func() {
			So(Outpool.GetMaxConnections(), ShouldEqual, Numcons)
		})

		err := Outpool.InitPool()
		Convey("init ok", func() {
			So(err, ShouldBeNil)
		})

		Convey("initially all free connections", func() {
			So(Outpool.NumFree(), ShouldEqual, Numcons)
		})

		have_con, err := Outpool.Open()
		Convey("get a connection", func() {
			So(have_con, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})

		err = Outpool.Close(have_con)
		Convey("should close", func() {
			So(err, ShouldBeNil)
		})

		Convey("have free connections", func() {
			So(Outpool.NumFree(), ShouldEqual, Numcons)
		})

		Outpool.DestroyAll()
		Convey("be destoyable", func() {
			So(Outpool.NumFree(), ShouldEqual, 0)
		})

		err = Outpool.InitPool()
		Convey("re-init ok", func() {
			So(err, ShouldBeNil)
		})

		have_con, err = Outpool.Open()
		Convey("have connection a proper index", func() {
			So(have_con.Index(), ShouldBeIn, []int{0, 1})
		})
		Convey("settable DealLine", func() {
			t := time.Now()

			So(have_con.SetWriteDeadline(t), ShouldBeNil)
		})

		Convey("have a resetable connection", func() {
			So(Outpool.ResetConn(have_con), ShouldBeNil)
		})

		Convey("have should close", func() {
			So(Outpool.Close(have_con), ShouldBeNil)
		})

		Convey("have a proper connection", func() {
			conn := have_con.Conn()
			So(conn, ShouldNotBeNil)
		})
		Convey("should be flushable", func() {
			_, err := have_con.Flush()
			So(err, ShouldBeNil)
		})
		Convey("have a writable connection", func() {

			_, err := have_con.Write([]byte{123})
			So(err, ShouldBeNil)
		})
		moo := RandChars(512)
		have_con.Write([]byte(moo))
		_, err = have_con.Write([]byte(moo))
		_, ferr := have_con.Flush()

		Convey("have a buffer flush write", func() {
			So(err, ShouldBeNil)
			So(ferr, ShouldBeNil)
		})

		have_con.SetConn(nil)

		Convey("have should reopen", func() {
			aux_conn, _ := Outpool.Open()
			So(aux_conn, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})

	})
}

func TestPooler(t *testing.T) {
	Numcons := 2

	list := MakeTesterServer("tcp", "127.0.0.1:30001")
	defer list.Close()

	Outpool := NewNetpool("tcp", "localhost:30001")
	Outpool.SetMaxConnections(Numcons)

	ConveyPoolerTester(t, Outpool, "BasicNetPool", Numcons)

	BOutpool := NewBufferedNetpool("tcp", "localhost:30001", 512)
	BOutpool.SetMaxConnections(Numcons)

	ConveyPoolerTester(t, BOutpool, "BufferedNetPool", Numcons)

	DeadPool := NewNetpool("tcp", "localhost:10")

	Convey("Dead NetPool should", t, func() {
		err := DeadPool.InitPool()
		Convey("init should fail", func() {
			So(err, ShouldNotBeNil)
		})

	})

}
