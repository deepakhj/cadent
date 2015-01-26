/*
   The server we listen for our data
*/

package main

import (
	"fmt"
	"github.com/stathat/consistent"
	"log"
	//"net"
	//"sync/atomic"
	//"time"
)

// match the  ServerPoolRunner interface
type ConstHashHasher struct {
	Hasher     *consistent.Consistent
	ServerPool *CheckedServerPool

	ServerPutCounts uint64
	ServerGetCounts uint64
}

func (self ConstHashHasher) onServerUp(server string) {
	log.Printf("Adding server %s to hasher", server)
	self.Hasher.Add(server)
	log.Printf("Current members %s from hasher", self.Members())
}

func (self ConstHashHasher) onServerDown(server string) {
	log.Printf("Removing server %s from hasher", server)
	self.Hasher.Remove(server)
	log.Printf("Current members %s from hasher", self.Members())
}

//alias to hasher
func (self *ConstHashHasher) Get(in_key string) (string, error) {
	return self.Hasher.Get(in_key)
}

//alias to hasher
func (self *ConstHashHasher) Members() []string {
	return self.Hasher.Members()
}

//make from our basic config object
func createConstHasherFromConfig(cfg *ConstHashConfig) (*ConstHashHasher, error) {
	var hasher ConstHashHasher
	hasher.Hasher = consistent.New()
	s_pool_runner := ServerPoolRunner(hasher)
	s_pool, err := createServerPoolFromConfig(cfg, &s_pool_runner)

	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	hasher.ServerPool = s_pool
	return &hasher, nil

}

func createConstHasher(serverlist []string, protocal string) (*ConstHashHasher, error) {
	var hasher ConstHashHasher

	hasher.Hasher = consistent.New()

	// cast it to the interface
	s_pool_runner := ServerPoolRunner(hasher)

	s_pool, err := createServerPool(serverlist, protocal, &s_pool_runner)
	if err != nil {
		return nil, fmt.Errorf("Error setting up servers: %s", err)
	}
	hasher.ServerPool = &s_pool
	return &hasher, nil
}
