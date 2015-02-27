/*
   The server we listen for our data
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"
)

const (
	DEFAULT_WORKERS                    = int64(500)
	DEFAULT_NUM_STATS                  = 100
	DEFAULT_SENDING_CONNECTIONS_METHOD = "pool"
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
		if j.server.NetPoolConnections > 0 {
			outsrv.MaxConnections = j.server.NetPoolConnections
		}
		// populate it
		outsrv.WarmPool()
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
			j.server.SuccessSendCount.Up(1)
			StatsdClient.Incr("success.send", 1)
			StatsdClient.Incr("success.sent-bytes", int64(len(to_send)))
			outsrv.Close(conn)
		}

	} else {
		//tell the pool to reset the connection
		outsrv.Reset()
		outsrv.WarmPool()
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
func WorkerOutput(jobs <-chan *SendOut, sendmethod string) {
	for j := range jobs {
		if sendmethod == "single" {
			singleWorker(j)

		} else {
			poolWorker(j)
		}
	}
}

func RunRunner(job Runner, out chan string) {
	//direct timer to void leaks (i.e. NOT time.After(...))
	timer := time.NewTimer(500 * time.Millisecond)

	select {
	case out <- job.run() + "\n":
		timer.Stop()
		job.Client().Server().WorkerHold <- -1
	case <-timer.C:
		timer.Stop()
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

/****************** SERVERS *********************/

//helper object for json'ing the basic stat data
type ServerStats struct {
	ValidLineCount      int64 `json:"valid_line_count"`
	InvalidLineCount    int64 `json:"invalid_line_count"`
	SuccessSendCount    int64 `json:"success_send_count"`
	FailSendCount       int64 `json:"fail_send_count"`
	UnsendableSendCount int64 `json:"unsendable_send_count"`
	UnknownSendCount    int64 `json:"unknown_send_count"`
	AllLinesCount       int64 `json:"all_lines_count"`

	CurrentValidLineCount      int64 `json:"current_valid_line_count"`
	CurrentInvalidLineCount    int64 `json:"current_invalid_line_count"`
	CurrentSuccessSendCount    int64 `json:"current_success_send_count"`
	CurrentFailSendCount       int64 `json:"current_fail_send_count"`
	CurrentUnsendableSendCount int64 `json:"current_unsendable_send_count"`
	CurrentUnknownSendCount    int64 `json:"current_unknown_send_count"`
	CurrentAllLinesCount       int64 `json:"current_all_lines_count"`

	ValidLineCountList      []int64 `json:"valid_line_count_list"`
	InvalidLineCountList    []int64 `json:"invalid_line_count_list"`
	SuccessSendCountList    []int64 `json:"success_send_count_list"`
	FailSendCountList       []int64 `json:"fail_send_count_list"`
	UnsendableSendCountList []int64 `json:"unsendable_send_count_list"`
	UnknownSendCountList    []int64 `json:"unknown_send_count_list"`
	AllLinesCountList       []int64 `json:"all_lines_count_list"`
	GoRoutinesList          []int   `json:"go_routines_list"`
	TicksList               []int64 `json:ticks_list`

	GoRoutines                int      `json:"go_routines"`
	UpTimeSeconds             int64    `json:"uptime_sec"`
	ValidLineCountPerSec      float32  `json:"valid_line_count_persec"`
	InvalidLineCountPerSec    float32  `json:"invalid_line_count_persec"`
	SuccessSendCountPerSec    float32  `json:"success_send_count_persec"`
	UnsendableSendCountPerSec float32  `json:"unsendable_count_persec"`
	UnknownSendCountPerSec    float32  `json:"unknown_send_count_persec"`
	AllLinesCountPerSec       float32  `json:"all_lines_count_persec"`
	Listening                 string   `json:"listening"`
	ServersUp                 []string `json:"servers_up"`
	ServersDown               []string `json:"servers_down"`
	ServersChecks             []string `json:"servers_checking"`
}

// a server set of stats
type Server struct {
	Name      string
	ListenURL *url.URL

	ValidLineCount      StatCount
	InvalidLineCount    StatCount
	SuccessSendCount    StatCount
	FailSendCount       StatCount
	UnsendableSendCount StatCount
	UnknownSendCount    StatCount
	AllLinesCount       StatCount
	NumStats            uint

	// our bound connection if TCP
	Connection net.Listener
	UDPConn    *net.UDPConn

	//Hasher objects
	Hasher *ConstHasher

	// we can use a "pool" of connections, or single connecitons per line
	// performance will be depending on the system and work load tcp vs udp, etc
	// "pool" or "single"
	// default is pool
	SendingConnectionMethod string
	//number of connections in the NetPool
	NetPoolConnections int

	//number of replicas to fire data to (i.e. dupes)
	Replicas int

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

	stats ServerStats

	Logger *log.Logger
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

func NewServer(cfg *Config) (server *Server, err error) {

	serv := new(Server)
	serv.Name = cfg.Name
	serv.ListenURL = cfg.ListenURL
	serv.StartTime = time.Now()
	serv.WorkerHold = make(chan int64)

	serv.Logger = log.New(os.Stdout, fmt.Sprintf("[Server: %s] ", serv.Name), log.Ldate|log.Ltime)
	serv.Logger.Printf("Binding server to %s", cfg.ListenURL.String())

	//find the runner types
	serv.RunnerTypeString = cfg.MsgType
	serv.RunnerConfig = cfg.MsgConfig
	serv.Replicas = cfg.Replicas

	serv.NumStats = DEFAULT_NUM_STATS
	if cfg.HealthServerPoints > 0 {
		serv.NumStats = cfg.HealthServerPoints
	}

	serv.NetPoolConnections = cfg.MaxPoolConnections
	serv.SendingConnectionMethod = DEFAULT_SENDING_CONNECTIONS_METHOD

	if len(cfg.SendingConnectionMethod) > 0 {
		serv.SendingConnectionMethod = cfg.SendingConnectionMethod
	}

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

func (server *Server) StatsTick() {
	elapsed := time.Since(server.StartTime)
	elasped_sec := float64(elapsed) / float64(time.Second)
	t_stamp := time.Now().UnixNano()

	server.stats.ValidLineCount = server.ValidLineCount.TotalCount.Get()
	server.stats.InvalidLineCount = server.InvalidLineCount.TotalCount.Get()
	server.stats.SuccessSendCount = server.SuccessSendCount.TotalCount.Get()
	server.stats.FailSendCount = server.FailSendCount.TotalCount.Get()
	server.stats.UnsendableSendCount = server.UnsendableSendCount.TotalCount.Get()
	server.stats.UnknownSendCount = server.UnknownSendCount.TotalCount.Get()
	server.stats.AllLinesCount = server.AllLinesCount.TotalCount.Get()

	server.stats.CurrentValidLineCount = server.ValidLineCount.TickCount.Get()
	server.stats.CurrentInvalidLineCount = server.InvalidLineCount.TickCount.Get()
	server.stats.CurrentSuccessSendCount = server.SuccessSendCount.TickCount.Get()
	server.stats.CurrentFailSendCount = server.FailSendCount.TickCount.Get()
	server.stats.CurrentUnknownSendCount = server.UnknownSendCount.TickCount.Get()
	server.stats.CurrentAllLinesCount = server.AllLinesCount.TickCount.Get()

	server.stats.ValidLineCountList = append(server.stats.ValidLineCountList, server.ValidLineCount.TickCount.Get())
	server.stats.InvalidLineCountList = append(server.stats.InvalidLineCountList, server.InvalidLineCount.TickCount.Get())
	server.stats.SuccessSendCountList = append(server.stats.SuccessSendCountList, server.SuccessSendCount.TickCount.Get())
	server.stats.FailSendCountList = append(server.stats.FailSendCountList, server.FailSendCount.TickCount.Get())
	server.stats.UnknownSendCountList = append(server.stats.UnknownSendCountList, server.UnknownSendCount.TickCount.Get())
	server.stats.UnsendableSendCountList = append(server.stats.UnsendableSendCountList, server.UnsendableSendCount.TickCount.Get())
	server.stats.AllLinesCountList = append(server.stats.AllLinesCountList, server.AllLinesCount.TickCount.Get())
	// javascript resolution is ms .. not nanos
	server.stats.TicksList = append(server.stats.TicksList, int64(t_stamp/int64(time.Millisecond)))
	server.stats.GoRoutinesList = append(server.stats.GoRoutinesList, runtime.NumGoroutine())

	if uint(len(server.stats.ValidLineCountList)) > server.NumStats {
		server.stats.ValidLineCountList = server.stats.ValidLineCountList[1:server.NumStats]
		server.stats.InvalidLineCountList = server.stats.InvalidLineCountList[1:server.NumStats]
		server.stats.SuccessSendCountList = server.stats.SuccessSendCountList[1:server.NumStats]
		server.stats.FailSendCountList = server.stats.FailSendCountList[1:server.NumStats]
		server.stats.UnknownSendCountList = server.stats.UnknownSendCountList[1:server.NumStats]
		server.stats.UnsendableSendCountList = server.stats.UnsendableSendCountList[1:server.NumStats]
		server.stats.AllLinesCountList = server.stats.AllLinesCountList[1:server.NumStats]
		server.stats.TicksList = server.stats.TicksList[1:server.NumStats]
		server.stats.GoRoutinesList = server.stats.GoRoutinesList[1:server.NumStats]
	}
	server.stats.UpTimeSeconds = int64(elasped_sec)

	server.stats.ValidLineCountPerSec = server.ValidLineCount.TotalRate(elapsed)
	server.stats.InvalidLineCountPerSec = server.InvalidLineCount.TotalRate(elapsed)
	server.stats.SuccessSendCountPerSec = server.SuccessSendCount.TotalRate(elapsed)
	server.stats.UnsendableSendCountPerSec = server.UnsendableSendCount.TotalRate(elapsed)
	server.stats.UnknownSendCountPerSec = server.UnknownSendCount.TotalRate(elapsed)
	server.stats.AllLinesCountPerSec = server.AllLinesCount.TotalRate(elapsed)
	server.stats.Listening = server.ListenURL.String()
	server.stats.ServersUp = server.Hasher.Members()
	server.stats.ServersDown = server.Hasher.DroppedServers()
	server.stats.ServersChecks = server.Hasher.CheckingServers()

}

// dump some json data about the stats and server status
func (server *Server) StatsJsonString() string {
	resbytes, _ := json.Marshal(server.stats)
	return string(resbytes)
}

// a little function to log out some collected stats
func (server *Server) tickDisplay() {
	server.StatsTick()

	server.Logger.Printf("Server: ValidLineCount: %d", server.ValidLineCount.TotalCount)
	server.Logger.Printf("Server: InvalidLineCount: %d", server.InvalidLineCount.TotalCount)
	server.Logger.Printf("Server: SuccessSendCount: %d", server.SuccessSendCount.TotalCount)
	server.Logger.Printf("Server: FailSendCount: %d", server.FailSendCount.TotalCount)
	server.Logger.Printf("Server: UnsendableSendCount: %d", server.UnsendableSendCount.TotalCount)
	server.Logger.Printf("Server: UnknownSendCount: %d", server.UnknownSendCount.TotalCount)
	server.Logger.Printf("Server: AllLinesCount: %d", server.AllLinesCount.TotalCount)
	server.Logger.Printf("Server: GO Routines Running: %d", runtime.NumGoroutine())
	server.Logger.Printf("-------")
	server.Logger.Printf("Server Rate: Duration %ds", uint64(server.ticker/time.Second))
	server.Logger.Printf("Server Rate: ValidLineCount: %.2f/s", server.ValidLineCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: InvalidLineCount: %.2f/s", server.InvalidLineCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: SuccessSendCount: %.2f/s", server.SuccessSendCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: FailSendCount: %.2f/s", server.FailSendCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: UnsendableSendCount: %.2f/s", server.UnsendableSendCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: UnknownSendCount: %.2f/s", server.UnknownSendCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: AllLinesCount: %.2f/s", server.AllLinesCount.Rate(server.ticker))
	server.Logger.Printf("Server Send Method:: %s", server.SendingConnectionMethod)
	for idx, pool := range server.Outpool {
		server.Logger.Printf("Used NetPools [%s]: %d/%d", idx, pool.NumFree(), pool.MaxConnections)

	}

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
				server.Logger.Printf("Error Accecption Connection: %s", err)
				return
			}
			server.Logger.Printf("Accepted connection from %s", conn.RemoteAddr())

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
				server.Logger.Printf("Worker Queue Full %d", server.InWorkQueue.Get())
			}
		case client := <-done:
			client.Close()
		}
	}

}

// Fire up the http server for stats and healthchecks
// do this only if there is not a
func (server *Server) AddStatusHandlers() {

	stats := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		fmt.Fprintf(w, server.StatsJsonString())
	}
	status := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		if len(server.Hasher.Members()) <= 0 {
			http.Error(w, "all servers down", http.StatusServiceUnavailable)
			return
		} else {
			fmt.Fprintf(w, "ok")
		}
	}

	http.HandleFunc(fmt.Sprintf("/%s", server.Name), stats)
	http.HandleFunc(fmt.Sprintf("/%s/ops/status", server.Name), status)
	http.HandleFunc(fmt.Sprintf("/%s/ops/status/", server.Name), status)
	http.HandleFunc(fmt.Sprintf("/%s/status", server.Name), status)
	http.HandleFunc(fmt.Sprintf("/%s/stats/", server.Name), stats)
	http.HandleFunc(fmt.Sprintf("/%s/stats", server.Name), stats)

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
				server.Logger.Printf("Worker Queue Full %d", server.InWorkQueue.Get())
			}
		case client := <-done:
			client.Close()
		}
	}
}

func CreateServer(cfg *Config, hasher *ConstHasher) (*Server, error) {
	server, err := NewServer(cfg)

	if err != nil {
		panic(err)
	}
	server.Hasher = hasher
	server.Workers = DEFAULT_WORKERS
	if cfg.Workers > 0 {
		server.Workers = int64(cfg.Workers)
	}

	return server, err
}

func (server *Server) StartServer() {

	log.Printf("Using %d workers to process output", server.Workers)
	//fire up the send to workers
	done := make(chan Client)
	worker_queue := make(chan *SendOut)
	for w := int64(1); w <= server.Workers; w++ {
		go WorkerOutput(worker_queue, server.SendingConnectionMethod)
	}
	if server.UDPConn != nil {
		server.startUDPServer(server.Hasher, worker_queue, done)
	} else {
		server.startTCPServer(server.Hasher, worker_queue, done)
	}
}
