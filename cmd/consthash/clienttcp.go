/*
	TCP Client handling
*/

package main

import (
	"bufio"
	"net"
	"strings"
)

type TCPClient struct {
	server     *Server
	hashers    *[]*ConstHasher
	LineParser Runner

	Connection net.Conn
	LineCount  uint64

	//ins and outs
	writer *bufio.Writer
	reader *bufio.Reader

	channel      chan string
	done         chan Client
	worker_queue chan *SendOut
}

func NewTCPClient(server *Server, hashers *[]*ConstHasher, conn net.Conn, worker_queue chan *SendOut, done chan Client) *TCPClient {

	client := new(TCPClient)
	client.server = server
	client.hashers = hashers

	//client.writer = bufio.NewWriter(conn)
	client.reader = bufio.NewReader(conn)
	client.LineCount = 0
	client.Connection = conn

	client.channel = make(chan string)
	client.worker_queue = worker_queue
	client.done = done
	return client
}

func (client TCPClient) Server() (server *Server) {
	return client.server
}

func (client TCPClient) Hashers() (server *[]*ConstHasher) {
	return client.hashers
}
func (client TCPClient) WorkerQueue() chan *SendOut {
	return client.worker_queue
}

// close the 2 hooks, channel and connection
func (client TCPClient) Close() {
	close(client.channel)
	client.Connection.Close()
	client.server = nil
	client.hashers = nil

}

func (client TCPClient) run(line string) {
	client.server.AllLinesCount.Up(1)
	key, line, err := client.server.LineProcessor.ProcessLine(strings.Trim(line, "\n\t "))
	if err == nil {
		go client.server.RunRunner(key, line, client.channel)
	}
	StatsdClient.Incr("incoming.tcp.lines", 1)
}

func (client TCPClient) handleRequest() {

	for {

		line, err := client.reader.ReadString('\n')
		if err != nil || len(line) == 0 {
			break
		}

		go client.run(line)

	}
	//close it
	//client.done <- client

}

func (client TCPClient) handleSend() {

	for {
		message := <-client.channel
		if len(message) == 0 {
			break
		}
		//log.Print(message)

		//client.writer.WriteString(message)
		//client.writer.Flush()
		//clear buffer
		//client.reader.Reset(client.Connection)
	}
	//close it out
	client.done <- client
}
