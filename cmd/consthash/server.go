/*
   The server we listen for our data
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	DEFAULT_WORKERS = int64(500)
)

type SendOut struct {
	outserver string
	param     string
	client    Client
	server    *Server
}

// to send stat lines via a pool of connections
// rather then one socket per stat
func poolWorker(j *SendOut) {
	defer StatsdNanoTimeFunc("worker.process-time-ns", time.Now())

	var outsrv *Netpool

	if val, ok := j.server.Outpool[j.outserver]; ok {
		outsrv = val
	} else {
		m_url, err := url.Parse(j.outserver)
		if err != nil {
			StatsdClient.Incr("failed.bad-url", 1)
			j.server.FailSendCount.Up(1)
			log.Printf("Error sending to backend Invalid URL %s", err)
			return
		}
		if len(j.server.Outpool) == 0 {
			j.server.Outpool = make(map[string]*Netpool)
		}
		outsrv = NewNetpool(m_url.Scheme, m_url.Host)
		j.server.Outpool[j.outserver] = outsrv
	}
	conn, err := outsrv.Open()
	if err != nil {
		outsrv.Reset()
		StatsdClient.Incr("failed.bad-connection", 1)

		j.server.FailSendCount.Up(1)
		log.Printf("Error sending to backend %s", err)
		return
	}
	if conn != nil {
		// Conn.Write will raise a timeout error after 1 seconds
		conn.SetWriteDeadline(time.Now().Add(time.Second))
		to_send := []byte(j.param + "\n")
		_, err = conn.Write(to_send)
		if err != nil {
			StatsdClient.Incr("failed.connection-timeout", 1)
			j.server.FailSendCount.Up(1)
			outsrv.RemoveConn(conn)
			log.Printf("Error sending (writing) to backend: %s", err)
			return
		} else {
			StatsdClient.Incr("success.send", 1)
			StatsdClient.Incr("success.sent-bytes", int64(len(to_send)))
			outsrv.Close(conn)
		}

	} else {
		//tell the pool to reset the connection
		outsrv.Reset()
		j.server.SuccessSendCount.Up(1)
		StatsdClient.Incr("failed.aborted-connection", 1)
		j.server.FailSendCount.Up(1)
		log.Printf("Error sending (writing connection gone) to backend: %s", err)
		return
	}

}

//this is for using a simple tcp connection per stat we send out
//one can quickly run out of sockets if this is used under high load
func singleWorker(j *SendOut) {
	defer StatsdNanoTimeFunc("worker.process-time-ns", time.Now())

	m_url, err := url.Parse(j.outserver)
	if err != nil {
		StatsdClient.Incr("failed.bad-url", 1)
		j.server.FailSendCount.Up(1)
		log.Printf("Error sending to backend Invalid URL %s", err)
		return
	}
	conn, err := net.DialTimeout(m_url.Scheme, m_url.Host, 5*time.Second)
	if conn != nil {

		//send it and close it
		to_send := []byte(j.param + "\n")
		_, err = conn.Write(to_send)
		conn.Close()
		conn = nil
		if err != nil {
			StatsdClient.Incr("failed.bad-connection", 1)
			j.server.FailSendCount.Up(1)
			log.Printf("Error sending (writing) to backend: %s", err)
			return
		}
		StatsdClient.Incr("success.sent", 1)
		StatsdClient.Incr("success.sent-bytes", int64(len(to_send)))
		j.server.SuccessSendCount.Up(1)
	} else {
		log.Printf("Error sending (connection) to backend: %s", err)
		j.server.FailSendCount.Up(1)
	}
}

//spins up the queue of go routines to handle outgoing
func WorkerOutput(jobs <-chan *SendOut) {
	for j := range jobs {
		poolWorker(j)
	}
}

/****************** RUNNERS *********************/

type Runner interface {
	run() string
	Client() Client
	GetKey() string
}

type StatsdRunner struct {
	client    Client
	Hasher    *ConstHasher
	param     string
	key_param string
	params    []string
}

func NewStatsdRunner(client Client, conf map[string]interface{}, param string) (*StatsdRunner, error) {

	//<key>:blaaa
	job := &StatsdRunner{
		client: client,
		Hasher: client.Hasher(),
		param:  param,
	}
	statd_array := strings.Split(param, ":")
	if len(statd_array) == 2 {
		job.param = param
		job.key_param = statd_array[0]
		job.params = statd_array
		return job, nil
	}
	return nil, fmt.Errorf("Invalid Statsd line")
}

func (job StatsdRunner) Client() Client {
	return job.client
}

func (job StatsdRunner) GetKey() string {
	//<key>:blaaa
	return job.key_param
}

func (job StatsdRunner) run() string {
	//<key> <value>
	useme, err := job.Hasher.Get(job.GetKey())
	if err == nil {
		StatsdClient.Incr("success.valid-lines", 1)
		job.Client().Server().ValidLineCount.Up(1)
		sendOut := &SendOut{
			outserver: useme,
			param:     job.param,
			server:    job.Client().Server(),
			client:    job.Client(),
		}
		job.Client().Server().WorkerHold <- 1
		job.Client().WorkerQueue() <- sendOut
		return fmt.Sprintf("yay statsd : %v : %v", string(useme), string(job.param))
	}
	StatsdClient.Incr("failed.invalid-hash-server", 1)
	job.Client().Server().UnsendableSendCount.Up(1)
	return fmt.Sprintf("ERROR ON statsd %s", err)
}

type GraphiteRunner struct {
	client    Client
	Hasher    *ConstHasher
	key_param string
	param     string
	params    []string
}

func NewGraphiteRunner(client Client, conf map[string]interface{}, param string) (*GraphiteRunner, error) {

	//<key>:blaaa
	job := &GraphiteRunner{
		client: client,
		Hasher: client.Hasher(),
		param:  param,
	}
	graphite_array := strings.Split(param, " ")
	if len(graphite_array) == 3 {
		job.param = param
		job.key_param = graphite_array[1]
		job.params = graphite_array
		return job, nil
	}
	return nil, fmt.Errorf("Invalid Graphite line")
}

func (job GraphiteRunner) Client() Client {
	return job.client
}
func (job GraphiteRunner) GetKey() string {
	//<time> <key> <value>
	return job.key_param
}

func (job GraphiteRunner) run() string {
	// <time> <key> <value>
	useme, err := job.Hasher.Get(job.GetKey())
	if err == nil {
		StatsdClient.Incr("success.valid-lines", 1)
		job.Client().Server().ValidLineCount.Up(1)
		sendOut := &SendOut{
			outserver: useme,
			server:    job.Client().Server(),
			param:     job.param,
			client:    job.Client(),
		}
		job.Client().Server().WorkerHold <- 1

		job.Client().WorkerQueue() <- sendOut
		return fmt.Sprintf("yay graphite %s: %s", string(useme), string(job.param))
	}
	StatsdClient.Incr("failed.invalid-hash-server", 1)
	job.Client().Server().UnsendableSendCount.Up(1)
	return fmt.Sprintf("ERROR ON graphite %s", err)
}

type RegExRunner struct {
	client          Client
	Hasher          *ConstHasher
	param           string
	key_regex       *regexp.Regexp
	key_regex_names []string
	key_param       string
	params          []string
}

func NewRegExRunner(client Client, conf map[string]interface{}, param string) (*RegExRunner, error) {

	//<key>:blaaa
	job := &RegExRunner{
		client: client,
		Hasher: client.Hasher(),
		param:  param,
	}
	job.key_regex = conf["regexp"].(*regexp.Regexp)
	job.key_regex_names = conf["regexpNames"].([]string)

	matched := job.key_regex.FindAllStringSubmatch(param, -1)[0]
	for i, n := range matched {
		//fmt.Printf("%d. match='%s'\tname='%s'\n", i, n, n1[i])
		if job.key_regex_names[i] == "Key" {
			job.key_param = n
		}
	}

	if len(job.key_param) > 0 {
		job.param = param
		job.params = matched
		return job, nil
	}
	return nil, fmt.Errorf("Invalid RegEx line")
}

func (job RegExRunner) Client() Client {
	return job.client
}

func (job RegExRunner) GetKey() string {
	return job.key_param
}

func (job RegExRunner) run() string {
	// <time> <key> <value>
	useme, err := job.Hasher.Get(job.GetKey())
	if err == nil {
		StatsdClient.Incr("success.valid-lines", 1)
		job.Client().Server().ValidLineCount.Up(1)
		sendOut := &SendOut{
			outserver: useme,
			server:    job.Client().Server(),
			param:     job.param,
			client:    job.Client(),
		}
		job.Client().Server().WorkerHold <- 1

		job.Client().WorkerQueue() <- sendOut
		return fmt.Sprintf("yay regex %s: %s", string(useme), string(job.param))
	}
	StatsdClient.Incr("failed.invalid-hash-server", 1)
	job.Client().Server().UnsendableSendCount.Up(1)
	return fmt.Sprintf("ERROR ON graphite %s", err)
}

type UnknownRunner struct {
	client Client
	Hasher *ConstHasher
	param  string
	params []string
}

func (job UnknownRunner) Client() Client {
	return job.client
}

func (job UnknownRunner) run() string {
	StatsdClient.Incr("failed.unknown-lines", 1)
	job.Client().Server().UnknownSendCount.Up(1)
	return "ACK no idea what message i'm supposed to parse"
}

func (job UnknownRunner) GetKey() string {
	return ""
}

func RunRunner(job Runner, out chan string) {
	//direct timer to void leaks (i.e. NOT time.After(...))
	timer := time.NewTimer(500 * time.Millisecond)

	select {
	case out <- job.run() + "\n":
		timer.Stop()
		job.Client().Server().WorkerHold <- -1
	case <-timer.C:
		job.Client().Server().WorkerHold <- -1
		log.Printf("Job Channel Runner Timeout")
	}
}

func NewRunner(client Client, line string) (Runner, error) {
	msg_type := client.Server().RunnerTypeString

	defer StatsdNanoTimeFunc(fmt.Sprintf("factory.%s.process-time-ns", msg_type), time.Now())
	client.Server().AllLinesCount.Up(1)

	var runner Runner
	var err error
	if msg_type == "statsd" {
		runner, err = NewStatsdRunner(client, client.Server().RunnerConfig, line)
	} else if msg_type == "graphite" {
		runner, err = NewGraphiteRunner(client, client.Server().RunnerConfig, line)
	} else if msg_type == "regex" {
		runner, err = NewRegExRunner(client, client.Server().RunnerConfig, line)
	} else {
		runner = UnknownRunner{
			client: client,
			Hasher: client.Hasher(),
		}
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to configure Runner, aborting")
	}

	return runner, nil

}

/****************** CLIENTS *********************/

type Client interface {
	handleRequest()
	handleSend()
	Close()
	//Runner() *Runner
	Server() *Server
	Hasher() *ConstHasher
	WorkerQueue() chan *SendOut
}

type TCPClient struct {
	server *Server
	hasher *ConstHasher

	Connection   net.Conn
	LineCount    uint64
	MaxLineCount uint64
	CycleCount   uint64
	//ins and outs
	writer *bufio.Writer
	reader *bufio.Reader

	channel      chan string
	done         chan Client
	worker_queue chan *SendOut
}

func NewTCPClient(server *Server, hasher *ConstHasher, conn net.Conn, worker_queue chan *SendOut, done chan Client) *TCPClient {

	client := new(TCPClient)
	client.server = server
	client.hasher = hasher

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

func (client TCPClient) Server() (server *Server) {
	return client.server
}

func (client TCPClient) Hasher() (server *ConstHasher) {
	return client.hasher
}
func (client TCPClient) WorkerQueue() chan *SendOut {
	return client.worker_queue
}

// close the 2 hooks, channel and connection
func (client TCPClient) Close() {
	close(client.channel)
	client.Connection.Close()
	client.server = nil
	client.hasher = nil

}

func (client TCPClient) handleRequest() {

	for {

		line, err := client.reader.ReadString('\n')
		if err != nil || len(line) == 0 {
			break
		}
		client.LineCount += 1

		job, err := NewRunner(client, strings.Trim(line, "\n\t "))
		if err == nil {
			RunRunner(job, client.channel)
		}
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

type UDPClient struct {
	server *Server
	hasher *ConstHasher

	Connection   *net.UDPConn
	LineCount    uint64
	MaxLineCount uint64
	CycleCount   uint64

	channel      chan string
	done         chan Client
	worker_queue chan *SendOut
}

func NewUDPClient(server *Server, hasher *ConstHasher, conn *net.UDPConn, worker_queue chan *SendOut, done chan Client) *UDPClient {

	client := new(UDPClient)
	client.server = server
	client.hasher = hasher

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

func (client UDPClient) Hasher() (hasher *ConstHasher) {
	return client.hasher
}
func (client UDPClient) WorkerQueue() chan *SendOut {
	return client.worker_queue
}
func (client UDPClient) Close() {
	client.server = nil
	client.hasher = nil
	if client.Connection != nil {
		client.Connection.Close()
	}
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

			client.LineCount += 1
			job, err := NewRunner(client, strings.Trim(line, "\n\t "))
			if err == nil {
				go RunRunner(job, client.channel)
			}
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

/****************** SERVERS *********************/

//helper object for json'ing the basic stat data
type ServerStats struct {
	ValidLineCount            int64    `json:"valid_line_count"`
	InvalidLineCount          int64    `json:"invalid_line_count"`
	SuccessSendCount          int64    `json:"success_send_count"`
	FailSendCount             int64    `json:"fail_send_count"`
	UnsendableSendCount       int64    `json:"unsendable_send_count"`
	UnknownSendCount          int64    `json:"unknown_send_count"`
	AllLinesCount             int64    `json:"all_lines_count"`
	GoRoutines                int      `json:"go_routines"`
	UpTimeSeconds             int64    `json:"uptime_sec"`
	ValidLineCountPerSec      float32  `json:"valid_line_count_persec"`
	InvalidLineCountPerSec    float32  `json:"invalid_line_count_persec"`
	SuccessSendCountPerSec    float32  `json:"success_send_count_persec"`
	UnsendableSendCountPerSec float32  `json:"unsendable_count_persec"`
	UnknownSendCountPerSec    float32  `json:"unknown_send_count_persec"`
	AllLinesCountPerSec       float32  `json:"all_lines_count_persec"`
	ServersUp                 []string `json:"servers_up"`
}

// a server set of stats
type Server struct {
	ValidLineCount      StatCount
	InvalidLineCount    StatCount
	SuccessSendCount    StatCount
	FailSendCount       StatCount
	UnsendableSendCount StatCount
	UnknownSendCount    StatCount
	AllLinesCount       StatCount

	// our bound connection if TCP
	Connection net.Listener
	UDPConn    *net.UDPConn

	//Hasher objects
	Hasher *ConstHasher

	//pool the connections to the outgoing servers
	Outpool map[string]*Netpool

	ticker time.Duration

	WriteTimeout time.Duration

	//workers and ques sizes
	WorkerHold  chan int64
	InWorkQueue AtomicInt
	Workers     int64

	//the Runner type to determin "keys"
	RunnerTypeString string
	RunnerConfig     map[string]interface{}

	//uptime
	StartTime time.Time

	stats        ServerStats
	listen_stats string
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

func NewServer(cfg *Config) (connection *Server, err error) {
	log.Printf("Binding server to %s", cfg.ListenURL.String())

	serv := new(Server)
	serv.StartTime = time.Now()
	serv.WorkerHold = make(chan int64)
	serv.listen_stats = cfg.HealthServerBind

	//find the runner types
	serv.RunnerTypeString = cfg.MsgType
	serv.RunnerConfig = cfg.MsgConfig

	if cfg.ListenURL.Scheme == "udp" {
		udp_addr, err := net.ResolveUDPAddr(cfg.ListenURL.Scheme, cfg.ListenURL.Host)
		if err != nil {
			return nil, fmt.Errorf("Error binding: %s", err)
		}
		conn, err := net.ListenUDP(cfg.ListenURL.Scheme, udp_addr)
		if err != nil {
			return nil, fmt.Errorf("Error binding: %s", err)
		}
		serv.UDPConn = conn
		serv.UDPConn.SetReadBuffer(1048576) //set buffer size to 1024 bytes

	} else {
		conn, err := net.Listen(cfg.ListenURL.Scheme, cfg.ListenURL.Host)
		if err != nil {
			return nil, fmt.Errorf("Error binding: %s", err)
		}
		serv.Connection = conn
	}

	serv.ticker = time.Duration(5) * time.Second
	go serv.tickDisplay()
	return serv, nil

}

// dump some json data about the stats and server status
func (server *Server) StatsJsonString() string {

	elapsed := time.Since(server.StartTime)
	elasped_sec := float64(elapsed) / float64(time.Second)

	server.stats.ValidLineCount = server.ValidLineCount.TotalCount.Get()
	server.stats.InvalidLineCount = server.InvalidLineCount.TotalCount.Get()
	server.stats.SuccessSendCount = server.SuccessSendCount.TotalCount.Get()
	server.stats.FailSendCount = server.FailSendCount.TotalCount.Get()
	server.stats.UnknownSendCount = server.UnknownSendCount.TotalCount.Get()
	server.stats.UnknownSendCount = server.UnknownSendCount.TotalCount.Get()
	server.stats.AllLinesCount = server.AllLinesCount.TotalCount.Get()

	server.stats.GoRoutines = runtime.NumGoroutine()
	server.stats.UpTimeSeconds = int64(elasped_sec)

	server.stats.ValidLineCountPerSec = server.ValidLineCount.TotalRate(elapsed)
	server.stats.InvalidLineCountPerSec = server.InvalidLineCount.TotalRate(elapsed)
	server.stats.SuccessSendCountPerSec = server.SuccessSendCount.TotalRate(elapsed)
	server.stats.UnsendableSendCountPerSec = server.UnsendableSendCount.TotalRate(elapsed)
	server.stats.UnknownSendCountPerSec = server.UnknownSendCount.TotalRate(elapsed)
	server.stats.AllLinesCountPerSec = server.AllLinesCount.TotalRate(elapsed)
	server.stats.ServersUp = server.Hasher.Members()

	resbytes, _ := json.Marshal(server.stats)
	return string(resbytes)
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

// accept incoming TCP connections and push them into the
// a connection channel
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

func (server *Server) startTCPServer(hasher *ConstHasher, worker_queue chan *SendOut, done chan Client) {

	accepts, err := server.Accepter()
	if err != nil {
		panic(err)
	}
	for {
		select {
		case conn, ok := <-accepts:
			if !ok {
				return
			}
			client := NewTCPClient(server, hasher, conn, worker_queue, done)
			go client.handleRequest()
			go client.handleSend()
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			if server.InWorkQueue.Get() == server.Workers {
				log.Printf("Worker Queue Full %d", server.InWorkQueue.Get())
			}
		case client := <-done:
			client.Close()
		}
	}

}

// Fire up the http server for stats and healthchecks
func (server *Server) startStatsServer() {
	log.Printf("Starting Status server on %s", server.listen_stats)

	stats := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, server.StatsJsonString())
	}
	status := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	}

	http.HandleFunc("/", stats)
	http.HandleFunc("/ops/status", status)
	http.HandleFunc("/ops/status/", status)
	http.HandleFunc("/status/", status)
	http.HandleFunc("/stats/", stats)
	http.HandleFunc("/stats", stats)
	http.ListenAndServe(server.listen_stats, nil)

}

// different mechanism for UDP servers
func (server *Server) startUDPServer(hasher *ConstHasher, worker_queue chan *SendOut, done chan Client) {

	//just need on "client" here as we simply just pull from the socket
	client := NewUDPClient(server, hasher, server.UDPConn, worker_queue, done)
	go client.handleRequest()
	go client.handleSend()
	for {
		select {
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			if server.InWorkQueue.Get() == server.Workers {
				log.Printf("Worker Queue Full %d", server.InWorkQueue.Get())
			}
		case client := <-done:
			client.Close()
		}
	}
}

func startServer(cfg *Config, hasher *ConstHasher) {

	server, err := NewServer(cfg)
	server.Hasher = hasher
	if err != nil {
		panic(err)
	}
	done := make(chan Client)
	worker_queue := make(chan *SendOut)

	server.Workers = DEFAULT_WORKERS
	if cfg.Workers > 0 {
		server.Workers = int64(cfg.Workers)
	}

	log.Printf("Using %d workers to process output", server.Workers)
	//fire up the send to workers
	for w := int64(1); w <= server.Workers; w++ {
		go WorkerOutput(worker_queue)
	}

	//fire up the http stats if given
	if len(server.listen_stats) != 0 {
		go server.startStatsServer()
	}

	if server.UDPConn != nil {
		server.startUDPServer(hasher, worker_queue, done)
	} else {
		server.startTCPServer(hasher, worker_queue, done)
	}

}
