/*
	UDP clients
*/

package consthash

import (
	"./stats"
	"github.com/op/go-logging"
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

	out_queue    chan string
	done         chan Client
	input_queue  chan string
	worker_queue chan *SendOut

	log *logging.Logger
}

func NewUDPClient(server *Server, hashers *[]*ConstHasher, conn *net.UDPConn, done chan Client) *UDPClient {

	client := new(UDPClient)
	client.server = server
	client.hashers = hashers

	client.LineCount = 0
	client.Connection = conn
	client.SetBufferSize(UDP_BUFFER_SIZE)

	//to deref things
	client.worker_queue = server.WorkQueue
	client.input_queue = server.InputQueue
	client.out_queue = make(chan string, server.Workers)
	client.done = done
	client.log = server.log
	return client

}

func (client *UDPClient) SetBufferSize(size int) error {
	client.BufferSize = size
	return client.Connection.SetReadBuffer(size)
}

func (client UDPClient) Server() (server *Server) {
	return client.server
}

func (client UDPClient) Hashers() (hasher *[]*ConstHasher) {
	return client.hashers
}
func (client UDPClient) InputQueue() chan string {
	return client.input_queue
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

func (client *UDPClient) run() {
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
			client.server.RunRunner(key, n_line, client.out_queue)
		} else {
			client.log.Warning("Invalid Line: %s (%s)", err, n_line)
		}
		stats.StatsdClient.Incr("incoming.udp.lines", 1)
	}
}

func (client *UDPClient) getLines() {

	readStr := func(line string) {
		for _, n_line := range strings.Split(line, "\n") {
			if len(n_line) == 0 {
				continue
			}
			client.input_queue <- n_line
		}
	}

	var buf = make([]byte, client.BufferSize)
	for {
		rlen, _, _ := client.Connection.ReadFromUDP(buf[:])
		in_str := string(buf[0:rlen])
		if rlen > 0 {
			readStr(in_str)
		}
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
		message := <-client.out_queue
		if len(message) == 0 {
			break
		}
	}
	client.log.Notice("Closing UDP connection")
	//close it out
	client.Close()
}
