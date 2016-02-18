/*
	UDP clients
*/

package consthash

import (
	"consthash/server/splitter"
	"consthash/server/stats"
	logging "github.com/op/go-logging"
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

	out_queue    chan splitter.SplitItem
	done         chan Client
	input_queue  chan splitter.SplitItem
	worker_queue chan *OutputMessage
	close        chan bool

	line_queue chan string
	log        *logging.Logger
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
	client.out_queue = server.ProcessedQueue
	client.done = done
	client.log = server.log
	client.close = make(chan bool)
	client.line_queue = make(chan string, server.Workers)
	return client

}

func (client *UDPClient) ShutDown() {
	client.close <- true
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
func (client UDPClient) InputQueue() chan splitter.SplitItem {
	return client.input_queue
}
func (client UDPClient) WorkerQueue() chan *OutputMessage {
	return client.worker_queue
}
func (client UDPClient) Close() {
	client.server = nil
	client.hashers = nil
	if client.Connection != nil {
		client.Connection.Close()
		client.Connection = nil
	}
}

func (client *UDPClient) procLines(line string) {
	for _, n_line := range strings.Split(line, "\n") {
		if len(n_line) == 0 {
			continue
		}
		n_line = strings.Trim(n_line, "\r\n\t ")
		if len(n_line) == 0 {
			continue
		}
		client.server.AllLinesCount.Up(1)
		splitem, err := client.server.SplitterProcessor.ProcessLine(n_line)
		//log.Notice("MOOO UDP line: %v MOOO", splitem.Fields())
		if err == nil {
			splitem.SetOrigin(splitter.UDP)
			//client.server.ProcessSplitItem(splitem, client.out_queue)
			stats.StatsdClient.Incr("incoming.udp.lines", 1)
			client.server.ValidLineCount.Up(1)
			client.input_queue <- splitem
		} else {
			client.server.InvalidLineCount.Up(1)
			stats.StatsdClient.Incr("incoming.udp.invalidlines", 1)
			log.Warning("Invalid Line: %s (%s)", err, n_line)
			continue
		}
	}
	return
}
func (client *UDPClient) run(out_queue chan splitter.SplitItem) {
	for {
		select {
		case splitem := <-client.input_queue:
			client.server.ProcessSplitItem(splitem, out_queue)
		case <-client.close:
			break
		}
	}
	return
}

func (client *UDPClient) getLines() {

	var buf = make([]byte, client.BufferSize)
	for {
		rlen, _, _ := client.Connection.ReadFromUDP(buf[:])
		client.server.BytesReadCount.Up(uint64(rlen))

		in_str := string(buf[0:rlen])
		if rlen > 0 {
			client.procLines(in_str)
		}
	}
	return
}

func (client UDPClient) handleRequest(out_queue chan splitter.SplitItem) {
	for w := int64(1); w <= client.server.Workers; w++ {
		go client.run(out_queue)
		go client.run(client.out_queue) // bleed out non-socket inputs
	}

	go client.getLines()
	return
}

func (client UDPClient) handleSend(out_queue chan splitter.SplitItem) {

	for {
		message := <-out_queue
		if message == nil || !message.IsValid() {
			break
		}
	}
	client.log.Notice("Closing UDP connection")
	//close it out
	client.Close()
	return
}
