/*
   The server we listen for our data
*/

package main

import (
	"./netpool"
	"./runner"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DEFAULT_WORKERS                    = int64(500)
	DEFAULT_NUM_STATS                  = 100
	DEFAULT_SENDING_CONNECTIONS_METHOD = "bufferedpool"
	DEFAULT_RUNNER_TIMEOUT             = 5000 * time.Millisecond
	DEFAULT_WRITE_TIMEOUT              = 1000 * time.Millisecond
)

// the push meathod function type
type pushFunction func(*SendOut)

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

	var outsrv netpool.NetpoolInterface
	var ok bool

	// lock out Outpool map
	j.server.poolmu.Lock()
	if outsrv, ok = j.server.Outpool[j.outserver]; ok {
		ok = true
	} else {
		m_url, err := url.Parse(j.outserver)
		if err != nil {
			StatsdClient.Incr("failed.bad-url", 1)
			j.server.FailSendCount.Up(1)
			log.Printf("Error sending to backend Invalid URL %s", err)
			j.server.poolmu.Unlock()
			return
		}
		if len(j.server.Outpool) == 0 {
			j.server.Outpool = make(map[string]netpool.NetpoolInterface)

		}
		if j.server.SendingConnectionMethod == "bufferedpool" {
			outsrv = netpool.NewBufferedNetpool(m_url.Scheme, m_url.Host, j.server.BufferPoolSize)
		} else {
			outsrv = netpool.NewNetpool(m_url.Scheme, m_url.Host)
		}
		if j.server.NetPoolConnections > 0 {
			outsrv.SetMaxConnections(j.server.NetPoolConnections)
		}
		// populate it
		outsrv.InitPool()
		j.server.Outpool[j.outserver] = outsrv
	}
	// done with this locking ... the rest of the pool operations
	// are locked internally to the pooler
	j.server.poolmu.Unlock()

	netconn, err := outsrv.Open()
	defer outsrv.Close(netconn)

	if err != nil {
		StatsdClient.Incr("failed.bad-connection", 1)

		j.server.FailSendCount.Up(1)
		log.Printf("Error sending to backend %s", err)
		return
	}
	if netconn.Conn() != nil {
		// Conn.Write will raise a timeout error after 1 seconds
		netconn.SetWriteDeadline(time.Now().Add(j.server.WriteTimeout))
		//var wrote int
		to_send := []byte(j.param + "\n")
		_, err = netconn.Write(to_send)
		//log.Printf("SEND %s %s", wrote, err)
		if err != nil {
			StatsdClient.Incr("failed.connection-timeout", 1)
			j.server.FailSendCount.Up(1)
			outsrv.ResetConn(netconn)
			log.Printf("Error sending (writing) to backend: %s", err)
			return
		} else {
			j.server.SuccessSendCount.Up(1)
			StatsdClient.Incr("success.send", 1)
			StatsdClient.Incr("success.sent-bytes", int64(len(to_send)))
		}

	} else {
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

		conn.SetWriteDeadline(time.Now().Add(j.server.WriteTimeout))
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

/****************** SERVERS *********************/

//helper object for json'ing the basic stat data
type ServerStats struct {
	ValidLineCount       int64 `json:"valid_line_count"`
	WorkerValidLineCount int64 `json:"worker_line_count"`
	InvalidLineCount     int64 `json:"invalid_line_count"`
	SuccessSendCount     int64 `json:"success_send_count"`
	FailSendCount        int64 `json:"fail_send_count"`
	UnsendableSendCount  int64 `json:"unsendable_send_count"`
	UnknownSendCount     int64 `json:"unknown_send_count"`
	AllLinesCount        int64 `json:"all_lines_count"`

	CurrentValidLineCount       int64 `json:"current_valid_line_count"`
	CurrentWorkerValidLineCount int64 `json:"current_worker_line_count"`
	CurrentInvalidLineCount     int64 `json:"current_invalid_line_count"`
	CurrentSuccessSendCount     int64 `json:"current_success_send_count"`
	CurrentFailSendCount        int64 `json:"current_fail_send_count"`
	CurrentUnsendableSendCount  int64 `json:"current_unsendable_send_count"`
	CurrentUnknownSendCount     int64 `json:"current_unknown_send_count"`
	CurrentAllLinesCount        int64 `json:"current_all_lines_count"`

	ValidLineCountList       []int64 `json:"valid_line_count_list"`
	WorkerValidLineCountList []int64 `json:"worker_line_count_list"`
	InvalidLineCountList     []int64 `json:"invalid_line_count_list"`
	SuccessSendCountList     []int64 `json:"success_send_count_list"`
	FailSendCountList        []int64 `json:"fail_send_count_list"`
	UnsendableSendCountList  []int64 `json:"unsendable_send_count_list"`
	UnknownSendCountList     []int64 `json:"unknown_send_count_list"`
	AllLinesCountList        []int64 `json:"all_lines_count_list"`
	GoRoutinesList           []int   `json:"go_routines_list"`
	TicksList                []int64 `json:"ticks_list"`

	GoRoutines                 int      `json:"go_routines"`
	UpTimeSeconds              int64    `json:"uptime_sec"`
	ValidLineCountPerSec       float32  `json:"valid_line_count_persec"`
	WorkerValidLineCountPerSec float32  `json:"worker_line_count_persec"`
	InvalidLineCountPerSec     float32  `json:"invalid_line_count_persec"`
	SuccessSendCountPerSec     float32  `json:"success_send_count_persec"`
	UnsendableSendCountPerSec  float32  `json:"unsendable_count_persec"`
	UnknownSendCountPerSec     float32  `json:"unknown_send_count_persec"`
	AllLinesCountPerSec        float32  `json:"all_lines_count_persec"`
	Listening                  string   `json:"listening"`
	ServersUp                  []string `json:"servers_up"`
	ServersDown                []string `json:"servers_down"`
	ServersChecks              []string `json:"servers_checking"`
}

// helper object for the json info about a single "key"
// basically to see "what server" a key will end up going to
type ServerHashCheck struct {
	ToServers []string `json:"to_servers"`
	HashKey   string   `json:"hash_key"`
	HashValue []uint32 `json:"hash_value"`
}

// a server set of stats
type Server struct {
	Name      string
	ListenURL *url.URL

	ValidLineCount       StatCount
	WorkerValidLineCount StatCount
	InvalidLineCount     StatCount
	SuccessSendCount     StatCount
	FailSendCount        StatCount
	UnsendableSendCount  StatCount
	UnknownSendCount     StatCount
	AllLinesCount        StatCount
	NumStats             uint

	// our bound connection if TCP
	Connection       net.Listener
	UDPConn          *net.UDPConn
	ClientBufferSize int //for UDP read buffers

	// timeouts for tuning
	WriteTimeout  time.Duration // time out when sending lines
	RunnerTimeout time.Duration // timeout for work queue items

	//Hasher objects (can have multiple for replication of data)
	Hashers []*ConstHasher

	// we can use a "pool" of connections, or single connections per line
	// performance will be depending on the system and work load tcp vs udp, etc
	// "bufferedpool" or "pool" or "single"
	// default is pool
	SendingConnectionMethod string
	//number of connections in the NetPool
	NetPoolConnections int
	//if using the buffer pool, this is the buffer size
	BufferPoolSize int

	//number of replicas to fire data to (i.e. dupes)
	Replicas int

	//pool the connections to the outgoing servers
	poolmu  *sync.Mutex //when we make a new pool need to lock the hash below
	Outpool map[string]netpool.NetpoolInterface

	ticker time.Duration

	//workers and ques sizes
	WorkQueue   chan *SendOut
	WorkerHold  chan int64
	InWorkQueue AtomicInt
	Workers     int64

	//the Runner type to determine the keys to hash on
	RunnerTypeString string
	RunnerConfig     map[string]interface{}
	LineProcessor    runner.Runner

	//the push function
	PushFunction pushFunction

	//uptime
	StartTime time.Time

	stats ServerStats

	Logger *log.Logger
}

// set the "push" function we are using "pool" or "single"
func (server *Server) SetPushMethod() pushFunction {
	switch server.SendingConnectionMethod {
	case "single":
		server.PushFunction = singleWorker
	default:
		server.PushFunction = poolWorker
	}
	return server.PushFunction
}

func (server *Server) SetLineProcessor() (runner.Runner, error) {
	msg_type := server.RunnerTypeString

	var rner runner.Runner
	var err error = nil
	switch {
	case msg_type == "statsd":
		rner, err = runner.NewStatsdRunner(server.RunnerConfig)
	case msg_type == "graphite":
		rner, err = runner.NewGraphiteRunner(server.RunnerConfig)
	case msg_type == "regex":
		rner, err = runner.NewRegExRunner(server.RunnerConfig)
	default:
		return nil, fmt.Errorf("Failed to configure Runner, aborting")
	}
	server.LineProcessor = rner
	return server.LineProcessor, err

}

//spins up the queue of go routines to handle outgoing
func (server *Server) WorkerOutput(jobs <-chan *SendOut) {
	for j := range jobs {
		server.PushFunction(j)
	}
}

func (server *Server) RunRunner(key string, line string, out chan string) {
	//direct timer to void leaks (i.e. NOT time.After(...))

	timer := time.NewTimer(server.RunnerTimeout)
	defer StatsdNanoTimeFunc(fmt.Sprintf("factory.runner.process-time-ns"), time.Now())

	select {
	case out <- server.PushLine(key, line) + "\n":
		timer.Stop()
		server.WorkerHold <- -1
	case <-timer.C:
		timer.Stop()
		server.WorkerHold <- -1
		StatsdClient.Incr("failed.runner-timeout", 1)
		server.FailSendCount.Up(1)
		server.Logger.Printf("Timeout %d, %s", len(server.WorkQueue), key)
	}
}

//return the ServerHashCheck for a given key (more a utility debugger thing for
// the stats http server)
func (server *Server) HasherCheck(key string) ServerHashCheck {

	var out_check ServerHashCheck
	out_check.HashKey = key

	for _, hasher := range server.Hashers {
		// may have replicas inside the pool too that we need to deal with
		servs, err := hasher.GetN(key, server.Replicas)
		if err == nil {
			for _, useme := range servs {
				out_check.ToServers = append(out_check.ToServers, useme)
				out_check.HashValue = append(out_check.HashValue, hasher.Hasher.GetHasherValue(key))
			}
		}
	}
	return out_check
}

// the "main" hash chooser for a give line, the attaches it to a sender queue
func (server *Server) PushLine(key string, line string) string {

	//replicate the data across our Lists
	out_str := ""
	for idx, hasher := range server.Hashers {

		// may have replicas inside the pool too that we need to deal with
		servs, err := hasher.GetN(key, server.Replicas)
		if err == nil {
			for nidx, useme := range servs {
				// just log the valid lines "once" total ends stats are WorkerValidLineCount
				if idx == 0 && nidx == 0 {
					server.ValidLineCount.Up(1)
					StatsdClient.Incr("success.valid-lines", 1)
				}
				StatsdClient.Incr("success.valid-lines-sent-to-workers", 1)
				server.WorkerValidLineCount.Up(1)

				sendOut := &SendOut{
					outserver: useme,
					server:    server,
					param:     line,
				}
				server.WorkQueue <- sendOut
				server.WorkerHold <- 1
			}
			out_str += "ok"
		} else {

			StatsdClient.Incr("failed.invalid-hash-server", 1)
			server.UnsendableSendCount.Up(1)
			out_str += "failed"
		}
	}
	return out_str

}

func (server *Server) ResetTickers() {
	server.ValidLineCount.ResetTick()
	server.WorkerValidLineCount.ResetTick()
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

	serv.poolmu = new(sync.Mutex)

	serv.NumStats = DEFAULT_NUM_STATS
	if cfg.HealthServerPoints > 0 {
		serv.NumStats = cfg.HealthServerPoints
	}

	serv.WriteTimeout = DEFAULT_WRITE_TIMEOUT
	if cfg.WriteTimeout != 0 {
		serv.WriteTimeout = cfg.WriteTimeout
	}

	serv.RunnerTimeout = DEFAULT_RUNNER_TIMEOUT
	if cfg.RunnerTimeout != 0 {
		serv.RunnerTimeout = cfg.RunnerTimeout
	}

	serv.NetPoolConnections = cfg.MaxPoolConnections
	serv.BufferPoolSize = cfg.MaxPoolBufferSize
	serv.SendingConnectionMethod = DEFAULT_SENDING_CONNECTIONS_METHOD

	serv.ClientBufferSize = cfg.ReadBufferSize

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
	return serv, nil

}

func (server *Server) StatsTick() {
	elapsed := time.Since(server.StartTime)
	elasped_sec := float64(elapsed) / float64(time.Second)
	t_stamp := time.Now().UnixNano()

	server.stats.ValidLineCount = server.ValidLineCount.TotalCount.Get()
	server.stats.WorkerValidLineCount = server.WorkerValidLineCount.TotalCount.Get()
	server.stats.InvalidLineCount = server.InvalidLineCount.TotalCount.Get()
	server.stats.SuccessSendCount = server.SuccessSendCount.TotalCount.Get()
	server.stats.FailSendCount = server.FailSendCount.TotalCount.Get()
	server.stats.UnsendableSendCount = server.UnsendableSendCount.TotalCount.Get()
	server.stats.UnknownSendCount = server.UnknownSendCount.TotalCount.Get()
	server.stats.AllLinesCount = server.AllLinesCount.TotalCount.Get()

	server.stats.CurrentValidLineCount = server.ValidLineCount.TickCount.Get()
	server.stats.CurrentWorkerValidLineCount = server.WorkerValidLineCount.TickCount.Get()
	server.stats.CurrentInvalidLineCount = server.InvalidLineCount.TickCount.Get()
	server.stats.CurrentSuccessSendCount = server.SuccessSendCount.TickCount.Get()
	server.stats.CurrentFailSendCount = server.FailSendCount.TickCount.Get()
	server.stats.CurrentUnknownSendCount = server.UnknownSendCount.TickCount.Get()
	server.stats.CurrentAllLinesCount = server.AllLinesCount.TickCount.Get()

	server.stats.ValidLineCountList = append(server.stats.ValidLineCountList, server.ValidLineCount.TickCount.Get())
	server.stats.WorkerValidLineCountList = append(server.stats.WorkerValidLineCountList, server.WorkerValidLineCount.TickCount.Get())
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
		server.stats.WorkerValidLineCountList = server.stats.WorkerValidLineCountList[1:server.NumStats]
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
	server.stats.WorkerValidLineCountPerSec = server.WorkerValidLineCount.TotalRate(elapsed)
	server.stats.InvalidLineCountPerSec = server.InvalidLineCount.TotalRate(elapsed)
	server.stats.SuccessSendCountPerSec = server.SuccessSendCount.TotalRate(elapsed)
	server.stats.UnsendableSendCountPerSec = server.UnsendableSendCount.TotalRate(elapsed)
	server.stats.UnknownSendCountPerSec = server.UnknownSendCount.TotalRate(elapsed)
	server.stats.AllLinesCountPerSec = server.AllLinesCount.TotalRate(elapsed)
	server.stats.Listening = server.ListenURL.String()
	//XXX TODO FIX ME
	for idx, hasher := range server.Hashers {
		if idx == 0 {
			server.stats.ServersUp = hasher.Members()
			server.stats.ServersDown = hasher.DroppedServers()
			server.stats.ServersChecks = hasher.CheckingServers()
		} else {
			server.stats.ServersUp = append(server.stats.ServersUp, hasher.Members()...)
			server.stats.ServersDown = append(server.stats.ServersDown, hasher.DroppedServers()...)
			server.stats.ServersChecks = append(server.stats.ServersChecks, hasher.CheckingServers()...)
		}
		//tick the cacher stats
		length, size, capacity, _ := hasher.Cache.Stats()
		StatsdClient.GaugeAbsolute("lrucache.length", int64(length))
		StatsdClient.GaugeAbsolute("lrucache.size", int64(size))
		StatsdClient.GaugeAbsolute("lrucache.capacity", int64(capacity))

	}

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
	server.Logger.Printf("Server: WorkerValidLineCount: %d", server.WorkerValidLineCount.TotalCount)
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
	server.Logger.Printf("Server Rate: WorkerLineCount: %.2f/s", server.WorkerValidLineCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: InvalidLineCount: %.2f/s", server.InvalidLineCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: SuccessSendCount: %.2f/s", server.SuccessSendCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: FailSendCount: %.2f/s", server.FailSendCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: UnsendableSendCount: %.2f/s", server.UnsendableSendCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: UnknownSendCount: %.2f/s", server.UnknownSendCount.Rate(server.ticker))
	server.Logger.Printf("Server Rate: AllLinesCount: %.2f/s", server.AllLinesCount.Rate(server.ticker))
	server.Logger.Printf("Server Send Method:: %s", server.SendingConnectionMethod)
	for idx, pool := range server.Outpool {
		server.Logger.Printf("Free Connections in Pools [%s]: %d/%d", idx, pool.NumFree(), pool.GetMaxConnections())
	}

	server.ResetTickers()
	runtime.GC()
	time.Sleep(server.ticker)

	go server.tickDisplay()
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
		have_s := 0
		for _, hasher := range server.Hashers {
			have_s += len(hasher.Members())
		}
		if have_s <= 0 {
			http.Error(w, "all servers down", http.StatusServiceUnavailable)
			return
		} else {
			fmt.Fprintf(w, "ok")
		}
	}

	// add a new hasher node to a server dynamically
	addnode := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		r.ParseForm()
		server_str := strings.TrimSpace(r.Form.Get("server"))
		if len(server_str) == 0 {
			http.Error(w, "Invalid server name", http.StatusBadRequest)
			return
		}
		server_url, err := url.Parse(server_str)
		if err != nil {
			http.Error(w, "Not a valid server URL", http.StatusBadRequest)
			return
		}
		chk_server_str := strings.TrimSpace(r.Form.Get("check_server"))
		if len(chk_server_str) == 0 {
			chk_server_str = server_str
		}
		chk_server_url, err := url.Parse(chk_server_str)
		if err != nil {
			http.Error(w, "Not a valid Check server URL", http.StatusBadRequest)
			return
		}
		if chk_server_url.Scheme != "tcp" && chk_server_url.Scheme != "http" {
			http.Error(w, "Check server can only be TCP or HTTP", http.StatusBadRequest)
			return
		}
		// since we can have replicas .. need an index to add it to
		replica_str := r.Form.Get("replica")
		if len(replica_str) == 0 {
			replica_str = "0"
		}
		replica_int, err := strconv.Atoi(replica_str)
		if err != nil {
			http.Error(w, "Replica index is not an int", http.StatusBadRequest)
			return
		}
		if replica_int > len(server.Hashers)-1 {
			http.Error(w, "Replica index Too large", http.StatusBadRequest)
			return
		}
		if replica_int < 0 {
			http.Error(w, "Replica index Too small", http.StatusBadRequest)
			return
		}

		//we can also accept a hash key for that server
		hash_key_str := r.Form.Get("hashkey")
		if len(hash_key_str) == 0 {
			hash_key_str = server_str
		}

		server.Hashers[replica_int].AddServer(server_url, chk_server_url, hash_key_str)
		fmt.Fprintf(w, "Server "+server_str+" Added")
	}

	// add a new hasher node to a server dynamically
	purgenode := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		r.ParseForm()
		server_str := strings.TrimSpace(r.Form.Get("server"))
		if len(server_str) == 0 {
			http.Error(w, "Invalid server name", http.StatusBadRequest)
			return
		}
		server_url, err := url.Parse(server_str)
		if err != nil {
			http.Error(w, "Not a valid server URL", http.StatusBadRequest)
			return
		}

		// since we can have replicas .. need an index to add it to
		replica_str := r.Form.Get("replica")
		if len(replica_str) == 0 {
			replica_str = "0"
		}
		replica_int, err := strconv.Atoi(replica_str)
		if err != nil {
			http.Error(w, "Replica index is not an int", http.StatusBadRequest)
			return
		}
		if replica_int > len(server.Hashers)-1 {
			http.Error(w, "Replica index Too large", http.StatusBadRequest)
			return
		}
		if replica_int < 0 {
			http.Error(w, "Replica index Too small", http.StatusBadRequest)
			return
		}

		server.Hashers[replica_int].PurgeServer(server_url)

		// we need to CLOSE and thing in the Outpool
		if serv, ok := server.Outpool[server_str]; ok {
			serv.DestroyAll()
			delete(server.Outpool, server_str)
		}

		fmt.Fprintf(w, "Server "+server_str+" Purged")
	}

	//stats and status
	http.HandleFunc(fmt.Sprintf("/%s", server.Name), stats)
	http.HandleFunc(fmt.Sprintf("/%s/ops/status", server.Name), status)
	http.HandleFunc(fmt.Sprintf("/%s/ops/status/", server.Name), status)
	http.HandleFunc(fmt.Sprintf("/%s/status", server.Name), status)
	http.HandleFunc(fmt.Sprintf("/%s/stats/", server.Name), stats)
	http.HandleFunc(fmt.Sprintf("/%s/stats", server.Name), stats)

	//admin like functions
	http.HandleFunc(fmt.Sprintf("/%s/addserver", server.Name), addnode)
	http.HandleFunc(fmt.Sprintf("/%s/purgeserver", server.Name), purgenode)

}

// accept incoming TCP connections and push them into the
// a connection channel
func (server *Server) Accepter() (<-chan *net.TCPConn, error) {

	conns := make(chan *net.TCPConn, server.Workers)
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

			conns <- conn.(*net.TCPConn)
		}
	}()
	return conns, nil
}

func (server *Server) startTCPServer(hashers *[]*ConstHasher, worker_queue chan *SendOut, done chan Client) {

	//for TCPclients, since we basically create a client on each connect each time (unlike the UDP case)
	// and we want only one processing Q, set up this "queue" here
	// let this queue get big as the workers for TCP may been much more buffering due to bursting
	input_queue := make(chan string, server.Workers)
	defer close(input_queue)
	out_queue := make(chan string, server.Workers)
	defer close(out_queue)

	run := func() {
		for line := range input_queue {
			defer StatsdNanoTimeFunc(fmt.Sprintf("factory.%s.process-time-ns", server.LineProcessor.Name), time.Now())

			key, line, err := server.LineProcessor.ProcessLine(strings.Trim(line, "\n\t "))
			if err == nil {
				go server.RunRunner(key, line, out_queue)
			}
			server.AllLinesCount.Up(1)
			StatsdClient.Incr("incoming.tcp.lines", 1)
		}
	}
	//fire up the workers
	for w := int64(1); w <= server.Workers; w++ {
		go run()
	}

	accepts_queue, err := server.Accepter()
	if err != nil {
		panic(err)
	}

	for {
		select {
		case conn, ok := <-accepts_queue:
			if !ok {
				return
			}
			client := NewTCPClient(server, hashers, conn, worker_queue, done, input_queue, out_queue)
			client.SetBufferSize(server.ClientBufferSize)
			go client.handleRequest()
			go client.handleSend()
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			ct := server.InWorkQueue.Get()
			if ct >= server.Workers {
				StatsdClient.Incr("worker.queue.isfull", 1)
				//server.Logger.Printf("Worker Queue Full %d", ct)
			}
			StatsdClient.GaugeAvg("worker.queue.length", ct)

		case client := <-done:
			client.Close() //this will close the connection too
			client = nil
		}
	}

}

// different mechanism for UDP servers
func (server *Server) startUDPServer(hashers *[]*ConstHasher, worker_queue chan *SendOut, done chan Client) {

	//just need on "client" here as we simply just pull from the socket
	client := NewUDPClient(server, hashers, server.UDPConn, worker_queue, done)
	client.SetBufferSize(server.ClientBufferSize)
	go client.handleRequest()
	go client.handleSend()
	for {
		select {
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			ct := server.InWorkQueue.Get()
			if ct >= server.Workers {
				StatsdClient.Incr("worker.queue.isfull", 1)
				//server.Logger.Printf("Worker Queue Full %d", ct)
			}
			StatsdClient.GaugeAvg("worker.queue.length", ct)

		case client := <-done:
			client.Close()
		}
	}
}

func CreateServer(cfg *Config, hashers []*ConstHasher) (*Server, error) {
	server, err := NewServer(cfg)

	if err != nil {
		panic(err)
	}
	server.Hashers = hashers
	server.Workers = DEFAULT_WORKERS
	if cfg.Workers > 0 {
		server.Workers = int64(cfg.Workers)
	}
	//start tickin'
	go server.tickDisplay()

	return server, err
}

func (server *Server) StartServer() {

	log.Printf("Using %d workers to process output", server.Workers)
	//fire up the send to workers
	done := make(chan Client)

	//the queue is only as big as the workers
	worker_queue := make(chan *SendOut, server.Workers)

	//set the push method (the WorkerOutput depends on it)
	server.SetPushMethod()

	for w := int64(1); w <= server.Workers; w++ {
		go server.WorkerOutput(worker_queue)
	}

	//get our line proessor in order
	_, err := server.SetLineProcessor()
	if err != nil {
		panic(err)
	}

	//the mighty queue for the main runner to push to
	server.WorkQueue = worker_queue

	if server.UDPConn != nil {
		server.startUDPServer(&server.Hashers, worker_queue, done)
	} else {
		server.startTCPServer(&server.Hashers, worker_queue, done)
	}
}
