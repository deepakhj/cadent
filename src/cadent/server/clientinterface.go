/*
	Interface to TCP/UDP clients
*/

package cadent

import "cadent/server/splitter"

type Client interface {
	handleRequest(out_queue chan splitter.SplitItem, close_chan chan bool)
	handleSend(out_queue chan splitter.SplitItem)
	Close()
	SetBufferSize(int)
	Server() *Server
	Hashers() *[]*ConstHasher
	WorkerQueue() chan *OutputMessage
	ShutDown()
}
