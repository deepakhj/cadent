/*
   The server we listen for our data
*/

package main

import (
	"fmt"
	"log"
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

	ServerPingCounts        AtomicInt
	ServerUpCounts          AtomicInt
	ServerDownCounts        AtomicInt
	ServerCurrentDownCounts AtomicInt
	ServerRequestCounts     AtomicInt
}

type CheckedServerPool struct {
	mu sync.Mutex

	ServerList []*url.URL

	ServerActions ServerPoolRunner

	Protocal string

	Servers        []ServerPoolServer
	DroppedServers []ServerPoolServer
	AllPingCounts  AtomicInt

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
}

//make from our basic config object
func createServerPoolFromConfig(cfg *Config, serveraction ServerPoolRunner) (*CheckedServerPool, error) {

	serverp, err := createServerPool(cfg.ServerLists.ServerUrls, cfg.ServerLists.CheckUrls, serveraction)
	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	serverp.ConnectionRetry = cfg.ServerHeartBeat
	serverp.DownOutCount = int64(cfg.MaxServerHeartBeatFail)
	serverp.ConnectionTimeout = cfg.ServerHeartBeatTimeout
	serverp.DownPolicy = cfg.ServerDownPolicy
	//log.Print("createServerPoolFromConfig ", &serveraction)
	return serverp, nil

}

func createServerPool(serverlist []*url.URL, checklist []*url.URL, serveraction ServerPoolRunner) (serverp *CheckedServerPool, err error) {
	serverp = new(CheckedServerPool)

	serverp.ServerList = serverlist

	serverp.ServerActions = serveraction

	serverp.AllPingCounts = 0

	for idx, server := range serverlist {

		//up
		serverp.ServerActions.onServerUp(*server)

		c_server := ServerPoolServer{
			Name:                    fmt.Sprintf("%s", server),
			ServerURL:               *server,
			CheckName:               fmt.Sprintf("%s", checklist[idx]),
			CheckURL:                *checklist[idx],
			ServerPingCounts:        0,
			ServerUpCounts:          0,
			ServerDownCounts:        0,
			ServerCurrentDownCounts: 0,
			ServerRequestCounts:     0,
		}

		serverp.Servers = append(serverp.Servers, c_server)
	}

	serverp.ConnectionRetry = DEFAULT_SERVER_RETRY
	serverp.DownOutCount = DEFAULT_SERVER_OUT_COUNT
	serverp.ConnectionTimeout = SERVER_TIMEOUT
	serverp.DoChecks = true

	return serverp, nil
}

func (self *CheckedServerPool) stopChecks() {
	self.DoChecks = false
}

func (self *CheckedServerPool) startChecks() {
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
		log.Printf("Readded old dead server %s", self.DroppedServers[idx].Name)
		self.DroppedServers[idx].ServerCurrentDownCounts = 0
		new_s = append(new_s, self.DroppedServers[idx])
	}
	self.DroppedServers = nil
	self.Servers = new_s
}

func (self *CheckedServerPool) dropServer(server *ServerPoolServer) {
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

func (self *CheckedServerPool) testConnections() error {

	testConnection := func(url url.URL, timeout time.Duration) error {
		if url.Scheme == "tcp" || url.Scheme == "udp" {
			conn, err := net.DialTimeout(url.Scheme, url.Host, timeout)
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

	testUp := func(server *ServerPoolServer, out chan bool) {
		pings := server.ServerPingCounts.Add(1)

		log.Printf("Health Checking %s via %s, check %d", server.Name, server.CheckName, pings)
		err := testConnection(server.CheckURL, self.ConnectionTimeout)

		if err != nil {
			log.Printf("Healthcheck for %s Failed: %s - Down %d times", server.Name, err, server.ServerDownCounts+1)
			if self.DownOutCount > 0 && server.ServerCurrentDownCounts.Get()+1 > self.DownOutCount {
				log.Printf("Health %s Fail too many times, taking out of pool", server.Name)
				self.dropServer(server)
			}
			out <- false
		}
		out <- true
	}

	gotResponse := func(server *ServerPoolServer, in <-chan bool) {
		message := <-in
		if message {
			server.ServerCurrentDownCounts.Set(0)
			server.ServerUpCounts.Add(1)
		} else {
			server.ServerDownCounts.Add(1)
			server.ServerCurrentDownCounts.Add(1)
		}
	}
	if self.ConnectionRetry < 2*self.ConnectionTimeout {
		log.Printf("Connection Retry CANNOT be less then 2x the Connection Timeout")
		return nil
	}

	for self.DoChecks {
		self.mu.Lock()
		for idx, _ := range self.Servers {
			channel := make(chan bool)
			defer close(channel)
			go testUp(&self.Servers[idx], channel)
			go gotResponse(&self.Servers[idx], channel)
		}
		self.mu.Unlock()
		//re-add any killed servers after X ticker counts just to retest them
		self.AllPingCounts.Add(1)
		if (self.AllPingCounts%DEFAULT_SERVER_RE_ADD_TICK == 0) && len(self.DroppedServers) > 0 {
			log.Printf("Attempting to re-add old dead server")
			self.reAddAllDroppedServers()
		}
		time.Sleep(self.ConnectionRetry)
	}
	return nil

}
