/*
Here we have a NetPooler but that buffers writes before sending things in specifc chunks

*/

package main

import (
	"net"
	"sync"
	"time"
)

const DEFAULT_BUFFER_SIZE = 512
const DEFAULT_FORCE_FLUSH = time.Second

type BufferedNetpool struct {
	pool           *Netpool
	BufferSize     int
	ForceFlushTime time.Duration

	//lock to grab all the active cons and flush them
	flushLock sync.Mutex
}

type BufferedNetpoolConn struct {
	conn        net.Conn
	started     time.Time
	idx         int
	writebuffer []byte
}

func NewBufferedNetpoolConn(conn net.Conn) NetpoolConnInterface {
	bconn := &BufferedNetpoolConn{
		conn:        conn,
		started:     time.Now(),
		writebuffer: make([]byte, 0),
	}

	//set up the flush timer
	go bconn.PeriodicFlush()
	return bconn
}

// periodically flush data no matter what
func (n *BufferedNetpoolConn) PeriodicFlush() {
	for {
		n.Flush()
		//log.Println("Flushed: ", n.idx)
		time.Sleep(DEFAULT_FORCE_FLUSH)
	}
}

func (n *BufferedNetpoolConn) Conn() net.Conn        { return n.conn }
func (n *BufferedNetpoolConn) SetConn(conn net.Conn) { n.conn = conn }

func (n *BufferedNetpoolConn) Started() time.Time     { return n.started }
func (n *BufferedNetpoolConn) SetStarted(t time.Time) { n.started = t }

func (n *BufferedNetpoolConn) Index() int     { return n.idx }
func (n *BufferedNetpoolConn) SetIndex(i int) { n.idx = i }

func (n *BufferedNetpoolConn) SetWriteDeadline(t time.Time) error {
	return n.conn.SetWriteDeadline(t)
}

func (n *BufferedNetpoolConn) Write(b []byte) (wrote int, err error) {

	if len(n.writebuffer) > DEFAULT_BUFFER_SIZE {
		//log.Println("Conn: ", n.conn.RemoteAddr(), "Wrote: ", wrote, " bin:", len(b), " buffsize: ", len(n.writebuffer))
		wrote, err = n.conn.Write(n.writebuffer)
		n.writebuffer = []byte("")
	}
	n.writebuffer = append(n.writebuffer, b...)

	return wrote, err
}

func (n *BufferedNetpoolConn) Flush() (wrote int, err error) {
	if len(n.writebuffer) > 0 {
		wrote, err = n.conn.Write(n.writebuffer)
		n.writebuffer = []byte("")
	}
	return wrote, err
}

/*** Poooler ***/

func NewBufferedNetpool(protocal string, name string) *BufferedNetpool {
	pool := NewNetpool(protocal, name)
	//override
	pool.newConnectionFunc = NewBufferedNetpoolConn

	bpool := &BufferedNetpool{
		pool:           pool,
		BufferSize:     DEFAULT_BUFFER_SIZE,
		ForceFlushTime: DEFAULT_FORCE_FLUSH,
	}
	return bpool
}

func (n *BufferedNetpool) GetMaxConnections() int {
	return n.pool.GetMaxConnections()
}
func (n *BufferedNetpool) SetMaxConnections(maxconn int) {
	n.pool.SetMaxConnections(maxconn)
}

//proxy to pool
func (n *BufferedNetpool) NumFree() int {
	return len(n.pool.free)
}

// reset and clear a connection .. proxy to pool
func (n *BufferedNetpool) ResetConn(net_conn NetpoolConnInterface) error {
	return n.pool.ResetConn(net_conn)
}

// proxy to pool
func (n *BufferedNetpool) InitPool() error {
	err := n.pool.InitPool()

	if err != nil {
		return err
	}
	return err
}

func (n *BufferedNetpool) Open() (conn NetpoolConnInterface, err error) {
	return n.pool.Open()
}

//add it back to the queue
func (n *BufferedNetpool) Close(conn NetpoolConnInterface) error {
	return n.pool.Close(conn)
}
