/*
	UDP clients
*/

package main

import (
	"log"
	"net"
	"strings"
)

const UDP_BUFFER_SIZE = 26214400

type UDPClient struct {
	server  *Server
	hashers *[]*ConstHasher

	Connection   *net.UDPConn
	LineCount    uint64
	MaxLineCount uint64
	CycleCount   uint64

	channel      chan string
	done         chan Client
	input_queue  chan string
	worker_queue chan *SendOut
}

func NewUDPClient(server *Server, hashers *[]*ConstHasher, conn *net.UDPConn, worker_queue chan *SendOut, done chan Client) *UDPClient {

	client := new(UDPClient)
	client.server = server
	client.hashers = hashers

	client.LineCount = 0
	client.Connection = conn
	client.Connection.SetReadBuffer(UDP_BUFFER_SIZE)

	// we "parrael" this many processes then block until we are done
	client.MaxLineCount = 1024
	client.CycleCount = 0

	client.worker_queue = worker_queue
	client.input_queue = make(chan string, server.Workers)
	client.channel = make(chan string, server.Workers)
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

func (client UDPClient) run() {
	for line := range client.input_queue {
		if line == "" {
			continue
		}
		//log.Println(len(client.input_queue), line)
		client.server.AllLinesCount.Up(1)
		key, line, err := client.server.LineProcessor.ProcessLine(strings.Trim(line, "\n\t "))
		if err == nil {
			client.server.RunRunner(key, line, client.channel)
		}
		StatsdClient.Incr("incoming.udp.lines", 1)
	}
}

func (client UDPClient) getLines(idx int64) {

	readStr := func(line string) {
		for _, line := range strings.Split(line, "\n") {
			if len(line) == 0 {
				continue
			}
			//log.Println("REQ: ", len(client.input_queue), line)
			client.input_queue <- line
		}
	}

	var buf [UDP_BUFFER_SIZE]byte
	for {
		rlen, _, _ := client.Connection.ReadFromUDP(buf[:])
		in_str := string(buf[0:rlen])
		go readStr(in_str)
	}
}

func (client UDPClient) handleRequest() {
	for w := int64(1); w <= client.server.Workers; w++ {
		go client.run()
	}

	//for w := int64(1); w <= client.server.Workers; w++ {
	go client.getLines(1)
	//}
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
