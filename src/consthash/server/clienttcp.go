/*
	TCP Client handling
	this also does UnixSocket connections too
*/

package consthash

import (
	"bufio"
	//"log"
	"consthash/server/splitter"
	"consthash/server/stats"
	"net"
	"reflect"
	"strings"
)

const TCP_BUFFER_SIZE = 1048576

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

	out_queue    chan splitter.SplitItem
	input_queue  chan splitter.SplitItem
	done         chan Client
	worker_queue chan *SendOut
	close        chan bool
}

func NewTCPClient(server *Server,
	hashers *[]*ConstHasher,
	conn net.Conn,
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
	client.worker_queue = server.WorkQueue
	client.input_queue = server.InputQueue

	client.out_queue = server.ProcessedQueue

	client.done = done
	client.close = make(chan bool)

	return client
}

func (client *TCPClient) ShutDown() {
	client.close <- true
}

func (client *TCPClient) connType() reflect.Type {
	return reflect.TypeOf(client.Connection)
}

func (client *TCPClient) SetBufferSize(size int) {
	client.BufferSize = size
	if client.connType() == reflect.TypeOf(new(net.TCPConn)) {
		client.Connection.(*net.TCPConn).SetReadBuffer(client.BufferSize)
	} else if client.connType() == reflect.TypeOf(new(net.UnixConn)) {
		client.Connection.(*net.UnixConn).SetReadBuffer(client.BufferSize)
	}
}

func (client *TCPClient) Server() (server *Server) {
	return client.server
}

func (client *TCPClient) Hashers() (server *[]*ConstHasher) {
	return client.hashers
}
func (client *TCPClient) WorkerQueue() chan *SendOut {
	return client.worker_queue
}
func (client *TCPClient) InputQueue() chan splitter.SplitItem {
	return client.input_queue
}

// close the 2 hooks, channel and connection
func (client *TCPClient) Close() {
	client.reader = nil
	if client.Connection != nil {
		client.Connection.Close()
	}
	client.Connection = nil
	client.server = nil
	client.hashers = nil
}

func (client *TCPClient) handleRequest(outqueue chan splitter.SplitItem) {
	//spin up the splitters

	buf := bufio.NewReaderSize(client.Connection, client.BufferSize)
	for {
		line, err := buf.ReadString('\n')

		if err != nil {
			break
		}
		if len(line) == 0 {
			continue
		}
		client.server.BytesReadCount.Up(uint64(len(line)))
		client.server.AllLinesCount.Up(1)
		splitem, err := client.server.SplitterProcessor.ProcessLine(strings.Trim(line, "\n\t "))
		if err == nil {
			//this will block once the queue is full
			splitem.SetOrigin(splitter.TCP)
			client.server.ValidLineCount.Up(1)
			client.input_queue <- splitem
			stats.StatsdClient.Incr("incoming.tcp.invalidlines", 1)
		} else {
			client.server.InvalidLineCount.Up(1)
			stats.StatsdClient.Incr("incoming.tcp.invalidlines", 1)
			log.Warning("Invalid Line: %s (%s)", err, line)
		}

	}

	buf = nil
	//close it and end the send routing
	outqueue <- splitter.BlankSplitterItem()
	client.done <- client

}

func (client *TCPClient) handleSend(outqueue chan splitter.SplitItem) {
	//just "bleed" it
	for {
		message := <-outqueue
		if message == nil || !message.IsValid() {
			return
		}
	}
}
