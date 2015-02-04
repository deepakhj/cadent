/*
   The server we listen for our data
*/

package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/url"
	"runtime"
	"strings"
	"time"
)

const (
	WORKERS = 100
)

// if our Queue is full we need to block the input until its not
var WorkerHold = make(chan int64)
var InWorkQueue AtomicInt

// a server set of stats
type Server struct {
	ValidLineCount      StatCount
	InvalidLineCount    StatCount
	SuccessSendCount    StatCount
	FailSendCount       StatCount
	UnsendableSendCount StatCount
	UnknownSendCount    StatCount
	AllLinesCount       StatCount

	// our bound connection
	Connection net.Listener

	//pool the connections to the outgoing servers
	Outpool map[string]*Netpool

	ticker time.Duration

	WriteTimeout time.Duration
}

func (server *Server) ResetTickers() {
	server.ValidLineCount.ResetTick()
	server.InvalidLineCount.ResetTick()
	server.SuccessSendCount.ResetTick()
	server.FailSendCount.ResetTick()
	server.UnsendableSendCount.ResetTick()
	server.UnknownSendCount.ResetTick()
	server.AllLinesCount.ResetTick()
}

type Runner interface {
	run() string
}
type StatsdRunner struct {
	Client *Client
	Hasher *ConstHasher
	param  string
	params []string
}
type GraphiteRunner struct {
	Client *Client
	Hasher *ConstHasher
	param  string
	params []string
}
type UnknownRunner struct {
	Client *Client
	Hasher *ConstHasher
	param  string
	params []string
}

type SendOut struct {
	outserver string
	param     string
	client    *Client
}

func poolWorker(j *SendOut) {
	var outsrv *Netpool

	if val, ok := j.client.Server.Outpool[j.outserver]; ok {
		outsrv = val
	} else {
		m_url, err := url.Parse(j.outserver)
		if err != nil {
			j.client.Server.FailSendCount.Up(1)
			log.Printf("Error sending to backend Invalid URL %s", err)
			return
		}
		if len(j.client.Server.Outpool) == 0 {
			j.client.Server.Outpool = make(map[string]*Netpool)
		}
		outsrv = NewNetpool(m_url.Scheme, m_url.Host)
		j.client.Server.Outpool[j.outserver] = outsrv
	}
	conn, err := outsrv.Open()
	if err != nil {
		outsrv.Reset()
		j.client.Server.FailSendCount.Up(1)
		log.Printf("Error sending to backend %s", err)
		return
	}
	if conn != nil {
		// Conn.Write will raise a timeout error after 1 seconds
		conn.SetWriteDeadline(time.Now().Add(time.Second))
		_, err = conn.Write([]byte(j.param + "\n"))
		if err != nil {
			j.client.Server.FailSendCount.Up(1)
			outsrv.RemoveConn(conn)
			log.Printf("Error sending (writing) to backend: %s", err)
			return
		} else {
			outsrv.Close(conn)
		}

	} else {
		//tell the pool to reset the connection
		outsrv.Reset()
		j.client.Server.FailSendCount.Up(1)
		log.Printf("Error sending (writing connection gone) to backend: %s", err)
		return
	}

}
func singleWorker(j *SendOut) {

	m_url, err := url.Parse(j.outserver)
	if err != nil {
		j.client.Server.FailSendCount.Up(1)
		log.Printf("Error sending to backend Invalid URL %s", err)
		return
	}
	conn, err := net.DialTimeout(m_url.Scheme, m_url.Host, 5*time.Second)
	if conn != nil {

		//send it and close it
		_, err = conn.Write([]byte(j.param + "\n"))
		conn.Close()
		conn = nil
		if err != nil {
			j.client.Server.FailSendCount.Up(1)
			log.Printf("Error sending (writing) to backend: %s", err)
			return
		}
		j.client.Server.SuccessSendCount.Up(1)
	} else {
		log.Printf("Error sending (connection) to backend: %s", err)
		j.client.Server.FailSendCount.Up(1)
	}
}
func WorkerOutput(jobs <-chan *SendOut) {
	for j := range jobs {
		poolWorker(j)
	}

}

func (job StatsdRunner) run() string {
	//<key> <value>
	useme, err := job.Hasher.Get(job.params[0])
	if err == nil {
		job.Client.Server.ValidLineCount.Up(1)
		sendOut := &SendOut{
			outserver: useme,
			param:     job.param,
			client:    job.Client,
		}
		WorkerHold <- 1
		job.Client.worker_queue <- sendOut
		return fmt.Sprintf("yay statsd : %v : %v", string(useme), string(job.param))
	}
	job.Client.Server.UnsendableSendCount.Up(1)
	return fmt.Sprintf("ERROR ON statsd %s", err)
}

func (job GraphiteRunner) run() string {
	// <time> <key> <value>
	useme, err := job.Hasher.Get(job.params[1])
	if err == nil {
		job.Client.Server.ValidLineCount.Up(1)
		sendOut := &SendOut{
			outserver: useme,
			param:     job.param,
			client:    job.Client,
		}
		WorkerHold <- 1

		job.Client.worker_queue <- sendOut
		return fmt.Sprintf("yay graphite %s: %s", string(useme), string(job.param))
	}
	job.Client.Server.UnsendableSendCount.Up(1)
	return fmt.Sprintf("ERROR ON graphite %s", err)
}

func (job UnknownRunner) run() string {
	job.Client.Server.UnknownSendCount.Up(1)
	return "ACK no idea what message i'm supposed to parse"
}

func RunRunner(job Runner, out chan string) {
	//direct timer to void leaks (i.e. NOT time.After(...))
	timer := time.NewTimer(500 * time.Millisecond)

	select {
	case out <- job.run() + "\n":
		timer.Stop()
		WorkerHold <- -1
	case <-timer.C:
		WorkerHold <- -1
		log.Printf("Job Channel Runner Timeout")

	}
}

func Factory(client *Client, input string) Runner {
	array := strings.Split(input, " ")
	if len(array) == 2 {
		return StatsdRunner{
			Client: client,
			Hasher: client.Hasher,
			param:  input,
			params: array,
		}
	} else if len(array) == 3 {
		return GraphiteRunner{
			Client: client,
			Hasher: client.Hasher,
			param:  input,
			params: array,
		}
	}

	return UnknownRunner{
		Client: client,
		Hasher: client.Hasher,
		param:  "",
	}
}

type Client struct {
	Server *Server
	Hasher *ConstHasher

	Connection   net.Conn
	LineCount    uint64
	MaxLineCount uint64
	CycleCount   uint64
	//ins and outs
	writer *bufio.Writer
	reader *bufio.Reader

	channel      chan string
	done         chan *Client
	worker_queue chan *SendOut
}

func NewClient(server *Server, hasher *ConstHasher, conn net.Conn, worker_queue chan *SendOut, done chan *Client) *Client {

	client := new(Client)
	client.Server = server
	client.Hasher = hasher

	//client.writer = bufio.NewWriter(conn)
	client.reader = bufio.NewReader(conn)
	client.LineCount = 0

	// we "parrael" this many processes then block until we are done
	client.MaxLineCount = 1024
	client.CycleCount = 0
	client.Connection = conn

	client.channel = make(chan string)
	client.worker_queue = worker_queue
	client.done = done
	return client

}

// close the 2 hooks, channel and connection
func (client *Client) Close() {
	close(client.channel)
	client.Connection.Close()
}

func (client *Client) handleRequest() {

	for {

		line, err := client.reader.ReadString('\n')
		if err != nil || len(line) == 0 {
			break
		}
		client.LineCount += 1

		job := Factory(client, strings.Trim(line, "\n\t "))
		RunRunner(job, client.channel)
	}
	//close it
	//client.done <- client

}

func (client *Client) handleSend() {

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

func NewServer(cfg *ConstHashConfig) (connection *Server, err error) {
	log.Printf("Binding server to %s", cfg.ListenURL.String())

	serv := new(Server)

	conn, err := net.Listen(cfg.ListenURL.Scheme, cfg.ListenURL.Host)
	if err != nil {
		return nil, fmt.Errorf("Error binding: %s", err)
	}
	serv.Connection = conn
	serv.ticker = time.Duration(5) * time.Second
	go serv.tickDisplay()
	return serv, nil

}

// a little function to log out some collected stats
func (server *Server) tickDisplay() {

	log.Printf("Server: ValidLineCount: %d", server.ValidLineCount.TotalCount)
	log.Printf("Server: InvalidLineCount: %d", server.InvalidLineCount.TotalCount)
	log.Printf("Server: SuccessSendCount: %d", server.SuccessSendCount.TotalCount)
	log.Printf("Server: FailSendCount: %d", server.FailSendCount.TotalCount)
	log.Printf("Server: UnsendableSendCount: %d", server.UnsendableSendCount.TotalCount)
	log.Printf("Server: UnknownSendCount: %d", server.UnknownSendCount.TotalCount)
	log.Printf("Server: AllLinesCount: %d", server.AllLinesCount.TotalCount)
	log.Printf("Server: GO Routines Running: %d", runtime.NumGoroutine())
	log.Printf("-------")
	log.Printf("Server Rate: Duration %ds", uint64(server.ticker/time.Second))
	log.Printf("Server Rate: ValidLineCount: %.2f/s", server.ValidLineCount.Rate(server.ticker))
	log.Printf("Server Rate: InvalidLineCount: %.2f/s", server.InvalidLineCount.Rate(server.ticker))
	log.Printf("Server Rate: SuccessSendCount: %.2f/s", server.SuccessSendCount.Rate(server.ticker))
	log.Printf("Server Rate: FailSendCount: %.2f/s", server.FailSendCount.Rate(server.ticker))
	log.Printf("Server Rate: UnsendableSendCount: %.2f/s", server.UnsendableSendCount.Rate(server.ticker))
	log.Printf("Server Rate: UnknownSendCount: %.2f/s", server.UnknownSendCount.Rate(server.ticker))
	log.Printf("Server Rate: AllLinesCount: %.2f/s", server.AllLinesCount.Rate(server.ticker))

	server.ResetTickers()
	runtime.GC()
	time.Sleep(server.ticker)
	go server.tickDisplay()
}

// accept oyr
func (server *Server) Accepter() (<-chan net.Conn, error) {

	conns := make(chan net.Conn)
	go func() {
		defer server.Connection.Close()
		defer close(conns)
		for {
			conn, err := server.Connection.Accept()
			if err != nil {
				log.Printf("Error Accecption Connection: %s", err)
				return
			}
			log.Printf("Accepted connection from %s", conn.RemoteAddr())

			conns <- conn
		}
	}()
	return conns, nil
}

func startServer(cfg *ConstHashConfig, hasher *ConstHasher) {
	server, err := NewServer(cfg)
	if err != nil {
		return
	}

	accepts, err := server.Accepter()
	done := make(chan *Client)
	worker_queue := make(chan *SendOut)

	//fire up the send to workers
	for w := 1; w <= WORKERS; w++ {
		go WorkerOutput(worker_queue)
	}

	for {
		select {
		case conn, ok := <-accepts:
			if !ok {
				return
			}
			client := NewClient(server, hasher, conn, worker_queue, done)
			go client.handleRequest()
			go client.handleSend()
		case workerUpDown := <-WorkerHold:
			InWorkQueue.Add(workerUpDown)
			if InWorkQueue.Get() == WORKERS {
				log.Printf("Worker Queue Full %d", InWorkQueue.Get())
			}
		case client := <-done:
			client.Close()
		}
	}

}
