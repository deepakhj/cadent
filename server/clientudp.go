/*
	UDP clients
*/

package consthash

import (
	"log"
	"net"
	"strings"
)

// 1Mb default buffer size
const UDP_BUFFER_SIZE = 1048576

type UDPClient struct {
	server  *Server
	hashers *[]*ConstHasher

	Connection *net.UDPConn
	LineCount  uint64
	BufferSize int

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
	client.BufferSize = UDP_BUFFER_SIZE
	client.Connection.SetReadBuffer(UDP_BUFFER_SIZE)

	client.worker_queue = worker_queue
	client.input_queue = make(chan string, server.Workers)
	client.channel = make(chan string, server.Workers)
	client.done = done
	return client

}

func (client UDPClient) SetBufferSize(size int) {
	client.BufferSize = size
	client.Connection.SetReadBuffer(client.BufferSize)
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

		n_line := strings.Trim(line, "\n\t ")
		if len(n_line) == 0 {
			continue
		}
		client.server.AllLinesCount.Up(1)
		key, _, err := client.server.LineProcessor.ProcessLine(n_line)
		if err == nil {
			client.server.RunRunner(key, n_line, client.channel)
		} else {
			log.Printf("Invalid Line: %s (%s)", err, n_line)
		}
		StatsdClient.Incr("incoming.udp.lines", 1)
	}
}

func (client UDPClient) getLines() {

	readStr := func(line string) {
		for _, n_line := range strings.Split(line, "\n") {
			if len(n_line) == 0 {
				continue
			}
			//log.Println("REQ: ", len(client.input_queue), line)
			client.input_queue <- n_line
		}
	}
	bufsize := UDP_BUFFER_SIZE
	if client.BufferSize > 0 {
		bufsize = client.BufferSize
	}
	var buf = make([]byte, bufsize)
	for {
		rlen, _, _ := client.Connection.ReadFromUDP(buf[:])
		in_str := string(buf[0:rlen])
		readStr(in_str)
	}
}

func (client UDPClient) handleRequest() {
	for w := int64(1); w <= client.server.Workers; w++ {
		go client.run()
	}

	go client.getLines()
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
