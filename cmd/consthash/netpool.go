package main

import (
	"errors"
	"net"
	"sync"
	"time"
)

const MaxConnections = 20
const ConnectionTimeout = time.Duration(5 * time.Second)
const RecycleTimeoutDuration = time.Duration(5 * time.Minute)

var ErrMaxConn = errors.New("Maximum connections reached")

type Netpool struct {
	mu             sync.Mutex
	name           string
	protocal       string
	conns          int
	MaxConnections int
	RecycleTimeout time.Duration
	free           chan NetpoolConn
}

// add a "global" timeout in order to pick up any new IPs from names of things
// in case DNS has changed and to release old connections that are unused
type NetpoolConn struct {
	conn    net.Conn
	started time.Time
}

func NewNetpool(protocal string, name string) *Netpool {
	pool := &Netpool{
		name:           name,
		protocal:       protocal,
		MaxConnections: MaxConnections,
		RecycleTimeout: RecycleTimeoutDuration,
	}
	pool.free = make(chan NetpoolConn, pool.MaxConnections)
	return pool
}

func (n *Netpool) NumFree() int {
	return len(n.free)

}

// reset and clear a connection
func (n *Netpool) ResetConn(net_conn NetpoolConn) error {

	if net_conn.conn != nil {
		net_conn.conn.Close()
	}
	net_conn.conn = nil

	conn, err := net.DialTimeout(n.protocal, n.name, ConnectionTimeout)
	if err != nil {
		return err
	}
	net_conn.conn = conn
	net_conn.started = time.Now()

	// put it back on the queue
	n.free <- net_conn

	return nil
}

func (n *Netpool) InitPool() error {

	n.mu.Lock()
	defer n.mu.Unlock()

	n.free = nil
	n.free = make(chan NetpoolConn, n.MaxConnections)

	//fill up the channels with our connections
	for i := 0; i < n.MaxConnections; i++ {
		conn, err := net.DialTimeout(n.protocal, n.name, ConnectionTimeout)
		if err != nil {
			return err
		}
		n.free <- NetpoolConn{conn: conn, started: time.Now()}
	}
	return nil
}

func (n *Netpool) Open() (conn NetpoolConn, err error) {
	// pop it off

	net_conn := <-n.free

	//recylce connections if we need to

	if time.Now().Sub(net_conn.started) > n.RecycleTimeout {
		if net_conn.conn != nil {
			net_conn.conn.Close()
		}
		net_conn.conn = nil

		conn, err := net.DialTimeout(n.protocal, n.name, ConnectionTimeout)
		if err != nil {
			return net_conn, err
		}
		net_conn.conn = conn
		net_conn.started = time.Now()
	}
	return net_conn, nil

}

//add it back to the queue
func (n *Netpool) Close(conn NetpoolConn) error {
	n.free <- conn
	return nil
}
