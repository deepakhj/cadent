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
	TCP Client handling
	this also does UnixSocket connections too
*/

package cadent

import (
	"bufio"
	//"log"
	"bytes"
	"cadent/server/dispatch"
	"cadent/server/repr"
	"cadent/server/splitter"
	"cadent/server/stats"
	"fmt"
	"net"
	"reflect"
	"time"
)

// TCP_BUFFER_SIZE Size in bytes of the TCP read buffer
const TCP_BUFFER_SIZE = 1048576

// TCP_READ_TIMEOUT TCP read timeout
const TCP_READ_TIMEOUT = 5 * time.Second // second

/************************** TCP Dispatcher Job *******************************/

type tcpJob struct {
	Client  *TCPClient
	Splitem splitter.SplitItem
	retry   int
}

func (t tcpJob) IncRetry() int {
	t.retry++
	return t.retry
}

func (t tcpJob) OnRetry() int {
	return t.retry
}

func (t tcpJob) DoWork() {
	if t.Splitem == nil {
		return
	}
	llen := (int64)(len(t.Splitem.Line()))
	t.Client.server.AddToCurrentTotalBufferSize(llen)
	if t.Client.server.NeedBackPressure() {
		t.Client.server.log.Warning(
			"Error::Max Queue or buffer reached dropping connection (Buffer %v, queue len: %v)",
			t.Client.server.CurrentReadBufferRam.Get(),
			len(t.Client.server.WorkQueue))
		t.Client.server.BackPressure()
	}
	t.Client.server.ProcessSplitItem(t.Splitem, t.Client.server.ProcessedQueue)
	t.Client.server.AddToCurrentTotalBufferSize(-llen)
}

/*
TCPClient Should be one per connection (unlike udp/http which is one for all servers, so handling is
treated a bit differently here
*/
type TCPClient struct {
	server     *Server
	hashers    *[]*ConstHasher
	LineParser splitter.Splitter

	Connection net.Conn
	LineCount  uint64

	BufferSize int

	//ins and outs
	writer *bufio.Writer
	reader *bufio.Reader

	outQueue      chan splitter.SplitItem
	inputQueue    chan splitter.SplitItem
	dispatchQueue chan dispatch.IJob
	done          chan Client
	workerQueue   chan *OutputMessage
	close         chan bool
	shutitdown    bool
}

// NewTCPClient create a new TCP client
func NewTCPClient(server *Server,
	hashers *[]*ConstHasher,
	conn net.Conn,
	dispatchQueue chan dispatch.IJob,
	done chan Client,
) *TCPClient {

	client := new(TCPClient)
	client.server = server
	client.hashers = hashers

	//client.writer = bufio.NewWriter(conn)
	//client.reader = bufio.NewReaderSize(conn, TCP_BUFFER_SIZE)
	client.LineCount = 0
	client.Connection = conn
	client.SetBufferSize(TCP_BUFFER_SIZE)

	//to deref things
	client.workerQueue = server.WorkQueue
	client.inputQueue = server.InputQueue
	client.dispatchQueue = dispatchQueue

	client.outQueue = server.ProcessedQueue

	client.done = done
	client.shutitdown = false
	client.close = make(chan bool)

	return client
}

// ShutDown initiate a client shudown
func (client *TCPClient) ShutDown() {
	client.shutitdown = true
	client.close <- true
}

func (client *TCPClient) connType() reflect.Type {
	return reflect.TypeOf(client.Connection)
}

// SetBufferSize read buffer for TCP conections
func (client *TCPClient) SetBufferSize(size int) {
	client.BufferSize = size
	if client.connType() == reflect.TypeOf(new(net.TCPConn)) {
		client.Connection.(*net.TCPConn).SetReadBuffer(client.BufferSize)
	} else if client.connType() == reflect.TypeOf(new(net.UnixConn)) {
		client.Connection.(*net.UnixConn).SetReadBuffer(client.BufferSize)
	}
}

// Server of this client
func (client *TCPClient) Server() (server *Server) {
	return client.server
}

// Hashers of this client
func (client *TCPClient) Hashers() (server *[]*ConstHasher) {
	return client.hashers
}

// WorkerQueue chan
func (client *TCPClient) WorkerQueue() chan *OutputMessage {
	return client.workerQueue
}

// InputQueue chan
func (client *TCPClient) InputQueue() chan splitter.SplitItem {
	return client.inputQueue
}

// Close the 2 hooks, channel and connection
func (client *TCPClient) Close() {
	defer stats.StatsdClient.Incr(fmt.Sprintf("worker.%s.tcp.connection.close", client.server.Name), 1)
	defer log.Debug("TCP client: Closing conn %v", client.Connection.RemoteAddr())
	//client.close <- true
	client.reader = nil
	if client.Connection != nil {
		client.Connection.Close()
	}
	client.Connection = nil
	client.server = nil
	client.hashers = nil
}

func (client *TCPClient) handleRequest(outqueue chan splitter.SplitItem, closeClient chan bool) {
	//spin up the splitters

	//client.Connection.SetReadDeadline(time.Now().Add(TCP_READ_TIMEOUT))
	buf := bufio.NewReaderSize(client.Connection, client.BufferSize)
	for {
		select {
		case <-client.close:
			break
		case <-closeClient:
			client.Connection.Close()
			return
		default:
		}
		line, err := buf.ReadBytes(repr.NEWLINE_SEPARATOR_BYTE)

		if err != nil {
			break
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		client.server.BytesReadCount.Up(uint64(len(line)))
		client.server.AllLinesCount.Up(1)
		splitem, err := client.server.SplitterProcessor.ProcessLine(line)
		if err == nil {
			if client.shutitdown {
				break
			}
			//this will block once the queue is full
			splitem.SetOrigin(splitter.TCP)
			splitem.SetOriginName(client.server.Name)
			client.server.ValidLineCount.Up(1)
			client.inputQueue <- splitem
			//client.dispatchQueue <- TCPJob{Client: client, Splitem: splitem}
			stats.StatsdClient.Incr("incoming.tcp.lines", 1)
		} else {
			client.server.InvalidLineCount.Up(1)
			stats.StatsdClient.Incr("incoming.tcp.invalidlines", 1)
			log.Warning("Invalid Line: %s (%s)", err, line)
		}

	}

	if client.done != nil {
		client.done <- client
	}

	return
}

func (client *TCPClient) handleSend(outqueue chan splitter.SplitItem) {
	//just "bleed" it
	// NOT NEED for this really.  the client.done <- will take care of the
	// close and we don't use the outqueue for anything, so we waist channel locking cycles for no reason

	for {
		select {
		case message := <-outqueue:
			if message == nil || !message.IsValid() {
				return
			}
		case <-client.close:
			return
		}
	}
}
