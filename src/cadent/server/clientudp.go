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
	UDP clients
*/

package cadent

import (
	"bytes"
	"cadent/server/broadcast"
	"cadent/server/dispatch"
	"cadent/server/repr"
	"cadent/server/splitter"
	"cadent/server/stats"
	"github.com/reusee/mmh3"
	logging "gopkg.in/op/go-logging.v1"
	"net"
)

// 1Mb default buffer size
const UDP_BUFFER_SIZE = 1048576

type UDPJob struct {
	Client   *UDPClient
	Splitem  splitter.SplitItem
	OutQueue chan splitter.SplitItem
	retry    int
}

func (j UDPJob) IncRetry() int {
	j.retry++
	return j.retry
}

func (j UDPJob) OnRetry() int {
	return j.retry
}

func (u UDPJob) DoWork() error {
	//u.Client.server.log.Debug("UDP: %s", u.Splitem)
	u.Client.server.ProcessSplitItem(u.Splitem, u.OutQueue)
	return nil
}

type UDPClient struct {
	server  *Server
	hashers *[]*ConstHasher

	Connection net.PacketConn
	LineCount  uint64
	BufferSize int

	out_queue    chan splitter.SplitItem
	done         chan Client
	input_queue  chan splitter.SplitItem
	worker_queue chan *OutputMessage
	shutdowner   *broadcast.Broadcaster

	line_queue chan string
	log        *logging.Logger
}

func NewUDPClient(server *Server, hashers *[]*ConstHasher, conn net.PacketConn, done chan Client) *UDPClient {

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
	client.shutdowner = broadcast.New(0)
	client.line_queue = make(chan string, server.Workers)

	return client

}

func (client *UDPClient) ShutDown() {
	client.shutdowner.Close()
}

func (client *UDPClient) SetBufferSize(size int) error {
	client.BufferSize = size
	return nil
	// no read buffer for "Packet Types" for the SO_REUSE goodness
	//return client.Connection.SetReadBuffer(size)
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

// we set up a static array of input and output channels
// each worker then processes one of these channels, and lines are fed via a mmh3 hash to the proper
// worker/channel set
func (client *UDPClient) createWorkers(workers int64, out_queue chan splitter.SplitItem) error {
	chs := make([]chan splitter.SplitItem, workers)
	for i := 0; i < int(workers); i++ {
		ch := make(chan splitter.SplitItem, 128)
		chs[i] = ch
		go client.consume(ch, out_queue)
	}
	go client.delegate(chs, uint32(workers))
	return nil
}

func (client *UDPClient) consume(inchan chan splitter.SplitItem, out_queue chan splitter.SplitItem) {
	shuts := client.shutdowner.Listen()
	defer shuts.Close()

	for {
		select {
		case splitem := <-inchan:
			client.server.ProcessSplitItem(splitem, out_queue)
		case <-shuts.Ch:
			return
		}
	}
}

// pick a channel to push the stat to
func (client *UDPClient) delegate(inchan []chan splitter.SplitItem, workers uint32) {
	shuts := client.shutdowner.Listen()
	defer shuts.Close()
	for {
		select {
		case splitem := <-client.input_queue:
			hash := mmh3.Hash32(splitem.Key()) % workers
			inchan[hash] <- splitem
		case <-shuts.Ch:
			return
		}
	}
}

func (client *UDPClient) procLines(lines []byte, job_queue chan dispatch.IJob, out_queue chan splitter.SplitItem) {
	for _, n_line := range bytes.Split(lines, repr.NEWLINE_SEPARATOR_BYTES) {
		if len(n_line) == 0 {
			continue
		}
		n_line = bytes.TrimSpace(n_line)
		if len(n_line) == 0 {
			continue
		}
		client.server.AllLinesCount.Up(1)
		splitem, err := client.server.SplitterProcessor.ProcessLine(n_line)
		//log.Notice("MOOO UDP line: %v MOOO", splitem.Fields())
		if err == nil {
			splitem.SetOrigin(splitter.UDP)
			splitem.SetOriginName(client.server.Name)
			//client.server.ProcessSplitItem(splitem, client.out_queue)
			stats.StatsdClient.Incr("incoming.udp.lines", 1)
			client.server.ValidLineCount.Up(1)
			client.input_queue <- splitem
			//client.delegateOne(splitem, uint32(client.server.Workers))

			//performs worse
			//job_queue <- UDPJob{Client: client, Splitem: splitem, OutQueue: out_queue}

		} else {
			client.server.InvalidLineCount.Up(1)
			stats.StatsdClient.Incr("incoming.udp.invalidlines", 1)
			log.Warning("Invalid Line: %s (%s)", err, n_line)
			continue
		}
	}
	return
}

func (client *UDPClient) run(out_queue chan splitter.SplitItem, close_client chan bool) {
	shuts := client.shutdowner.Listen()
	defer shuts.Close()

	for {
		select {
		case splitem := <-client.input_queue:
			client.server.ProcessSplitItem(splitem, out_queue)
		case <-shuts.Ch:
			return
		case <-close_client:
			return
		}
	}
}

func (client *UDPClient) getLines(job_queue chan dispatch.IJob, out_queue chan splitter.SplitItem) {
	shuts := client.shutdowner.Listen()
	defer shuts.Close()
	var buf = make([]byte, client.BufferSize)
	for {
		select {
		case <-shuts.Ch:
			return
		default:
			rlen, _, _ := client.Connection.ReadFrom(buf[:])
			client.server.BytesReadCount.Up(uint64(rlen))

			if rlen > 0 {
				client.procLines(buf[0:rlen], job_queue, out_queue)
			}
		}
	}
}

func (client UDPClient) handleRequest(out_queue chan splitter.SplitItem, close_client chan bool) {

	// UDP clients are basically "one" uber client (at least per socket)
	// DO NOT use work counts here, the UDP sockets are "multiplexed" using SO_CONNREUSE
	// so we have kernel level toggling between the various sockets so each "worker" is really
	// it's own little UDP listener land
	go client.run(out_queue, close_client)
	go client.run(client.out_queue, close_client) // bleed out non-socket inputs
	go client.getLines(nil, out_queue)

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
