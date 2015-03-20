/*
	UDP clients
*/

package main

import (
	"log"
	"net"
	"strings"
)

type UDPClient struct {
	server  *Server
	hashers *[]*ConstHasher

	Connection   *net.UDPConn
	LineCount    uint64
	MaxLineCount uint64
	CycleCount   uint64

	channel      chan string
	done         chan Client
	worker_queue chan *SendOut
}

func NewUDPClient(server *Server, hashers *[]*ConstHasher, conn *net.UDPConn, worker_queue chan *SendOut, done chan Client) *UDPClient {

	client := new(UDPClient)
	client.server = server
	client.hashers = hashers

	client.LineCount = 0
	client.Connection = conn

	// we "parrael" this many processes then block until we are done
	client.MaxLineCount = 1024
	client.CycleCount = 0

	client.channel = make(chan string)
	client.worker_queue = worker_queue
	client.done = done
	return client

}

func (client UDPClient) Server() (server *Server) {
	return client.server
}

func (client UDPClient) Hashers() (hasher *[]*ConstHasher) {
	return client.hashers
}
func (client UDPClient) WorkerQueue() chan *SendOut {
	return client.worker_queue
}
func (client UDPClient) Close() {
	client.server = nil
	client.hashers = nil
	if client.Connection != nil {
		client.Connection.Close()
	}
}

func (client UDPClient) run(line string) {
	client.server.AllLinesCount.Up(1)

	key, line, err := client.server.LineProcessor.ProcessLine(strings.Trim(line, "\n\t "))
	if err == nil {
		go client.server.RunRunner(key, line, client.channel)
	}
	StatsdClient.Incr("incoming.udp.lines", 1)
}

func (client UDPClient) handleRequest() {
	for {
		var buf [1024]byte
		rlen, _, _ := client.Connection.ReadFromUDP(buf[:])
		in_str := string(buf[0:rlen])
		for _, line := range strings.Split(in_str, "\n") {
			if len(line) == 0 {
				continue
			}
			go client.run(line)
		}
	}

}

func (client UDPClient) handleSend() {

	for {
		message := <-client.channel
		if len(message) == 0 {
			break
		}
	}
	log.Println("Close")
	//close it out
	client.Close()
}
