/*
	HTTP clients

	different then the tcp/udp/socket cases as the "client" starts a server itself on init
*/

package consthash

import (
	"bufio"
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

	out_queue    chan string
	done         chan Client
	input_queue  chan string
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
	client.out_queue = make(chan string, server.Workers)
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
func (client HTTPClient) InputQueue() chan string {
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
		client.input_queue <- line
		lines += 1
	}
	io.WriteString(w, fmt.Sprintf("lines processed %d", lines))
	// flush it
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

}

func (client *HTTPClient) run() {

	for line := range client.input_queue {

		if line == "" {
			continue
		}

		n_line := strings.Trim(line, "\n\t ")
		if len(n_line) == 0 {
			continue
		}
		client.server.AllLinesCount.Up(1)
		key, _, err := client.server.SplitterProcessor.ProcessLine(n_line)
		if err == nil {

			//based on the Key we get re may need to redirect this to another backend
			// due to the PreReg items
			// so you may ask why Here and not before the InputQueue, well backpressure, and we need to
			// we also want to make sure the NetConn gets sucked in faster before the processing
			// match on the KEY not the entire string
			if client.server.PreRegFilter != nil {
				use_backend, reject, _ := client.server.PreRegFilter.FirstMatchBackend(key)
				if reject {
					stats.StatsdClient.Incr(fmt.Sprintf("prereg.backend.reject.%s", use_backend), 1)
					client.server.RejectedLinesCount.Up(1)
					//client.log.Notice("REJECT LINE %s", n_line)
				} else if use_backend != client.server.Name {
					// redirect to another input queue
					stats.StatsdClient.Incr(fmt.Sprintf("prereg.backend.redirect.%s", use_backend), 1)
					client.server.RedirectedLinesCount.Up(1)
					SERVER_BACKENDS.Send(use_backend, n_line)
				} else {
					client.server.RunSplitter(key, n_line, client.out_queue)
				}
			} else {
				client.server.RunSplitter(key, n_line, client.out_queue)
			}
		} else {
			stats.StatsdClient.Incr("incoming.http.invalidlines", 1)
			client.log.Warning("Invalid Line: %s (%s)", err, n_line)
		}
		stats.StatsdClient.Incr("incoming.http.lines", 1)
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
		if len(message) == 0 {
			break
		}
	}
	client.log.Notice("Closing Http connection")
	//close it out
	client.Close()
}
