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

package netpool

import (
	"errors"
	"net"
	"time"
)

var ErrMaxConn = errors.New("Maximum connections reached")

type NewNetPoolConnection func(net.Conn, NetpoolInterface) NetpoolConnInterface

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
	DestroyAll() error
}
