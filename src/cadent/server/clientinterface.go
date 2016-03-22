/*
	Interface to TCP/UDP clients
*/

package cadent

import "cadent/server/splitter"

type Client interface {
	handleRequest(out_queue chan splitter.SplitItem)
	handleSend(out_queue chan splitter.SplitItem)
	Close()
	SetBufferSize(int)
	Server() *Server
	Hashers() *[]*ConstHasher
	WorkerQueue() chan *OutputMessage
	ShutDown()
}