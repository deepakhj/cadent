/*
	TCP Client handling
	this also does UnixSocket connections too
*/

package consthash

import (
	"bufio"
	//"log"
	"./splitter"
	"net"
	"reflect"
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

	out_queue    chan string
	input_queue  chan string
	done         chan Client
	worker_queue chan *SendOut
}

func NewTCPClient(server *Server,
	hashers *[]*ConstHasher,
	conn net.Conn,
	done chan Client,
	out_queue chan string) *TCPClient {

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

	client.out_queue = out_queue

	client.done = done

	return client
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
func (client *TCPClient) InputQueue() chan string {
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

func (client *TCPClient) handleRequest() {
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

		//this will block once the queue is full
		client.input_queue <- line
	}

	buf = nil
	//close it and end the send routing
	client.out_queue <- ""
	client.done <- client

}

func (client *TCPClient) handleSend() {
	//just "bleed" it
	for {
		if len(<-client.out_queue) == 0 {
			return
		}
	}
}
