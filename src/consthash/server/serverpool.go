/*
   The server we listen for our data
*/

package consthash

import (
	"consthash/server/stats"
	"fmt"
	logging "github.com/op/go-logging"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	SERVER_TIMEOUT             = time.Duration(5) * time.Second
	DEFAULT_SERVER_OUT_COUNT   = 3
	DEFAULT_SERVER_RETRY       = time.Duration(2) * time.Second
	DEFAULT_SERVER_RE_ADD_TICK = DEFAULT_SERVER_OUT_COUNT * 4
)

// to be used by another object
type ServerPoolRunner interface {
	onServerUp(server url.URL)
	onServerDown(server url.URL)
}

type ServerPoolServer struct {
	//name (ip:port) of the real server
	Name string

	//the actual check URL ()
	// this string can be
	// tcp://ip:port
	// http://ip:port/path -- > 200 == healthy
	// this is here in case the actuall servers are UDP which
	// while checkable is not as a good way to really check servers
	// if will default to tcp://Name above
	CheckName string

	// proper URLs used for checks and servers
	ServerURL url.URL
	CheckURL  url.URL

	ServerPingCounts        stats.AtomicInt
	ServerUpCounts          stats.AtomicInt
	ServerDownCounts        stats.AtomicInt
	ServerCurrentDownCounts stats.AtomicInt
	ServerRequestCounts     stats.AtomicInt
}

type CheckedServerPool struct {
	mu sync.Mutex

	ServerList []*url.URL

	ServerActions ServerPoolRunner

	Protocal string

	Servers        []ServerPoolServer
	DroppedServers []ServerPoolServer
	AllPingCounts  stats.AtomicInt

	// some stats
	ServerActiveList map[string]bool

	// if a server node is "down" do we take it out of the hash pile or just fail the key
	// "remove_node" or "fail_key"
	DownPolicy string

	// if 0 we never take out the node .. the check internal
	DownOutCount int64

	ConnectionTimeout time.Duration

	ConnectionRetry time.Duration

	DoChecks bool

	log *logging.Logger
}

func (serv *CheckedServerPool) ServerNames() []string {
	var slist []string
	for _, srv := range serv.Servers {
		slist = append(slist, srv.Name)
	}
	return slist
}

// create a list of pools
func createServerPoolsFromConfig(cfg *Config, serveraction ServerPoolRunner) (pools []*CheckedServerPool, err error) {

	for _, on_servers := range cfg.ServerLists {
		serverp, err := createServerPoolFromConfig(cfg, on_servers, serveraction)
		if err != nil {
			return nil, fmt.Errorf("Error setting up servers: %s", err)
		}
		pools = append(pools, serverp)
	}
	return pools, nil
}

// create a single
func createServerPoolFromConfig(cfg *Config, serverlist *ParsedServerConfig, serveraction ServerPoolRunner) (*CheckedServerPool, error) {

	serverp, err := createServerPool(serverlist.ServerUrls, serverlist.CheckUrls, serveraction)
	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	serverp.ConnectionRetry = cfg.ServerHeartBeat
	serverp.DownOutCount = int64(cfg.MaxServerHeartBeatFail)
	serverp.ConnectionTimeout = cfg.ServerHeartBeatTimeout
	serverp.DownPolicy = cfg.ServerDownPolicy
	return serverp, nil
}

func createServerPool(serverlist []*url.URL, checklist []*url.URL, serveraction ServerPoolRunner) (serverp *CheckedServerPool, err error) {
	serverp = new(CheckedServerPool)

	serverp.log = logging.MustGetLogger("consthash.server")

	serverp.ServerList = serverlist

	serverp.ServerActions = serveraction

	serverp.AllPingCounts = 0

	for idx, server := range serverlist {
		serverp.AddServer(server, checklist[idx])
	}

	serverp.ConnectionRetry = DEFAULT_SERVER_RETRY
	serverp.DownOutCount = DEFAULT_SERVER_OUT_COUNT
	serverp.ConnectionTimeout = SERVER_TIMEOUT
	serverp.DoChecks = true
	return serverp, nil
}

func (self *CheckedServerPool) StopChecks() {
	self.DoChecks = false
}

func (self *CheckedServerPool) StartChecks() {
	self.DoChecks = true
	self.testConnections()
}

func (self *CheckedServerPool) isDown(server string) bool {
	for _, val := range self.DroppedServers {
		if server == val.Name {
			return true
		}
	}
	return false
}

func (self *CheckedServerPool) isUp(server string) bool {
	return !self.isDown(server)
}

// add any "dropped" servers back into the pool
func (self *CheckedServerPool) reAddAllDroppedServers() {
	new_s := self.Servers
	self.mu.Lock()
	defer self.mu.Unlock()

	for idx, _ := range self.DroppedServers {

		//add it back to the hash pool
		if self.DownPolicy == "remove_node" {
			self.ServerActions.onServerUp(self.DroppedServers[idx].ServerURL)
		}
		self.log.Notice("Readded old dead server %s", self.DroppedServers[idx].Name)
		self.DroppedServers[idx].ServerCurrentDownCounts = 0
		new_s = append(new_s, self.DroppedServers[idx])
	}
	self.DroppedServers = nil
	self.Servers = new_s
}

func (self *CheckedServerPool) AddServer(serverUrl *url.URL, checkUrl *url.URL) {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.ServerActions.onServerUp(*serverUrl)

	c_server := ServerPoolServer{
		Name:                    fmt.Sprintf("%s", serverUrl),
		ServerURL:               *serverUrl,
		CheckName:               fmt.Sprintf("%s", checkUrl),
		CheckURL:                *checkUrl,
		ServerPingCounts:        0,
		ServerUpCounts:          0,
		ServerDownCounts:        0,
		ServerCurrentDownCounts: 0,
		ServerRequestCounts:     0,
	}

	self.Servers = append(self.Servers, c_server)
	self.log.Notice("Added Server %s checked via %s", c_server.Name, c_server.CheckName)
}

func (self *CheckedServerPool) PurgeServer(serverUrl *url.URL) bool {
	// REMOVE a server in totality, it does not obey the "drop rules"
	// and won't be re-added __ever__

	self.mu.Lock()
	defer self.mu.Unlock()

	//the "new" server list
	var new_s []ServerPoolServer
	var new_s_list []*url.URL
	did := false

	for idx, serv := range self.Servers {

		if serv.ServerURL == *serverUrl {
			self.ServerActions.onServerDown(self.Servers[idx].ServerURL)
			did = true
		} else {
			new_s_list = append(new_s_list, &serv.ServerURL)
			new_s = append(new_s, serv)
		}

	}

	self.Servers = new_s
	self.ServerList = new_s_list
	self.log.Notice("Purged Server %s", serverUrl)
	return did
}

func (self *CheckedServerPool) dropServer(server *ServerPoolServer) {
	// called when we loose connections, only temp takes it out if desired

	self.mu.Lock()
	defer self.mu.Unlock()

	var new_s []ServerPoolServer

	for idx, _ := range self.Servers {
		if &self.Servers[idx] == server {
			self.DroppedServers = append(self.DroppedServers, self.Servers[idx])
			// remove from server pool if we don't want it any more
			if self.DownPolicy == "remove_node" {
				self.ServerActions.onServerDown(self.Servers[idx].ServerURL)
			}
		} else {
			new_s = append(new_s, self.Servers[idx])
		}
	}
	self.Servers = new_s
}

func (self *CheckedServerPool) testSingleConnection(url url.URL, timeout time.Duration) error {
	if url.Scheme == "tcp" || url.Scheme == "udp" || url.Scheme == "unix" {
		conn, err := net.DialTimeout(url.Scheme, url.Host+url.Path, timeout)
		if err == nil {
			conn.Close()
		}
		return err
	}

	//http
	client := &http.Client{
		Timeout: timeout,
	}
	_, err := client.Get(url.String())
	return err
}

func (self *CheckedServerPool) testUp(server *ServerPoolServer, out chan bool) {
	pings := server.ServerPingCounts.Add(1)

	self.log.Info("Health Checking %s via %s, check %d", server.Name, server.CheckName, pings)
	err := self.testSingleConnection(server.CheckURL, self.ConnectionTimeout)

	if err != nil {
		self.log.Warning("Healthcheck for %s Failed: %s - Down %d times", server.Name, err, server.ServerDownCounts+1)
		if self.DownOutCount > 0 && server.ServerCurrentDownCounts.Get()+1 > self.DownOutCount {
			self.log.Warning("Health %s Fail too many times, taking out of pool", server.Name)
			self.dropServer(server)
		}
		out <- false
		return
	}
	out <- true
}

func (self *CheckedServerPool) gotTestResponse(server *ServerPoolServer, in chan bool) {
	message := <-in
	if message {
		server.ServerCurrentDownCounts.Set(0)
		up := server.ServerUpCounts.Add(1)
		self.log.Info("Healthcheck for %s OK: UP %d pings", server.Name, up)
	} else {
		server.ServerDownCounts.Add(1)
		server.ServerCurrentDownCounts.Add(1)
	}
	close(in)
}

func (self *CheckedServerPool) cleanChannels(tester chan bool, done chan bool) {
	message := <-done
	if message {
		close(tester)
		close(done)
		self.log.Info("Closed all Channels")
	}
}

func (self *CheckedServerPool) testConnections() error {

	if self.ConnectionRetry < 2*self.ConnectionTimeout {
		self.log.Critical("Connection Retry CANNOT be less then 2x the Connection Timeout")
		return nil
	}

	for self.DoChecks {
		for idx, _ := range self.Servers {
			testerchan := make(chan bool, 1)

			go self.testUp(&self.Servers[idx], testerchan)
			go self.gotTestResponse(&self.Servers[idx], testerchan)
			// channels is closed in gotTestResponse when done
		}

		//re-add any killed servers after X ticker counts just to retest them
		self.AllPingCounts.Add(1)
		if (self.AllPingCounts%DEFAULT_SERVER_RE_ADD_TICK == 0) && len(self.DroppedServers) > 0 {
			self.log.Notice("Attempting to re-add old dead server")
			self.reAddAllDroppedServers()
		}
		time.Sleep(self.ConnectionRetry)
	}
	return nil

}