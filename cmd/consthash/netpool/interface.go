package netpool

import (
	"errors"
	"net"
	"time"
)

var ErrMaxConn = errors.New("Maximum connections reached")

type NewNetPoolConnection func(net.Conn, NetpoolInterface) NetpoolConnInterface

/** a "connection" interface **/

type NetpoolConnInterface interface {
	Conn() net.Conn
	SetConn(net.Conn)
	Started() time.Time
	SetStarted(time.Time)
	Index() int
	SetIndex(int)
	SetWriteDeadline(time.Time) error
	Write([]byte) (int, error)
	Flush() (int, error)
}

///***** POOLER ****///

type NetpoolInterface interface {
	GetMaxConnections() int
	SetMaxConnections(int)
	NumFree() int
	ResetConn(net_conn NetpoolConnInterface) error
	InitPoolWith(obj NetpoolInterface) error
	InitPool() error
	Open() (conn NetpoolConnInterface, err error)
	Close(conn NetpoolConnInterface) error
}
