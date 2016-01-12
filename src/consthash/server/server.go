/*
   Pretty much the main work hourse

   ties the lot together

   Listener -> [PreReg] -> Backend -> Hasher -> [Replicator] -> NetPool -> Out

   Optional items are in the `[]`

   If we are using accumulators to basically "be" a graphite aggregator or statsd aggregator the flow is a bit
   Different

   Listener
   	-> PreReg (for rejection processing)
   	-> Accumulator
   		-> `Flush`
   		-> PreReg (again as the accumulator can generate "more" keys, statsd timers for instance)
   			-> Backend
   			-> Hasher
   				-> [Replicator]
   				-> Netpool
   				-> Out

*/

package consthash

import (
	"consthash/server/netpool"
	"consthash/server/prereg"
	"consthash/server/splitter"
	"consthash/server/stats"
	"encoding/json"
	"fmt"
	logging "github.com/op/go-logging"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	DEFAULT_WORKERS                    = int64(500)
	DEFAULT_NUM_STATS                  = 100
	DEFAULT_SENDING_CONNECTIONS_METHOD = "bufferedpool"
	DEFAULT_SPLITTER_TIMEOUT           = 5000 * time.Millisecond
	DEFAULT_WRITE_TIMEOUT              = 1000 * time.Millisecond
	DEFAULT_BACKPRESSURE_SLEEP         = 1000 * time.Millisecond
	DEFAULT_READ_BUFFER_SIZE           = 4096
)

// the push method function type
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
	defer stats.StatsdNanoTimeFunc("worker.process-time-ns", time.Now())

	var outsrv netpool.NetpoolInterface
	var ok bool

	// lock out Outpool map
	j.server.poolmu.Lock()
	if outsrv, ok = j.server.Outpool[j.outserver]; ok {
		ok = true
	} else {
		m_url, err := url.Parse(j.outserver)
		if err != nil {
			stats.StatsdClient.Incr("failed.bad-url", 1)
			j.server.FailSendCount.Up(1)
			log.Error("Error sending to backend Invalid URL %s", err)
			j.server.poolmu.Unlock()
			return
		}
		if len(j.server.Outpool) == 0 {
			j.server.Outpool = make(map[string]netpool.NetpoolInterface)

		}
		if j.server.SendingConnectionMethod == "bufferedpool" {
			outsrv = netpool.NewBufferedNetpool(m_url.Scheme, m_url.Host+m_url.Path, j.server.WriteBufferPoolSize)
		} else {
			outsrv = netpool.NewNetpool(m_url.Scheme, m_url.Host+m_url.Path)
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
		stats.StatsdClient.Incr("failed.bad-connection", 1)

		j.server.FailSendCount.Up(1)
		j.server.log.Error("Error sending to backend %s", err)
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
			stats.StatsdClient.Incr("failed.connection-timeout", 1)
			j.server.FailSendCount.Up(1)
			outsrv.ResetConn(netconn)
			j.server.log.Error("Error sending (writing) to backend: %s", err)
			return
		} else {
			j.server.SuccessSendCount.Up(1)
			stats.StatsdClient.Incr("success.send", 1)
			stats.StatsdClient.Incr("success.sent-bytes", int64(len(to_send)))
		}

	} else {
		stats.StatsdClient.Incr("failed.aborted-connection", 1)
		j.server.FailSendCount.Up(1)
		j.server.log.Error("Error sending (writing connection gone) to backend: %s", err)
		return
	}
}

//this is for using a simple tcp connection per stat we send out
//one can quickly run out of sockets if this is used under high load
func singleWorker(j *SendOut) {
	defer stats.StatsdNanoTimeFunc("worker.process-time-ns", time.Now())

	m_url, err := url.Parse(j.outserver)
	if err != nil {
		stats.StatsdClient.Incr("failed.bad-url", 1)
		j.server.FailSendCount.Up(1)
		j.server.log.Error("Error sending to backend Invalid URL %s", err)
		return
	}
	conn, err := net.DialTimeout(m_url.Scheme, m_url.Host+m_url.Path, 5*time.Second)
	if conn != nil {

		conn.SetWriteDeadline(time.Now().Add(j.server.WriteTimeout))
		//send it and close it
		to_send := []byte(j.param + "\n")
		_, err = conn.Write(to_send)
		conn.Close()
		conn = nil
		if err != nil {
			stats.StatsdClient.Incr("failed.bad-connection", 1)
			j.server.FailSendCount.Up(1)
			j.server.log.Error("Error sending (writing) to backend: %s", err)
			return
		}
		stats.StatsdClient.Incr("success.sent", 1)
		stats.StatsdClient.Incr("success.sent-bytes", int64(len(to_send)))
		j.server.SuccessSendCount.Up(1)
	} else {
		j.server.log.Error("Error sending (connection) to backend: %s", err)
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
	RedirectedLinesCount int64 `json:"redirected_lines_count"`
	RejectedLinesCount   int64 `json:"rejected_lines_count"`

	CurrentValidLineCount       int64 `json:"current_valid_line_count"`
	CurrentWorkerValidLineCount int64 `json:"current_worker_line_count"`
	CurrentInvalidLineCount     int64 `json:"current_invalid_line_count"`
	CurrentSuccessSendCount     int64 `json:"current_success_send_count"`
	CurrentFailSendCount        int64 `json:"current_fail_send_count"`
	CurrentUnsendableSendCount  int64 `json:"current_unsendable_send_count"`
	CurrentUnknownSendCount     int64 `json:"current_unknown_send_count"`
	CurrentAllLinesCount        int64 `json:"current_all_lines_count"`
	CurrentRejectedLinesCount   int64 `json:"current_rejected_lines_count"`
	CurrentRedirectedLinesCount int64 `json:"current_redirected_lines_count"`

	ValidLineCountList       []int64 `json:"valid_line_count_list"`
	WorkerValidLineCountList []int64 `json:"worker_line_count_list"`
	InvalidLineCountList     []int64 `json:"invalid_line_count_list"`
	SuccessSendCountList     []int64 `json:"success_send_count_list"`
	FailSendCountList        []int64 `json:"fail_send_count_list"`
	UnsendableSendCountList  []int64 `json:"unsendable_send_count_list"`
	UnknownSendCountList     []int64 `json:"unknown_send_count_list"`
	AllLinesCountList        []int64 `json:"all_lines_count_list"`
	RedirectedCountList      []int64 `json:"redirected_lines_count_list"`
	RejectedCountList        []int64 `json:"rejected_lines_count_list"`
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
	RedirectedLinesCountPerSec float32  `json:"redirected_lines_count_persec"`
	RejectedLinesCountPerSec   float32  `json:"rejected_lines_count_persec"`
	Listening                  string   `json:"listening"`
	ServersUp                  []string `json:"servers_up"`
	ServersDown                []string `json:"servers_down"`
	ServersChecks              []string `json:"servers_checking"`

	CurrentReadBufferSize int64 `json:"current_read_buffer_size"`
	MaxReadBufferSize     int64 `json:"max_read_buffer_size"`
	InputQueueSize        int   `json:"input_queue_size"`
	WorkQueueSize         int   `json:"work_queue_size"`
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

	ValidLineCount       stats.StatCount
	WorkerValidLineCount stats.StatCount
	InvalidLineCount     stats.StatCount
	SuccessSendCount     stats.StatCount
	FailSendCount        stats.StatCount
	UnsendableSendCount  stats.StatCount
	UnknownSendCount     stats.StatCount
	AllLinesCount        stats.StatCount
	RejectedLinesCount   stats.StatCount
	RedirectedLinesCount stats.StatCount
	NumStats             uint

	// our bound connection if TCP or UnixSocket
	Connection net.Listener
	UDPConn    *net.UDPConn

	//if our "buffered" bits exceded this, we're basically out of ram
	// so we "pause" until we can do something
	ClientReadBufferSize int64           //for net read buffers
	MaxReadBufferSize    int64           // the biggest we can make this buffer before "failing"
	CurrentReadBufferRam stats.AtomicInt // the amount of buffer we're on

	// timeouts for tuning
	WriteTimeout    time.Duration // time out when sending lines
	SplitterTimeout time.Duration // timeout for work queue items

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
	WriteBufferPoolSize int

	//number of replicas to fire data to (i.e. dupes)
	Replicas int

	//pool the connections to the outgoing servers
	poolmu  *sync.Mutex //when we make a new pool need to lock the hash below
	Outpool map[string]netpool.NetpoolInterface

	ticker time.Duration

	//input queue for incoming lines
	InputQueue chan splitter.SplitItem

	//workers and ques sizes
	WorkQueue   chan *SendOut
	WorkerHold  chan int64
	InWorkQueue stats.AtomicInt
	Workers     int64

	//the Splitter type to determine the keys to hash on
	SplitterTypeString string
	SplitterConfig     map[string]interface{}
	SplitterProcessor  splitter.Splitter

	// Prereg filters to push to other backends or drop
	PreRegFilter *prereg.PreReg

	//allow us to push backpressure on TCP sockets if we need to
	back_pressure       chan bool
	_back_pressure_on   bool // makes sure we dont' fire a billion things in the channel
	back_pressure_sleep time.Duration
	back_pressure_lock  sync.Mutex

	//trap some signals yo
	ShutDown chan bool

	//the push function (polling, direct, etc)
	PushFunction pushFunction

	//uptime
	StartTime time.Time

	stats ServerStats

	log *logging.Logger
}

func (server *Server) AddToCurrentTotalBufferSize(length int64) int64 {
	return server.CurrentReadBufferRam.Add(length)
}

func (server *Server) GetStats() (stats *ServerStats) {
	return &server.stats
}

func (server *Server) TrapExit() {
	//trap kills to flush queues and close connections
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func(ins *Server) {
		s := <-sc
		ins.log.Notice("Caught %s: Closing out before quit ", s)

		// need to clen up the socket here otherwise it may not get cleaned
		if ins.ListenURL != nil && ins.ListenURL.Scheme == "unix" {
			os.Remove("/" + ins.ListenURL.Host + ins.ListenURL.Path)
		}

		//shut this guy down too
		if ins.PreRegFilter.Accumulator != nil {
			ins.PreRegFilter.Accumulator.Stop()
		}

		ins.ShutDown <- true

		signal.Stop(sc)
		close(sc)

		// re-raise it
		process, _ := os.FindProcess(os.Getpid())
		process.Signal(s)
	}(server)
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

func (server *Server) SetSplitterProcessor() (splitter.Splitter, error) {
	gots, err := splitter.NewSplitterItem(server.SplitterTypeString, server.SplitterConfig)
	server.SplitterProcessor = gots
	return server.SplitterProcessor, err

}

func (server *Server) BackPressure() {
	if server._back_pressure_on {
		return
	}
	server.back_pressure <- server.NeedBackPressure()
	server._back_pressure_on = true
}

func (server *Server) NeedBackPressure() bool {
	return server.CurrentReadBufferRam.Get() > server.MaxReadBufferSize || len(server.WorkQueue) >= (int)(server.Workers)
}

//spins up the queue of go routines to handle outgoing
func (server *Server) WorkerOutput() {
	for j := range server.WorkQueue {
		server.PushFunction(j)
	}
}

func (server *Server) SendtoOutputWorkers(spl splitter.SplitItem, out chan splitter.SplitItem) {
	//direct timer to void leaks (i.e. NOT time.After(...))

	timer := time.NewTimer(server.SplitterTimeout)
	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("factory.splitter.process-time-ns"), time.Now())
	defer func() { server.WorkerHold <- -1 }()

	select {
	case out <- server.PushLineToBackend(spl):

		timer.Stop()

	case <-timer.C:
		timer.Stop()
		stats.StatsdClient.Incr("failed.splitter-timeout", 1)
		server.FailSendCount.Up(1)
		server.log.Warning("Timeout %d, %s", len(server.WorkQueue), spl.Key())
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
func (server *Server) PushLineToBackend(spl splitter.SplitItem) splitter.SplitItem {

	//replicate the data across our Lists
	out_str := ""
	for idx, hasher := range server.Hashers {

		// may have replicas inside the pool too that we need to deal with
		servs, err := hasher.GetN(spl.Key(), server.Replicas)
		if err == nil {
			for nidx, useme := range servs {
				// just log the valid lines "once" total ends stats are WorkerValidLineCount
				if idx == 0 && nidx == 0 {
					server.ValidLineCount.Up(1)
					stats.StatsdClient.Incr("success.valid-lines", 1)
				}
				stats.StatsdClient.Incr("success.valid-lines-sent-to-workers", 1)
				server.WorkerValidLineCount.Up(1)

				sendOut := &SendOut{
					outserver: useme,
					server:    server,
					param:     spl.Line(),
				}

				server.WorkQueue <- sendOut
				server.WorkerHold <- 1

			}
			out_str += "ok"
		} else {

			stats.StatsdClient.Incr("failed.invalid-hash-server", 1)
			server.UnsendableSendCount.Up(1)
			out_str += "failed"
		}
	}
	return spl

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
	server.RedirectedLinesCount.ResetTick()
	server.RejectedLinesCount.ResetTick()
}

func NewServer(cfg *Config) (server *Server, err error) {

	serv := new(Server)
	serv.Name = cfg.Name
	serv.ListenURL = cfg.ListenURL
	serv.StartTime = time.Now()
	serv.WorkerHold = make(chan int64)

	serv.log = logging.MustGetLogger(fmt.Sprintf("server.%s", serv.Name))

	//log.New(os.Stdout, fmt.Sprintf("[Server: %s] ", serv.Name), log.Ldate|log.Ltime)
	if cfg.ListenURL != nil {
		serv.log.Notice("Binding server to %s", serv.ListenURL.String())
	} else {
		serv.log.Notice("Using as a Backend Only to %s", serv.Name)
	}

	//find the runner types
	serv.SplitterTypeString = cfg.MsgType
	serv.SplitterConfig = cfg.MsgConfig
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

	serv.SplitterTimeout = DEFAULT_SPLITTER_TIMEOUT
	if cfg.SplitterTimeout != 0 {
		serv.SplitterTimeout = cfg.SplitterTimeout
	}

	serv.NetPoolConnections = cfg.MaxPoolConnections
	serv.WriteBufferPoolSize = cfg.MaxWritePoolBufferSize
	serv.SendingConnectionMethod = DEFAULT_SENDING_CONNECTIONS_METHOD

	serv.ClientReadBufferSize = cfg.ClientReadBufferSize
	if serv.ClientReadBufferSize <= 0 {
		serv.ClientReadBufferSize = DEFAULT_READ_BUFFER_SIZE
	}
	serv.MaxReadBufferSize = cfg.MaxReadBufferSize

	if serv.MaxReadBufferSize <= 0 {
		serv.MaxReadBufferSize = 1000 * serv.ClientReadBufferSize // reasonable default for max buff
	}

	serv.back_pressure_sleep = DEFAULT_BACKPRESSURE_SLEEP

	if len(cfg.SendingConnectionMethod) > 0 {
		serv.SendingConnectionMethod = cfg.SendingConnectionMethod
	}

	// if there's a PreReg assign to the server
	if cfg.PreRegFilter != nil {
		// the config should have checked this already, but just in case
		if cfg.ListenStr == "backend_only" {
			return nil, fmt.Errorf("Backend Only cannot have PreReg filters")
		}
		serv.PreRegFilter = cfg.PreRegFilter
	}

	if serv.ListenURL == nil {
		// just a backend, no connections

	} else if cfg.ListenURL.Scheme == "udp" {
		udp_addr, err := net.ResolveUDPAddr(cfg.ListenURL.Scheme, cfg.ListenURL.Host)
		if err != nil {
			return nil, fmt.Errorf("Error binding: %s", err)
		}
		conn, err := net.ListenUDP(cfg.ListenURL.Scheme, udp_addr)
		if err != nil {
			return nil, fmt.Errorf("Error binding: %s", err)
		}
		serv.UDPConn = conn
		serv.UDPConn.SetReadBuffer((int)(serv.ClientReadBufferSize)) //set buffer size to 1024 bytes

	} else if cfg.ListenURL.Scheme == "http" {

		//http is yet another "special" case, client HTTP does the hard work

	} else {
		var conn net.Listener
		var err error
		conn, err = net.Listen(cfg.ListenURL.Scheme, cfg.ListenURL.Host+cfg.ListenURL.Path)

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
	server.stats.RedirectedLinesCount = server.RedirectedLinesCount.TotalCount.Get()
	server.stats.RejectedLinesCount = server.RejectedLinesCount.TotalCount.Get()

	server.stats.CurrentValidLineCount = server.ValidLineCount.TickCount.Get()
	server.stats.CurrentWorkerValidLineCount = server.WorkerValidLineCount.TickCount.Get()
	server.stats.CurrentInvalidLineCount = server.InvalidLineCount.TickCount.Get()
	server.stats.CurrentSuccessSendCount = server.SuccessSendCount.TickCount.Get()
	server.stats.CurrentFailSendCount = server.FailSendCount.TickCount.Get()
	server.stats.CurrentUnknownSendCount = server.UnknownSendCount.TickCount.Get()
	server.stats.CurrentAllLinesCount = server.AllLinesCount.TickCount.Get()
	server.stats.CurrentRedirectedLinesCount = server.RedirectedLinesCount.TickCount.Get()
	server.stats.CurrentRejectedLinesCount = server.RejectedLinesCount.TickCount.Get()

	server.stats.ValidLineCountList = append(server.stats.ValidLineCountList, server.ValidLineCount.TickCount.Get())
	server.stats.WorkerValidLineCountList = append(server.stats.WorkerValidLineCountList, server.WorkerValidLineCount.TickCount.Get())
	server.stats.InvalidLineCountList = append(server.stats.InvalidLineCountList, server.InvalidLineCount.TickCount.Get())
	server.stats.SuccessSendCountList = append(server.stats.SuccessSendCountList, server.SuccessSendCount.TickCount.Get())
	server.stats.FailSendCountList = append(server.stats.FailSendCountList, server.FailSendCount.TickCount.Get())
	server.stats.UnknownSendCountList = append(server.stats.UnknownSendCountList, server.UnknownSendCount.TickCount.Get())
	server.stats.UnsendableSendCountList = append(server.stats.UnsendableSendCountList, server.UnsendableSendCount.TickCount.Get())
	server.stats.AllLinesCountList = append(server.stats.AllLinesCountList, server.AllLinesCount.TickCount.Get())
	server.stats.RejectedCountList = append(server.stats.RejectedCountList, server.RejectedLinesCount.TickCount.Get())
	server.stats.RedirectedCountList = append(server.stats.RedirectedCountList, server.RedirectedLinesCount.TickCount.Get())
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
		server.stats.RejectedCountList = server.stats.RejectedCountList[1:server.NumStats]
		server.stats.RedirectedCountList = server.stats.RedirectedCountList[1:server.NumStats]
		server.stats.TicksList = server.stats.TicksList[1:server.NumStats]
		server.stats.GoRoutinesList = server.stats.GoRoutinesList[1:server.NumStats]
	}
	server.stats.UpTimeSeconds = int64(elasped_sec)
	server.stats.CurrentReadBufferSize = server.CurrentReadBufferRam.Get()
	server.stats.MaxReadBufferSize = server.MaxReadBufferSize

	server.stats.InputQueueSize = len(server.InputQueue)
	server.stats.WorkQueueSize = len(server.WorkQueue)

	stats.StatsdClient.GaugeAbsolute(fmt.Sprintf("%s.inputqueue.length", server.Name), int64(server.stats.InputQueueSize))
	stats.StatsdClient.GaugeAbsolute(fmt.Sprintf("%s.workqueue.length", server.Name), int64(server.stats.WorkQueueSize))
	stats.StatsdClient.GaugeAbsolute(fmt.Sprintf("%s.readbuffer.length", server.Name), int64(server.stats.CurrentReadBufferSize))

	server.stats.ValidLineCountPerSec = server.ValidLineCount.TotalRate(elapsed)
	server.stats.WorkerValidLineCountPerSec = server.WorkerValidLineCount.TotalRate(elapsed)
	server.stats.InvalidLineCountPerSec = server.InvalidLineCount.TotalRate(elapsed)
	server.stats.SuccessSendCountPerSec = server.SuccessSendCount.TotalRate(elapsed)
	server.stats.UnsendableSendCountPerSec = server.UnsendableSendCount.TotalRate(elapsed)
	server.stats.UnknownSendCountPerSec = server.UnknownSendCount.TotalRate(elapsed)
	server.stats.AllLinesCountPerSec = server.AllLinesCount.TotalRate(elapsed)
	server.stats.RejectedLinesCountPerSec = server.RedirectedLinesCount.TotalRate(elapsed)
	server.stats.RejectedLinesCountPerSec = server.RejectedLinesCount.TotalRate(elapsed)

	if server.ListenURL == nil {
		server.stats.Listening = "BACKEND-ONLY"
	} else {
		server.stats.Listening = server.ListenURL.String()
	}

	//XXX TODO FIX ME to look like a multi service line, not one big puddle
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
		stats.StatsdClient.GaugeAbsolute(fmt.Sprintf("%s.lrucache.length", server.Name), int64(length))
		stats.StatsdClient.GaugeAbsolute(fmt.Sprintf("%s.lrucache.size", server.Name), int64(size))
		stats.StatsdClient.GaugeAbsolute(fmt.Sprintf("%s.lrucache.capacity", server.Name), int64(capacity))

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

	server.log.Info("Server: ValidLineCount: %d", server.ValidLineCount.TotalCount)
	server.log.Info("Server: WorkerValidLineCount: %d", server.WorkerValidLineCount.TotalCount)
	server.log.Info("Server: InvalidLineCount: %d", server.InvalidLineCount.TotalCount)
	server.log.Info("Server: SuccessSendCount: %d", server.SuccessSendCount.TotalCount)
	server.log.Info("Server: FailSendCount: %d", server.FailSendCount.TotalCount)
	server.log.Info("Server: UnsendableSendCount: %d", server.UnsendableSendCount.TotalCount)
	server.log.Info("Server: UnknownSendCount: %d", server.UnknownSendCount.TotalCount)
	server.log.Info("Server: AllLinesCount: %d", server.AllLinesCount.TotalCount)
	server.log.Info("Server: GO Routines Running: %d", runtime.NumGoroutine())
	server.log.Info("Server: Current Buffer Size: %d/%d bytes", server.CurrentReadBufferRam, server.MaxReadBufferSize)
	server.log.Info("Server: Current Out Queue length: %d", len(server.WorkQueue))
	server.log.Info("Server: Current Input Queue length: %d", len(server.InputQueue))
	server.log.Info("-------")
	server.log.Info("Server Rate: Duration %ds", uint64(server.ticker/time.Second))
	server.log.Info("Server Rate: ValidLineCount: %.2f/s", server.ValidLineCount.Rate(server.ticker))
	server.log.Info("Server Rate: WorkerLineCount: %.2f/s", server.WorkerValidLineCount.Rate(server.ticker))
	server.log.Info("Server Rate: InvalidLineCount: %.2f/s", server.InvalidLineCount.Rate(server.ticker))
	server.log.Info("Server Rate: SuccessSendCount: %.2f/s", server.SuccessSendCount.Rate(server.ticker))
	server.log.Info("Server Rate: FailSendCount: %.2f/s", server.FailSendCount.Rate(server.ticker))
	server.log.Info("Server Rate: UnsendableSendCount: %.2f/s", server.UnsendableSendCount.Rate(server.ticker))
	server.log.Info("Server Rate: UnknownSendCount: %.2f/s", server.UnknownSendCount.Rate(server.ticker))
	server.log.Info("Server Rate: RejectedSendCount: %.2f/s", server.RejectedLinesCount.Rate(server.ticker))
	server.log.Info("Server Rate: RedirectedSendCount: %.2f/s", server.RedirectedLinesCount.Rate(server.ticker))
	server.log.Info("Server Rate: AllLinesCount: %.2f/s", server.AllLinesCount.Rate(server.ticker))
	server.log.Info("Server Send Method:: %s", server.SendingConnectionMethod)
	for idx, pool := range server.Outpool {
		server.log.Info("Free Connections in Pools [%s]: %d/%d", idx, pool.NumFree(), pool.GetMaxConnections())
	}

	server.ResetTickers()
	//runtime.GC()
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

	//admin like functions to add and remove servers to a hashring
	http.HandleFunc(fmt.Sprintf("/%s/addserver", server.Name), addnode)
	http.HandleFunc(fmt.Sprintf("/%s/purgeserver", server.Name), purgenode)

}

// Takes a split item and processes it, all clients need to call this to actually "do" something
// once things are split from the incoming
func (server *Server) ProcessSplitItem(splitem splitter.SplitItem, out_queue chan splitter.SplitItem) error {

	// based on the Key we get re may need to redirect this to another backend
	// due to the PreReg items
	// so you may ask why Here and not before the InputQueue
	// 1. back pressure
	// 2. input queue buffering
	// 3. Incoming lines gets sucked in faster before (and "return ok") quickly

	// if the split item has "already been parsed" we DO NOT send it here (AccumulatedParsed)
	// instead we send it to the Worker output queues

	// nothing to see here move along
	if server.PreRegFilter == nil {
		server.SendtoOutputWorkers(splitem, out_queue)
		return nil
	}

	//if we've already done the AccumulatedParsed then we do a final pregre
	// on the new items then direct send, otherwise if there is an Accumulator
	// we send to that
	accumulate := splitem.Phase() != splitter.AccumulatedParsed && server.PreRegFilter.Accumulator != nil
	if accumulate {
		return server.PreRegFilter.Accumulator.ProcessSplitItem(splitem)
	}

	// match on the KEY not the entire string
	use_backend, reject, _ := server.PreRegFilter.FirstMatchBackend(splitem.Key())

	//if server.PreRegFilter
	if reject {
		stats.StatsdClient.Incr(fmt.Sprintf("prereg.backend.reject.%s", use_backend), 1)
		server.RejectedLinesCount.Up(1)
	} else if use_backend != server.Name {
		// redirect to another input queue
		stats.StatsdClient.Incr(fmt.Sprintf("prereg.backend.redirect.%s", use_backend), 1)
		server.RedirectedLinesCount.Up(1)
		// send to different backend to "repeat" this process
		SERVER_BACKENDS.Send(use_backend, splitem)
	} else {
		server.SendtoOutputWorkers(splitem, out_queue)
	}
	return nil

}

// accept incoming TCP connections and push them into the
// a connection channel
func (server *Server) Accepter() (<-chan net.Conn, error) {

	conns := make(chan net.Conn, server.Workers)
	go func() {
		defer func() {
			server.Connection.Close()
			if server.ListenURL != nil && server.ListenURL.Scheme == "unix" {
				os.Remove("/" + server.ListenURL.Host + server.ListenURL.Path)
			}
		}()
		defer close(conns)
		defer close(server.ShutDown)
		for {
			select {
			case s := <-server.back_pressure:
				if s {
					server.back_pressure_lock.Lock()
					server.log.Warning("Backpressure triggered pausing connections for : %s", server.back_pressure_sleep)
					time.Sleep(server.back_pressure_sleep)
					server.back_pressure_lock.Unlock()
				}
			case <-server.ShutDown:
				//the defers close us out
				return
			default:
				conn, err := server.Connection.Accept()
				if err != nil {
					server.log.Warning("Error Accecption Connection: %s", err)
					return
				}
				server.log.Debug("Accepted connection from %s", conn.RemoteAddr())

				conns <- conn
			}

		}
	}()
	return conns, nil
}

func (server *Server) startTCPServer(hashers *[]*ConstHasher, done chan Client) {

	//for TCPclients (and unix sockets),
	// since we basically create a client on each connect each time (unlike the UDP case)
	// and we want only one processing Q, set up this "queue" here
	// the main Server input Queue should be larger then the UDP case as we can have many TCP clients
	// and only one UDP client

	out_queue := make(chan splitter.SplitItem, server.Workers)
	defer close(out_queue)

	// tells the Acceptor to "sleep" incase we need to apply some back pressure
	// when connections overflood the acceptor
	server.back_pressure = make(chan bool, 1)
	defer close(server.back_pressure)

	run := func() {
		// consume the input queue of lines
		for splitem := range server.InputQueue {
			l_len := (int64)(len(splitem.Line()))
			server.AddToCurrentTotalBufferSize(l_len)

			if server.NeedBackPressure() {
				server.log.Warning(
					"Error::Max Queue or buffer reached dropping connection (Buffer %v, queue len: %v)",
					server.CurrentReadBufferRam.Get(),
					len(server.WorkQueue))
				server.BackPressure()
			}

			server.ProcessSplitItem(splitem, out_queue)
			server.AddToCurrentTotalBufferSize(-l_len)
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

			client := NewTCPClient(server, hashers, conn, done, out_queue)
			client.SetBufferSize((int)(server.ClientReadBufferSize))

			go client.handleRequest()
			go client.handleSend()

		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			ct := (int64)(len(server.WorkQueue))
			if ct >= server.Workers {
				stats.StatsdClient.Incr("worker.queue.isfull", 1)
				stats.StatsdClient.Incr(fmt.Sprintf("worker.%s.queue.isfull", server.Name), 1)
				//server.Logger.Printf("Worker Queue Full %d", ct)
			}
			stats.StatsdClient.GaugeAvg("worker.queue.length", ct)
			stats.StatsdClient.GaugeAvg(fmt.Sprintf("worker.%s.queue.length", server.Name), ct)
		case client := <-done:
			client.Close() //this will close the connection too
			client = nil
		}
	}

}

// different mechanism for UDP servers
func (server *Server) startUDPServer(hashers *[]*ConstHasher, done chan Client) {

	//just need one "client" here as we simply just pull from the socket
	client := NewUDPClient(server, hashers, server.UDPConn, done)
	client.SetBufferSize((int)(server.ClientReadBufferSize))

	go client.handleRequest()
	go client.handleSend()

	for {
		select {
		case <-server.ShutDown:
			client.Close()
			return
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			work_len := (int64)(len(server.WorkQueue))
			if work_len >= server.Workers {
				stats.StatsdClient.Incr("worker.queue.isfull", 1)
			}
			stats.StatsdClient.GaugeAvg("worker.queue.length", work_len)

		case client := <-done:
			client.Close()
		}
	}
}

// different mechanism for http servers much like UDP,
// client connections are handled by the golang http client bits so it appears
// as "one" client to us
func (server *Server) startHTTPServer(hashers *[]*ConstHasher, done chan Client) {

	client, err := NewHTTPClient(server, hashers, server.ListenURL, done)
	if err != nil {
		panic(err)
	}
	client.SetBufferSize((int)(server.ClientReadBufferSize))

	go client.handleRequest()
	go client.handleSend()

	for {
		select {
		case <-server.ShutDown:
			client.Close()
			return
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			work_len := (int64)(len(server.WorkQueue))
			if work_len >= server.Workers {
				stats.StatsdClient.Incr("worker.queue.isfull", 1)
			}
			stats.StatsdClient.GaugeAvg("worker.queue.length", work_len)

		case client := <-done:
			client.Close()
		}
	}
}

func (server *Server) startBackendServer(hashers *[]*ConstHasher, done chan Client) {

	out_queue := make(chan splitter.SplitItem, server.Workers)
	defer close(out_queue)

	// start the "backend only loop"
	run := func() {
		for splitem := range server.InputQueue {
			l_len := (int64)(len(splitem.Line()))
			server.AddToCurrentTotalBufferSize(l_len)
			server.ProcessSplitItem(splitem, out_queue)
			server.AllLinesCount.Up(1)
			stats.StatsdClient.Incr("incoming.backend.lines", 1)
			stats.StatsdClient.Incr(fmt.Sprintf("incoming.backend.%s.lines", server.Name), 1)
			server.AddToCurrentTotalBufferSize(-l_len)
		}
	}

	//fire up the workers
	for w := int64(1); w <= server.Workers; w++ {
		go run()
	}

	for {
		select {
		case l := <-out_queue:
			if len(l.Line()) == 0 {
				return
			}
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			ct := (int64)(len(server.WorkQueue))
			if ct >= server.Workers {
				stats.StatsdClient.Incr("worker.queue.isfull", 1)
				stats.StatsdClient.Incr(fmt.Sprintf("worker.%s.queue.isfull", server.Name), 1)
				//server.Logger.Printf("Worker Queue Full %d", ct)
			}
			stats.StatsdClient.GaugeAvg("worker.queue.length", ct)
			stats.StatsdClient.GaugeAvg(fmt.Sprintf("worker.%s.queue.length", server.Name), ct)

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

	//signal out
	server.ShutDown = make(chan bool, 1)
	server.TrapExit() //trappers

	//start tickin'
	go server.tickDisplay()

	return server, err
}

func (server *Server) StartServer() {

	server.log.Info("Using %d workers to process output", server.Workers)

	//fire up the send to workers
	done := make(chan Client)
	defer close(done)

	//the queue is only as big as the workers
	server.WorkQueue = make(chan *SendOut, server.Workers)
	defer close(server.WorkQueue)

	//input queue
	server.InputQueue = make(chan splitter.SplitItem, server.Workers)
	defer close(server.InputQueue)

	//set the push method (the WorkerOutput depends on it)
	server.SetPushMethod()

	for w := int64(1); w <= server.Workers; w++ {
		go server.WorkerOutput()
	}

	//get our line proessor in order
	_, err := server.SetSplitterProcessor()
	if err != nil {
		panic(err)
	}

	// add it to the list of backends available
	SERVER_BACKENDS.Add(server.Name, server)

	if server.ListenURL == nil {
		// just the little queue listener for pre-reg only redirects
		server.startBackendServer(&server.Hashers, done)
	} else if server.UDPConn != nil {
		server.startUDPServer(&server.Hashers, done)
	} else if server.ListenURL.Scheme == "http" {
		server.startHTTPServer(&server.Hashers, done)
	} else {
		// we treat the generic TCP and UNIX listeners the same
		server.startTCPServer(&server.Hashers, done)
	}
}