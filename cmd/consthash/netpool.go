package main

import (
	"errors"
	"net"
	"sync"
	"time"
)

const MaxConnections = 20
const ConnectionTimeout = time.Duration(5 * time.Second)

var ErrMaxConn = errors.New("Maximum connections reached")

type Netpool struct {
	mu             sync.Mutex
	name           string
	protocal       string
	conns          int
	MaxConnections int
	free           []net.Conn
}

func NewNetpool(protocal string, name string) *Netpool {
	return &Netpool{
		name:           name,
		protocal:       protocal,
		MaxConnections: MaxConnections,
	}
}

// reset all the connections
func (n *Netpool) Reset() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, conn := range n.free {
		conn.Close()
	}
	n.free = make([]net.Conn, 0)
	n.conns = 0
}

func (n *Netpool) RemoveConn(in_conn net.Conn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	new_list := make([]net.Conn, 0)
	n.conns = 0
	for _, conn := range n.free {
		if conn == in_conn {
			conn.Close()
		} else {
			n.conns += 1
			n.free = append(n.free, conn)

		}
	}
	n.free = new_list

}

func (n *Netpool) Open() (conn net.Conn, err error) {
	if n.conns >= n.MaxConnections && len(n.free) == 0 {
		return nil, ErrMaxConn
	}
	n.mu.Lock()
	if len(n.free) > 0 {
		// return the first free connection in the pool
		conn = n.free[0]
		n.free = n.free[1:]
		//reset if we can
		if conn == nil {
			conn, err = net.DialTimeout(n.protocal, n.name, ConnectionTimeout)
		}
	} else {
		conn, err = net.DialTimeout(n.protocal, n.name, ConnectionTimeout)
		n.conns += 1
	}
	n.mu.Unlock()
	return conn, err
}

func (n *Netpool) Close(conn net.Conn) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.free = append(n.free, conn)
	return nil
}
