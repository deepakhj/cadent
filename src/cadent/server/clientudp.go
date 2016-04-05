/*
	UDP clients
*/

package cadent

import (
	"cadent/server/broadcast"
	"cadent/server/dispatch"
	"cadent/server/splitter"
	"cadent/server/stats"
	logging "gopkg.in/op/go-logging.v1"
	"net"
	"strings"
)

// 1Mb default buffer size
const UDP_BUFFER_SIZE = 1048576
const UDP_LINE_QUEUE_SIZE = 1024 * 1024
const UDP_WORKER_SIZE = 32

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

	Connection *net.UDPConn
	LineCount  uint64
	BufferSize int

	out_queue    chan splitter.SplitItem
	done         chan Client
	input_queue  chan splitter.SplitItem
	worker_queue chan *OutputMessage
	shutdowner   *broadcast.Broadcaster

	// UDP clients are basically "one" uber client (at least per socket)
	// to handle the bursts properly, we need to have a proper dispatch queue
	disp_work_queue       chan dispatch.IJob
	disp_dispatch_queue   chan chan dispatch.IJob
	disp_write_dispatcher *dispatch.Dispatch

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
	client.shutdowner = broadcast.New(0)
	client.line_queue = make(chan string, server.Workers)

	/**** dispatcher queue **
	**** PERFORMS WORSE then just straight buffered channels ****
	workers := UDP_WORKER_SIZE
	client.disp_work_queue = make(chan dispatch.IJob, UDP_LINE_QUEUE_SIZE)
	client.disp_dispatch_queue = make(chan chan dispatch.IJob, workers)
	client.disp_write_dispatcher = dispatch.NewDispatch(workers, client.disp_dispatch_queue, client.disp_work_queue)
	client.disp_write_dispatcher.SetRetries(2)
	client.disp_write_dispatcher.Run()
	*/
	return client

}

func (client *UDPClient) ShutDown() {
	client.shutdowner.Send(true)
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

func (client *UDPClient) procLines(line string, job_queue chan dispatch.IJob, out_queue chan splitter.SplitItem) {
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
			splitem.SetOriginName(client.server.Name)
			//client.server.ProcessSplitItem(splitem, client.out_queue)
			stats.StatsdClient.Incr("incoming.udp.lines", 1)
			client.server.ValidLineCount.Up(1)
			client.input_queue <- splitem

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
func (client *UDPClient) run(out_queue chan splitter.SplitItem) {
	shuts := client.shutdowner.Listen()
	defer shuts.Close()
	/*for splitem := range client.input_queue {
		client.server.ProcessSplitItem(splitem, out_queue)
	}

	return*/

	for {
		select {
		case splitem := <-client.input_queue:
			client.server.ProcessSplitItem(splitem, out_queue)
		case <-shuts.Ch:
			return
		}
	}
	return
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
			rlen, _, _ := client.Connection.ReadFromUDP(buf[:])
			client.server.BytesReadCount.Up(uint64(rlen))

			in_str := string(buf[0:rlen])
			if rlen > 0 {
				client.procLines(in_str, job_queue, out_queue)
			}
		}
	}
	return
}

func (client UDPClient) handleRequest(out_queue chan splitter.SplitItem) {

	for w := int64(1); w <= client.server.Workers; w++ {
		go client.run(out_queue)
		go client.run(client.out_queue) // bleed out non-socket inputs
		go client.getLines(client.disp_work_queue, out_queue)
	}

	//go client.getLines(job_queue, client.out_queue)
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
