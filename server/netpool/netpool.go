package netpool

import (
	"log"
	"net"
	"sync"
	"time"
)

const MaxConnections = 20
const ConnectionTimeout = time.Duration(5 * time.Second)
const RecycleTimeoutDuration = time.Duration(5 * time.Minute)

type Netpool struct {
	mu                sync.Mutex
	name              string
	protocal          string
	conns             int
	MaxConnections    int
	RecycleTimeout    time.Duration
	newConnectionFunc NewNetPoolConnection //this is to make "New" connections only
	free              chan NetpoolConnInterface
}

// add a "global" timeout in order to pick up any new IPs from names of things
// in case DNS has changed and to release old connections that are unused
type NetpoolConn struct {
	conn    net.Conn
	started time.Time
	idx     int
}

func NewNetPoolConn(conn net.Conn, pool NetpoolInterface) NetpoolConnInterface {
	return &NetpoolConn{
		conn:    conn,
		started: time.Now(),
	}
}

func (n *NetpoolConn) Conn() net.Conn        { return n.conn }
func (n *NetpoolConn) SetConn(conn net.Conn) { n.conn = conn }

func (n *NetpoolConn) Started() time.Time     { return n.started }
func (n *NetpoolConn) SetStarted(t time.Time) { n.started = t }

func (n *NetpoolConn) Index() int     { return n.idx }
func (n *NetpoolConn) SetIndex(i int) { n.idx = i }

func (n *NetpoolConn) SetWriteDeadline(t time.Time) error {
	return n.conn.SetWriteDeadline(t)
}

// null function
func (n *NetpoolConn) Flush() (int, error) {
	return 0, nil
}

func (n *NetpoolConn) Write(b []byte) (int, error) {
	return n.conn.Write(b)
}

///***** POOLER ****///

func NewNetpool(protocal string, name string) *Netpool {
	pool := &Netpool{
		name:              name,
		protocal:          protocal,
		MaxConnections:    MaxConnections,
		RecycleTimeout:    RecycleTimeoutDuration,
		newConnectionFunc: NewNetPoolConn,
	}
	pool.free = make(chan NetpoolConnInterface, pool.MaxConnections)
	return pool
}

func (n *Netpool) GetMaxConnections() int {
	return n.MaxConnections
}

func (n *Netpool) SetMaxConnections(maxconn int) {
	n.MaxConnections = maxconn
}

func (n *Netpool) NumFree() int {
	return len(n.free)
}

// reset and clear a connection
func (n *Netpool) ResetConn(net_conn NetpoolConnInterface) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if net_conn.Conn() != nil {
		net_conn.Flush()
		goterr := net_conn.Conn().Close()
		if goterr != nil {
			log.Println("[NetPool:ResetConn] Connection CLOSE error: ", goterr)
		}
	}
	net_conn.SetConn(nil)

	conn, err := net.DialTimeout(n.protocal, n.name, ConnectionTimeout)
	if err != nil {
		log.Println("[NetPool:ResetConn] Connection open error: ", err)
		return err
	}
	net_conn.SetConn(conn)
	net_conn.SetStarted(time.Now())

	// put it back on the queue
	log.Println("[NetPool:ResetConn] Reset Connection %s://%s ", n.protocal, n.name)
	// NONONO n.free <- net_conn let "Close" do this only

	return nil
}

func (n *Netpool) InitPoolWith(obj NetpoolInterface) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.free = nil
	n.free = make(chan NetpoolConnInterface, n.MaxConnections)

	//fill up the channels with our connections
	for i := 0; i < n.MaxConnections; i++ {
		conn, err := net.DialTimeout(n.protocal, n.name, ConnectionTimeout)
		if(n.protocal == "tcp"){
			conn.(*net.TCPConn).SetNoDelay(true)
		}
		if err != nil {
			log.Println("[NetPool:InitPool] Connection open error:  ", n.protocal, n.name, err)
			return err
		}
		netcon := n.newConnectionFunc(conn, obj)
		netcon.SetIndex(i)
		n.free <- netcon
	}
	return nil
}

func (n *Netpool) InitPool() error {
	return n.InitPoolWith(n)

}

func (n *Netpool) Open() (conn NetpoolConnInterface, err error) {
	// pop it off

	net_conn := <-n.free
	//recycle connections if we need to or reconnect if we need to
	if net_conn.Conn() == nil || time.Now().Sub(net_conn.Started()) > n.RecycleTimeout {
		if net_conn.Conn() != nil {
			net_conn.Flush()
			goterr := net_conn.Conn().Close()
			if goterr != nil {
				log.Printf("[NetPool:Open] Output Connection CLOSE error: %s", goterr)
			}
		}
		net_conn.SetConn(nil)

		conn, err := net.DialTimeout(n.protocal, n.name, ConnectionTimeout)
		if err != nil {
			log.Printf("[NetPool:Open] Output Connection open error: %s", err)
			// we CANNOT return here we need the connections in the queue even if they are "dead"
			// they will get re-tried
			//return net_conn, err
		}else {
			log.Printf("[NetPool:Open] New Output Connection opened to %s", conn.RemoteAddr())
		}
		net_conn.SetConn(conn)
		net_conn.SetStarted(time.Now())
	}
	return net_conn, err
}

//add it back to the queue
func (n *Netpool) Close(conn NetpoolConnInterface) error {
	n.free <- conn

	return nil
}

//nuke all the connections
func (n *Netpool) DestroyAll() error {

	for i := 0; i < len(n.free); i++ {
		con := <-n.free
		con.Conn().Close()
	}
	return nil
}
