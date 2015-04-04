/*
	TCP Client handling
*/

package main

import (
	"bufio"
	//"log"
	"net"
)

// not really used in scanner mode
const TCP_BUFFER_SIZE = 4096

type TCPClient struct {
	server     *Server
	hashers    *[]*ConstHasher
	LineParser Runner

	Connection *net.TCPConn
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
	conn *net.TCPConn,
	worker_queue chan *SendOut,
	done chan Client,
	input_queue chan string,
	out_queue chan string) *TCPClient {

	client := new(TCPClient)
	client.server = server
	client.hashers = hashers

	//client.writer = bufio.NewWriter(conn)
	//client.reader = bufio.NewReaderSize(conn, TCP_BUFFER_SIZE)
	client.LineCount = 0
	client.Connection = conn
	client.BufferSize = TCP_BUFFER_SIZE

	client.out_queue = out_queue
	client.worker_queue = worker_queue
	client.input_queue = input_queue
	client.done = done
	return client
}

//no need for TCP as we use a bufio reader
func (client *TCPClient) SetBufferSize(size int) {
	client.BufferSize = size
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
	//spin up the runners
	buf := bufio.NewReaderSize(client.Connection, client.BufferSize)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			break
		}
		if len(line) == 0 {
			continue
		}
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
