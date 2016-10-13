/*
Copyright 2016 Under Armour, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
	HTTP clients

	different then the tcp/udp/socket cases as the "client" starts a server itself on init
*/

package cadent

import (
	"bufio"
	"bytes"
	"cadent/server/repr"
	"cadent/server/splitter"
	"cadent/server/stats"
	"encoding/json"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// HTTP_BUFFER_SIZE tcp buffer for incoming http requests
const HTTP_BUFFER_SIZE = 4098

// HTTPClient for HTTP inputs
type HTTPClient struct {
	server     *Server
	hashers    *[]*ConstHasher
	Connection *net.TCPListener
	url        *url.URL

	LineCount  uint64
	BufferSize int

	outQueue    chan splitter.SplitItem
	done        chan Client
	inputQueue  chan splitter.SplitItem
	workerQueue chan *OutputMessage
	close       chan bool

	log *logging.Logger
}

// NewHTTPClient makes a new client
func NewHTTPClient(server *Server, hashers *[]*ConstHasher, url *url.URL, done chan Client) (*HTTPClient, error) {

	client := new(HTTPClient)
	client.server = server
	client.hashers = hashers

	client.LineCount = 0
	client.url = url

	//to deref things
	client.workerQueue = server.WorkQueue
	client.inputQueue = server.InputQueue
	client.outQueue = server.ProcessedQueue
	client.done = done
	client.close = make(chan bool)
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

// ShutDown the client
func (client *HTTPClient) ShutDown() {
	client.close <- true
}

// SetBufferSize Noop for http client
func (client *HTTPClient) SetBufferSize(size int) error {
	client.BufferSize = size
	return nil
}

// Server of the client
func (client HTTPClient) Server() (server *Server) {
	return client.server
}

// Hashers the list of hashers
func (client HTTPClient) Hashers() (hasher *[]*ConstHasher) {
	return client.hashers
}

// InputQueue chan
func (client HTTPClient) InputQueue() chan splitter.SplitItem {
	return client.inputQueue
}

// WorkerQueue chan
func (client HTTPClient) WorkerQueue() chan *OutputMessage {
	return client.workerQueue
}

// Close stop accepting
func (client HTTPClient) Close() {
	client.server = nil
	client.hashers = nil
	if client.Connection != nil {
		client.Connection.Close()
	}
}

// JSONHandler json `{httproot}/json` handler
func (client *HTTPClient) JSONHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	outError := func(err error) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf(`{"status":"error", "error": "%s"}`, err))
		// flush it
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		outError(err)
		return
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(body))

	decoder := json.NewDecoder(bytes.NewBuffer(body))

	spl := new(splitter.JsonSplitter)
	lines := 0

	processItem := func(m *splitter.JsonStructSplitItem) error {
		splitem, err := spl.ParseJson(m)

		if err != nil {
			client.log.Errorf("Invalid Json Stat: %s", err)
			return err
		}
		splitem.SetOrigin(splitter.HTTP)
		splitem.SetOriginName(client.server.Name)
		stats.StatsdClient.Incr("incoming.http.json.lines", 1)
		client.server.ValidLineCount.Up(1)
		client.inputQueue <- splitem
		lines++
		return nil
	}

	t, err := decoder.Token()
	if err != nil {
		client.log.Errorf("Invalid Json: %s", err)
		outError(err)
		return
	}

	// better be "[" or "{"
	var inChar string
	switch t.(type) {
	case json.Delim:
		inChar = t.(json.Delim).String()
	default:
		err := fmt.Errorf("Invalid Json input")
		client.log.Errorf("Invalid Json: %s", err)
		outError(err)
		return
	}
	if inChar == "[" {
		// while the array contains values
		for decoder.More() {
			var m splitter.JsonStructSplitItem
			// decode an array value (Message)
			err = decoder.Decode(&m)
			if err != nil {
				client.log.Errorf("Invalid Json Stat: %s", err)
				outError(err)
				return

			}
			err = processItem(&m)
			if err != nil {
				client.log.Errorf("Invalid Json Stat: %s", err)
				outError(err)
				return
			}
		}
	} else {
		// we have a "{" start, but now the object is a bit funky, but it shuld be a small
		// string so we can reset the buffer
		var m splitter.JsonStructSplitItem
		// decode an array value (Message)
		err := json.Unmarshal(body, &m)

		if err != nil {
			client.log.Errorf("Invalid Json Stat: %s", err)
			outError(err)
			return

		}
		if err != processItem(&m) {
			client.log.Errorf("Invalid Json Stat: %s", err)
			outError(err)
			return
		}

	}

	io.WriteString(w, fmt.Sprintf(`{"status": "ok", "processed": %d}`, lines))
	// flush it
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return
}

// HTTPHandler basic line based body protocol
// the Body of the requests shold be based on the format specified in the config
func (client *HTTPClient) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	buf := bufio.NewReaderSize(r.Body, client.BufferSize)
	lines := 0
	for {
		line, err := buf.ReadBytes(repr.NEWLINE_SEPARATOR_BYTE)

		if err != nil {
			break
		}
		if len(line) == 0 {
			continue
		}

		nLine := bytes.TrimSpace(line)
		if len(nLine) == 0 {
			continue
		}
		client.server.BytesReadCount.Up(uint64(len(line)))
		client.server.AllLinesCount.Up(1)
		splitem, err := client.server.SplitterProcessor.ProcessLine(nLine)
		if err == nil {
			splitem.SetOrigin(splitter.HTTP)
			splitem.SetOriginName(client.server.Name)
			stats.StatsdClient.Incr("incoming.http.lines", 1)
			client.server.ValidLineCount.Up(1)
			client.inputQueue <- splitem
			lines++
		} else {
			client.server.InvalidLineCount.Up(1)
			stats.StatsdClient.Incr("incoming.http.invalidlines", 1)
			client.log.Warning("Invalid Line: %s (%s)", err, nLine)
			continue
		}

	}
	io.WriteString(w, fmt.Sprintf("lines processed %d", lines))
	// flush it
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return
}

func (client *HTTPClient) run(outQueue chan splitter.SplitItem) {
	for {
		select {
		case splitem := <-client.inputQueue:
			client.server.ProcessSplitItem(splitem, outQueue)
		case <-client.close:
			return
		}
	}
}

// note close_client is not used, but for the interface due to the
// nature of http requests handling in go (different then straight TCP)
func (client HTTPClient) handleRequest(outQueue chan splitter.SplitItem, closeClient chan bool) {

	// multi http servers needs new muxers
	// start up the http listens
	pth := client.url.Path
	if len(pth) == 0 {
		pth = "/"
	}
	serverMux := http.NewServeMux()
	jsonPth := pth + "json"
	if !strings.HasSuffix(pth, "/") {
		jsonPth = pth + "/json"
	}
	serverMux.HandleFunc(jsonPth, client.JSONHandler)
	serverMux.HandleFunc(pth, client.HTTPHandler)

	go http.Serve(client.Connection, serverMux)

	for w := int64(1); w <= client.server.Workers; w++ {
		go client.run(outQueue)
		go client.run(client.outQueue) // bleed out non-socket inputs
	}
	return
}

func (client HTTPClient) handleSend(outQueue chan splitter.SplitItem) {

	for {
		message := <-outQueue
		if !message.IsValid() {
			break
		}
	}
	client.log.Notice("Closing Http connection")
	//close it out
	client.Close()
	return
}
