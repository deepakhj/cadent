/*
	Interface to TCP/UDP clients
*/

package consthash

import "consthash/server/splitter"

type Client interface {
	handleRequest(out_queue chan splitter.SplitItem)
	handleSend(out_queue chan splitter.SplitItem)
	Close()
	SetBufferSize(int)
	Server() *Server
	Hashers() *[]*ConstHasher
	WorkerQueue() chan *SendOut
	ShutDown()
}
