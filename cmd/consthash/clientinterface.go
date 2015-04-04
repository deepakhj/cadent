/*
	Interface to TCP/UDP clients
*/

package main

type Client interface {
	handleRequest()
	handleSend()
	Close()
	SetBufferSize(int)
	Server() *Server
	Hashers() *[]*ConstHasher
	WorkerQueue() chan *SendOut
}
