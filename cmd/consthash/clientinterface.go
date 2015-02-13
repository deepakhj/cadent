/*
	Interface to TCP/UDP clients
*/

package main

type Client interface {
	handleRequest()
	handleSend()
	Close()
	Server() *Server
	Hasher() *ConstHasher
	WorkerQueue() chan *SendOut
}
