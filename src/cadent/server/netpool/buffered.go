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

/*
Here we have a NetPooler but that buffers writes before sending things in specifc chunks

*/

package netpool

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const DEFAULT_BUFFER_SIZE = 512
const DEFAULT_FORCE_FLUSH = time.Second

type BufferedNetpool struct {
	pool           *Netpool
	BufferSize     int
	ForceFlushTime time.Duration
	didclose       int32

	//lock to grab all the active cons and flush them
	flushLock sync.Mutex
	closeLock sync.Mutex
}

type BufferedNetpoolConn struct {
	conn        net.Conn
	started     time.Time
	idx         int
	writebuffer []byte
	buffersize  int

	writeLock sync.Mutex
	connLock  sync.Mutex
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
		time.Sleep(DEFAULT_FORCE_FLUSH)
	}
}

func (n *BufferedNetpoolConn) Conn() net.Conn {
	//n.connLock.Lock()
	//defer n.connLock.Unlock()
	return n.conn
}
func (n *BufferedNetpoolConn) SetConn(conn net.Conn) {
	n.connLock.Lock()
	defer n.connLock.Unlock()
	n.conn = conn
}

func (n *BufferedNetpoolConn) Started() time.Time     { return n.started }
func (n *BufferedNetpoolConn) SetStarted(t time.Time) { n.started = t }

func (n *BufferedNetpoolConn) Index() int     { return n.idx }
func (n *BufferedNetpoolConn) SetIndex(i int) { n.idx = i }

func (n *BufferedNetpoolConn) SetWriteDeadline(t time.Time) error {
	return n.Conn().SetWriteDeadline(t)
}

func (n *BufferedNetpoolConn) Write(b []byte) (wrote int, err error) {

	n.writeLock.Lock()
	defer n.writeLock.Unlock()
	c := n.Conn()
	if (len(n.writebuffer)+len(b)) > n.buffersize && c != nil {
		wrote, err = c.Write(n.writebuffer)
		//log.Debug("BUF WRITE %v", string(n.writebuffer))
		if err != nil {
			c.Close() // Open will re-open it
			n.SetConn(nil)
			log.Warning("Error writing buffer: %s", err)
			return 0, err
		}
		//log.Debug("BUF WRITE %d/%d wrote: %d", len(n.writebuffer), n.buffersize, wrote)
		n.writebuffer = []byte("")
	}
	n.writebuffer = append(n.writebuffer, b...)

	return wrote, err
}

func (n *BufferedNetpoolConn) Flush() (wrote int, err error) {
	n.writeLock.Lock()
	defer n.writeLock.Unlock()
	//StatsdClient.Incr("success.packets.flushes", 1)
	c := n.Conn()
	//log.Debug("FLUSH BUF WRITE %d/%d wrote: %d", len(n.writebuffer), n.buffersize, wrote)
	if len(n.writebuffer) > 0 && c != nil {
		wrote, err = c.Write(n.writebuffer)
		//log.Debug("FLUSH BUF WRITE %v", string(n.writebuffer))
		//log.Debug("BUF WRITE %d/%d wrote: %d", len(n.writebuffer), n.buffersize, wrote)
		if err != nil {
			c.Close() // Open will re-open it
			n.SetConn(nil)
			log.Warning("Error writing flush: %s", err)
			return 0, err
		}
		n.writebuffer = []byte("")
	}
	return wrote, err
}

/*** Pooler ***/

func NewBufferedNetpool(protocol string, name string, buffersize int) *BufferedNetpool {
	pool := NewNetpool(protocol, name)
	//override
	pool.newConnectionFunc = NewBufferedNetpoolConn

	bpool := &BufferedNetpool{
		pool:           pool,
		BufferSize:     DEFAULT_BUFFER_SIZE,
		ForceFlushTime: DEFAULT_FORCE_FLUSH,
		didclose:       0,
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

	go func(np *BufferedNetpool) {
		s := <-sigc
		/*
			np.closeLock.Lock()
			defer np.closeLock.Unlock()
			log.Warning("Caught %s: Flushing Buffers before quit ", s)
			atomic.StoreInt32(&np.didclose, 1)
			defer close(np.pool.free)
			for i := 0; i < len(np.pool.free); i++ {
				con := <-np.pool.free
				con.Flush()
			}
		*/

		signal.Stop(sigc)
		close(sigc)
		// re-raise it
		process, _ := os.FindProcess(os.Getpid())
		process.Signal(s)
		return
	}(n)
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
	if atomic.LoadInt32(&n.didclose) == 0 {
		return n.pool.Open()
	}
	return nil, fmt.Errorf("Closing all buffers, cannot open")
}

//add it back to the queue
func (n *BufferedNetpool) Close(conn NetpoolConnInterface) error {
	//n.closeLock.Lock()
	//defer n.closeLock.Unlock()
	if atomic.LoadInt32(&n.didclose) == 0 {
		return n.pool.Close(conn)
	}
	return nil
}

//nuke all the connections
func (n *BufferedNetpool) DestroyAll() error {
	for i := 0; i < len(n.pool.free); i++ {
		con := <-n.pool.free
		con.Flush()
	}
	n.pool.DestroyAll()
	return nil
}
