/*
	Interface to TCP/UDP clients
*/

package main

type Client interface {
	handleRequest()
	handleSend()
	Close()
	Server() *Server
	Hashers() []*ConstHasher
	WorkerQueue() chan *SendOut
}
