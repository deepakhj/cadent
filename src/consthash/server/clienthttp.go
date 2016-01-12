/*
	HTTP clients

	different then the tcp/udp/socket cases as the "client" starts a server itself on init
*/

package consthash

import (
	"bufio"
	"consthash/server/splitter"
	"consthash/server/stats"
	"fmt"
	logging "github.com/op/go-logging"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const HTTP_BUFFER_SIZE = 4098

type HTTPClient struct {
	server     *Server
	hashers    *[]*ConstHasher
	Connection *net.TCPListener
	url        *url.URL

	LineCount  uint64
	BufferSize int

	out_queue    chan splitter.SplitItem
	done         chan Client
	input_queue  chan splitter.SplitItem
	worker_queue chan *SendOut

	log *logging.Logger
}

func NewHTTPClient(server *Server, hashers *[]*ConstHasher, url *url.URL, done chan Client) (*HTTPClient, error) {

	client := new(HTTPClient)
	client.server = server
	client.hashers = hashers

	client.LineCount = 0
	client.url = url

	//to deref things
	client.worker_queue = server.WorkQueue
	client.input_queue = server.InputQueue
	client.out_queue = make(chan splitter.SplitItem, server.Workers)
	client.done = done
	client.log = server.log

	// we make our own "connection" as we want to fiddle with the timeouts, and buffers
	tcpAddr, err := net.ResolveTCPAddr("tcp", url.Host)
	if err != nil {
		return nil, fmt.Errorf("Error resolving: %s", err)
	}

	client.Connection, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("Error listening: %s", err)
	}
	client.SetBufferSize(HTTP_BUFFER_SIZE)

	return client, nil
}

// noop basically
func (client *HTTPClient) SetBufferSize(size int) error {
	client.BufferSize = size
	return nil
}

func (client HTTPClient) Server() (server *Server) {
	return client.server
}

func (client HTTPClient) Hashers() (hasher *[]*ConstHasher) {
	return client.hashers
}
func (client HTTPClient) InputQueue() chan splitter.SplitItem {
	return client.input_queue
}
func (client HTTPClient) WorkerQueue() chan *SendOut {
	return client.worker_queue
}
func (client HTTPClient) Close() {
	client.server = nil
	client.hashers = nil
	if client.Connection != nil {
		client.Connection.Close()
	}
}

func (client *HTTPClient) HttpHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	buf := bufio.NewReaderSize(r.Body, client.BufferSize)
	lines := 0
	for {
		line, err := buf.ReadString('\n')

		if err != nil {
			break
		}
		if len(line) == 0 {
			continue
		}

		//this will block once the queue is full
		if line == "" {
			continue
		}

		n_line := strings.Trim(line, "\n\t ")
		if len(n_line) == 0 {
			continue
		}
		client.server.AllLinesCount.Up(1)
		splitem, err := client.server.SplitterProcessor.ProcessLine(n_line)
		if err == nil {
			stats.StatsdClient.Incr("incoming.http.lines", 1)
		} else {
			stats.StatsdClient.Incr("incoming.http.invalidlines", 1)
			client.log.Warning("Invalid Line: %s (%s)", err, n_line)
		}

		client.input_queue <- splitem
		lines += 1
	}
	io.WriteString(w, fmt.Sprintf("lines processed %d", lines))
	// flush it
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

}

func (client *HTTPClient) run() {

	for splitem := range client.input_queue {

		client.server.ProcessSplitItem(splitem, client.out_queue)

	}
}

func (client HTTPClient) handleRequest() {

	// multi http servers needs new muxers
	// start up the http listens
	pth := client.url.Path
	if len(pth) == 0 {
		pth = "/"
	}
	serverMux := http.NewServeMux()
	serverMux.HandleFunc(pth, client.HttpHandler)
	go http.Serve(client.Connection, serverMux)

	for w := int64(1); w <= client.server.Workers; w++ {
		go client.run()
	}
}

func (client HTTPClient) handleSend() {

	for {
		message := <-client.out_queue
		if !message.IsValid() {
			break
		}
	}
	client.log.Notice("Closing Http connection")
	//close it out
	client.Close()
}
