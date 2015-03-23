/*
Here we have a NetPooler but that buffers writes before sending things in specifc chunks

*/

package netpool

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const DEFAULT_BUFFER_SIZE = 512
const DEFAULT_FORCE_FLUSH = time.Second

type BufferedNetpool struct {
	pool           *Netpool
	BufferSize     int
	ForceFlushTime time.Duration
	didclose       bool

	//lock to grab all the active cons and flush them
	flushLock sync.Mutex
}

type BufferedNetpoolConn struct {
	conn        net.Conn
	started     time.Time
	idx         int
	writebuffer []byte
	buffersize  int

	writeLock sync.Mutex
}

func NewBufferedNetpoolConn(conn net.Conn, pool NetpoolInterface) NetpoolConnInterface {
	bconn := &BufferedNetpoolConn{
		conn:        conn,
		started:     time.Now(),
		writebuffer: make([]byte, 0),
		buffersize:  DEFAULT_BUFFER_SIZE,
	}
	if pool.(*BufferedNetpool).BufferSize > 0 {
		bconn.buffersize = pool.(*BufferedNetpool).BufferSize
	}
	//log.Printf("BufferSize: ", bconn.buffersize)
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

	//n.writeLock.Lock()
	//defer n.writeLock.Unlock()
	if len(n.writebuffer) > n.buffersize {
		//log.Println("Conn: ", n.conn.RemoteAddr(), "Wrote: ", wrote, " bin:", len(b), " buffsize: ", len(n.writebuffer))
		wrote, err = n.conn.Write(n.writebuffer)
		n.writebuffer = []byte("")
	}
	n.writebuffer = append(n.writebuffer, b...)

	return wrote, err
}

func (n *BufferedNetpoolConn) Flush() (wrote int, err error) {
	n.writeLock.Lock()
	defer n.writeLock.Unlock()
	if len(n.writebuffer) > 0 && n.conn != nil {
		wrote, err = n.conn.Write(n.writebuffer)
		n.writebuffer = []byte("")
	}
	return wrote, err
}

/*** Pooler ***/

func NewBufferedNetpool(protocal string, name string, buffersize int) *BufferedNetpool {
	pool := NewNetpool(protocal, name)
	//override
	pool.newConnectionFunc = NewBufferedNetpoolConn

	bpool := &BufferedNetpool{
		pool:           pool,
		BufferSize:     DEFAULT_BUFFER_SIZE,
		ForceFlushTime: DEFAULT_FORCE_FLUSH,
		didclose:       false,
	}
	if buffersize > 0 {
		bpool.BufferSize = buffersize
	}

	bpool.TrapExit()
	return bpool
}

func (n *BufferedNetpool) TrapExit() {
	//trap kills to flush the buffer
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		s := <-sigc
		log.Printf("Caught %s: Flushing Buffers before quit ", s)
		close(n.pool.free)
		n.didclose = true
		for con := range n.pool.free {
			con.Flush()
		}

		signal.Stop(sigc)
		close(sigc)

		// re-raise it
		process, _ := os.FindProcess(os.Getpid())
		process.Signal(s)
	}()
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

// alow us to put _this_ object into the init of a Connection dunction
func (n *BufferedNetpool) InitPoolWith(obj NetpoolInterface) error {
	return n.pool.InitPoolWith(obj)
}

// proxy to pool
func (n *BufferedNetpool) InitPool() error {
	return n.pool.InitPoolWith(n) //use our object not the pool
}

func (n *BufferedNetpool) Open() (conn NetpoolConnInterface, err error) {
	return n.pool.Open()
}

//add it back to the queue
func (n *BufferedNetpool) Close(conn NetpoolConnInterface) error {
	if !n.didclose {
		return n.pool.Close(conn)
	}
	return nil
}
