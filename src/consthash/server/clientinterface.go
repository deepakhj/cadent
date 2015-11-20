/*
	Interface to TCP/UDP clients
*/

package consthash

type Client interface {
	handleRequest()
	handleSend()
	Close()
	SetBufferSize(int)
	Server() *Server
	Hashers() *[]*ConstHasher
	WorkerQueue() chan *SendOut
}
