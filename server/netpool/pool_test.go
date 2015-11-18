package netpool

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"net"
	"net/url"
	"testing"
	"time"
)

func TesterTCPMockListener(inurl string) (net.Listener, error) {
	i_url, _ := url.Parse(inurl)
	net, err := net.Listen(i_url.Scheme, i_url.Host)
	if err != nil {
		return nil, err
	}
	log.Debug("Mock listener %v", i_url)

	return net, nil
}

func ConveyTester(t *testing.T, Outpool NetpoolInterface, name string, Numcons int) {
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

		Convey("have a writable connection", func() {
			_, err := have_con.Write([]byte{123})
			So(err, ShouldBeNil)
		})

		Convey("have a proper connection", func() {
			conn := have_con.Conn()
			So(conn, ShouldNotBeNil)
		})
		Convey("should be flushable", func() {
			_, err := have_con.Flush()
			So(err, ShouldBeNil)
		})
	})
}

func TestPooler(t *testing.T) {

	// open a few random conns
	r_port := 30001
	var srvs string
	var i_url *url.URL
	var listens net.Listener
	var err error
	for {
		srvs = fmt.Sprintf("tcp://127.0.0.1:%d", r_port)
		i_url, _ = url.Parse(srvs)
		listens, err = TesterTCPMockListener(srvs)
		if err == nil {
			break
		}
		r_port += 1

	}
	defer listens.Close()

	Numcons := 2

	handelConn := func(inCon net.Conn) {
		buf := make([]byte, 1024)
		_, err := inCon.Read(buf)
		if err != nil {

		}
		inCon.Close()
	}

	go func() {
		for {
			conn, err := listens.Accept()
			if err == nil {
				go handelConn(conn)
			}
		}
	}()

	Outpool := NewNetpool(i_url.Scheme, i_url.Host)
	Outpool.SetMaxConnections(Numcons)

	ConveyTester(t, Outpool, "BasicNetPool", Numcons)

	BOutpool := NewBufferedNetpool(i_url.Scheme, i_url.Host, 512)
	BOutpool.SetMaxConnections(Numcons)

	ConveyTester(t, BOutpool, "BufferedNetPool", Numcons)

	DeadPool := NewNetpool("tcp", "localhost:10")

	Convey("Dead NetPool should", t, func() {
		err := DeadPool.InitPool()
		Convey("init should fail", func() {
			So(err, ShouldNotBeNil)
		})

	})

}
