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

// UDP_BUFFER_SIZE 1Mb default buffer size
const UDP_BUFFER_SIZE = 1048576

type udpJob struct {
	Client   *UDPClient
	Splitem  splitter.SplitItem
	OutQueue chan splitter.SplitItem
	retry    int
}

func (u udpJob) IncRetry() int {
	u.retry++
	return u.retry
}

func (u udpJob) OnRetry() int {
	return u.retry
}

func (u udpJob) DoWork() error {
	//u.Client.server.log.Debug("UDP: %s", u.Splitem)
	u.Client.server.ProcessSplitItem(u.Splitem, u.OutQueue)
	return nil
}

// UDPClient basic server UDP input client
type UDPClient struct {
	server  *Server
	hashers *[]*ConstHasher

	Connection net.PacketConn
	LineCount  uint64
	BufferSize int

	outQueue    chan splitter.SplitItem
	done        chan Client
	inputQueue  chan splitter.SplitItem
	workerQueue chan *OutputMessage
	shutdowner  *broadcast.Broadcaster

	lineQueue chan string
	log       *logging.Logger
}

// NewUDPClient fire up a new client
func NewUDPClient(server *Server, hashers *[]*ConstHasher, conn net.PacketConn, done chan Client) *UDPClient {

	client := new(UDPClient)
	client.server = server
	client.hashers = hashers

	client.LineCount = 0
	client.Connection = conn
	client.SetBufferSize(UDP_BUFFER_SIZE)

	//to deref things
	client.workerQueue = server.WorkQueue
	client.inputQueue = server.InputQueue
	client.outQueue = server.ProcessedQueue
	client.done = done
	client.log = server.log
	client.shutdowner = broadcast.New(0)
	client.lineQueue = make(chan string, server.Workers)

	return client

}

// ShutDown alias for Close
func (client *UDPClient) ShutDown() {
	client.shutdowner.Close()
}

// SetBufferSize is a noop for SO_REUSE connection types
func (client *UDPClient) SetBufferSize(size int) error {
	client.BufferSize = size
	return nil
	// no read buffer for "Packet Types" for the SO_REUSE goodness
	//return client.Connection.SetReadBuffer(size)
}

// Server client attached server
func (client UDPClient) Server() (server *Server) {
	return client.server
}

// Hashers client hashers
func (client UDPClient) Hashers() (hasher *[]*ConstHasher) {
	return client.hashers
}

// InputQueue chan
func (client UDPClient) InputQueue() chan splitter.SplitItem {
	return client.inputQueue
}

// WorkerQueue chan
func (client UDPClient) WorkerQueue() chan *OutputMessage {
	return client.workerQueue
}

// Close initiate a close of the client
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
func (client *UDPClient) createWorkers(workers int64, outQueue chan splitter.SplitItem) error {
	chs := make([]chan splitter.SplitItem, workers)
	for i := 0; i < int(workers); i++ {
		ch := make(chan splitter.SplitItem, 128)
		chs[i] = ch
		go client.consume(ch, outQueue)
	}
	go client.delegate(chs, uint32(workers))
	return nil
}

func (client *UDPClient) consume(inchan chan splitter.SplitItem, outQueue chan splitter.SplitItem) {
	shuts := client.shutdowner.Listen()
	defer shuts.Close()

	for {
		select {
		case splitem := <-inchan:
			client.server.ProcessSplitItem(splitem, outQueue)
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
		case splitem := <-client.inputQueue:
			hash := mmh3.Hash32(splitem.Key()) % workers
			inchan[hash] <- splitem
		case <-shuts.Ch:
			return
		}
	}
}

func (client *UDPClient) procLines(lines []byte, jobQueue chan dispatch.IJob, outQueue chan splitter.SplitItem) {
	for _, nline := range bytes.Split(lines, repr.NEWLINE_SEPARATOR_BYTES) {
		if len(nline) == 0 {
			continue
		}
		nline = bytes.TrimSpace(nline)
		if len(nline) == 0 {
			continue
		}
		client.server.AllLinesCount.Up(1)
		splitem, err := client.server.SplitterProcessor.ProcessLine(nline)
		//log.Notice("MOOO UDP line: %v MOOO", splitem.Fields())
		if err == nil {
			splitem.SetOrigin(splitter.UDP)
			splitem.SetOriginName(client.server.Name)
			//client.server.ProcessSplitItem(splitem, client.outQueue)
			stats.StatsdClient.Incr("incoming.udp.lines", 1)
			client.server.ValidLineCount.Up(1)
			client.inputQueue <- splitem
			//client.delegateOne(splitem, uint32(client.server.Workers))

			//performs worse
			//jobQueue <- udpJob{Client: client, Splitem: splitem, OutQueue: outQueue}

		} else {
			client.server.InvalidLineCount.Up(1)
			stats.StatsdClient.Incr("incoming.udp.invalidlines", 1)
			log.Warning("Invalid Line: %s (%s)", err, nline)
			continue
		}
	}
	return
}

func (client *UDPClient) run(outQueue chan splitter.SplitItem, closeClient chan bool) {
	shuts := client.shutdowner.Listen()
	defer shuts.Close()

	for {
		select {
		case splitem := <-client.inputQueue:
			client.server.ProcessSplitItem(splitem, outQueue)
		case <-shuts.Ch:
			return
		case <-closeClient:
			return
		}
	}
}

func (client *UDPClient) getLines(jobQueue chan dispatch.IJob, outQueue chan splitter.SplitItem) {
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
				client.procLines(buf[0:rlen], jobQueue, outQueue)
			}
		}
	}
}

func (client UDPClient) handleRequest(outQueue chan splitter.SplitItem, closeClient chan bool) {

	// UDP clients are basically "one" uber client (at least per socket)
	// DO NOT use work counts here, the UDP sockets are "multiplexed" using SO_CONNREUSE
	// so we have kernel level toggling between the various sockets so each "worker" is really
	// it's own little UDP listener land
	go client.run(outQueue, closeClient)
	go client.run(client.outQueue, closeClient) // bleed out non-socket inputs
	go client.getLines(nil, outQueue)

	return
}

func (client UDPClient) handleSend(outQueue chan splitter.SplitItem) {

	for {
		message := <-outQueue
		if message == nil || !message.IsValid() {
			break
		}
	}
	client.log.Notice("Closing UDP connection")
	//close it out
	client.Close()
	return
}
