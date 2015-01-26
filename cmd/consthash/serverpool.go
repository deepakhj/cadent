/*
   The server we listen for our data
*/

package main

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

const (
	SERVER_TIMEOUT             = time.Duration(5) * time.Second
	DEFAULT_SERVER_OUT_COUNT   = uint64(3)
	DEFAULT_SERVER_RETRY       = time.Duration(2) * time.Second
	DEFAULT_SERVER_RE_ADD_TICK = DEFAULT_SERVER_OUT_COUNT * 4
)

// to be used by another object
type ServerPoolRunner interface {
	onServerUp(server string)
	onServerDown(server string)
}

type ServerPoolServer struct {
	Name string

	ServerPingCounts        uint64
	ServerUpCounts          uint64
	ServerDownCounts        uint64
	ServerCurrentDownCounts uint64
	ServerRequestCounts     uint64
}

type CheckedServerPool struct {
	ServerList []string

	ServerActions ServerPoolRunner

	Protocal string

	Servers        []ServerPoolServer
	DroppedServers []ServerPoolServer
	AllPingCounts  uint64

	// some stats
	ServerActiveList map[string]bool

	// if a server node is "down" do we take it out of the hash pile or just fail the key
	// "remove_node" or "fail_key"
	DownPolicy string

	// if 0 we never take out the node .. the check internal
	DownOutCount uint64

	ConnectionTimeout time.Duration

	ConnectionRetry time.Duration

	DoChecks bool
}

//make from our basic config object
func createServerPoolFromConfig(cfg *ConstHashConfig, serveraction *ServerPoolRunner) (*CheckedServerPool, error) {
	serverp, err := createServerPool(cfg.ServerListStr, cfg.Protocal, serveraction)
	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	serverp.ConnectionRetry = cfg.ServerHeartBeat
	serverp.DownOutCount = cfg.MaxServerHeartBeatFail
	serverp.ConnectionTimeout = cfg.ServerHeartBeatTimeout
	serverp.DownPolicy = cfg.ServerDownPolicy

	return &serverp, nil

}

func createServerPool(serverlist []string, protocal string, serveraction *ServerPoolRunner) (CheckedServerPool, error) {
	var serverp CheckedServerPool

	serverp.ServerList = serverlist
	serverp.Protocal = protocal

	serverp.ServerActions = *serveraction

	serverp.AllPingCounts = 0

	for _, server := range serverlist {

		//up
		serverp.ServerActions.onServerUp(server)

		c_server := ServerPoolServer{
			Name:                    server,
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

	for idx, _ := range self.DroppedServers {

		//add it back to the hash pool
		if self.DownPolicy == "remove_node" {
			self.ServerActions.onServerUp(self.DroppedServers[idx].Name)
		}
		log.Printf("Readded old dead server %s", self.DroppedServers[idx].Name)
		self.DroppedServers[idx].ServerCurrentDownCounts = 0
		new_s = append(new_s, self.DroppedServers[idx])
	}
	self.DroppedServers = nil
	self.Servers = new_s
}

func (self *CheckedServerPool) dropServer(server *ServerPoolServer) {

	var new_s []ServerPoolServer

	for idx, _ := range self.Servers {
		if &self.Servers[idx] == server {
			self.DroppedServers = append(self.DroppedServers, self.Servers[idx])
			// remove from serverp if we don't want it any more
			if self.DownPolicy == "remove_node" {
				self.ServerActions.onServerDown(self.Servers[idx].Name)
			}
		} else {
			new_s = append(new_s, self.Servers[idx])
		}
	}
	self.Servers = new_s
}

func (self *CheckedServerPool) testConnections() error {

	testUp := func(server *ServerPoolServer, out chan bool) {
		pings := atomic.AddUint64(&server.ServerPingCounts, 1)

		log.Printf("Health Checking %s, check %d", server.Name, pings)
		con, err := net.DialTimeout(self.Protocal, server.Name, self.ConnectionTimeout)

		if err != nil {
			log.Printf("Healthcheck for %s Failed: %s - Down %d times", server.Name, err, server.ServerDownCounts+1)
			if self.DownOutCount > 0 && server.ServerCurrentDownCounts+1 > self.DownOutCount {
				log.Printf("Health %s Fail too many times, taking out of pool", server.Name)
				self.dropServer(server)
			}
			out <- false
		} else {
			con.Close()
		}
		out <- true
	}

	gotResponse := func(server *ServerPoolServer, in <-chan bool) {
		message := <-in
		if message {
			server.ServerCurrentDownCounts = 0
			atomic.AddUint64(&server.ServerUpCounts, 1)
		} else {
			atomic.AddUint64(&server.ServerDownCounts, 1)
			atomic.AddUint64(&server.ServerCurrentDownCounts, 1)
		}
	}
	if self.ConnectionRetry < 2*self.ConnectionTimeout {
		log.Printf("Connection Retry CANNOT be less then 2x the Connection Timeout")
		return nil
	}

	for self.DoChecks {
		for idx, _ := range self.Servers {
			channel := make(chan bool)
			defer close(channel)
			go testUp(&self.Servers[idx], channel)
			go gotResponse(&self.Servers[idx], channel)
		}

		//re-add any killed servers after X ticker counts just to retest them
		atomic.AddUint64(&self.AllPingCounts, 1)
		if (self.AllPingCounts%DEFAULT_SERVER_RE_ADD_TICK == 0) && len(self.DroppedServers) > 0 {
			log.Printf("Attempting to re-add old dead server")
			self.reAddAllDroppedServers()
		}
		time.Sleep(self.ConnectionRetry)
	}
	return nil

}
