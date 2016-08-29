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
	worker_queue chan *OutputMessage
	close        chan bool

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
	client.out_queue = server.ProcessedQueue
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
func (client *HTTPClient) ShutDown() {
	client.close <- true
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
func (client HTTPClient) WorkerQueue() chan *OutputMessage {
	return client.worker_queue
}
func (client HTTPClient) Close() {
	client.server = nil
	client.hashers = nil
	if client.Connection != nil {
		client.Connection.Close()
	}
}

/*
A Json input handler ..
*/
func (client *HTTPClient) JsonHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	out_error := func(err error) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf(`{"status":"error", "error": "%s"}`, err))
		// flush it
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		out_error(err)
		return
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(body))

	decoder := json.NewDecoder(bytes.NewBuffer(body))

	spl := new(splitter.JsonSplitter)
	lines := 0

	process_item := func(m *splitter.JsonStructSplitItem) error {
		splitem, err := spl.ParseJson(m)

		if err != nil {
			client.log.Errorf("Invalid Json Stat: %s", err)
			return err
		}
		splitem.SetOrigin(splitter.HTTP)
		splitem.SetOriginName(client.server.Name)
		stats.StatsdClient.Incr("incoming.http.json.lines", 1)
		client.server.ValidLineCount.Up(1)
		client.input_queue <- splitem
		lines += 1
		return nil
	}

	t, err := decoder.Token()
	if err != nil {
		client.log.Errorf("Invalid Json: %s", err)
		out_error(err)
		return
	}

	// better be "[" or "{"
	var in_char string
	switch t.(type) {
	case json.Delim:
		in_char = t.(json.Delim).String()
	default:
		err := fmt.Errorf("Invalid Json input")
		client.log.Errorf("Invalid Json: %s", err)
		out_error(err)
		return
	}
	if in_char == "[" {
		// while the array contains values
		for decoder.More() {
			var m splitter.JsonStructSplitItem
			// decode an array value (Message)
			err = decoder.Decode(&m)
			if err != nil {
				client.log.Errorf("Invalid Json Stat: %s", err)
				out_error(err)
				return

			}
			err = process_item(&m)
			if err != nil {
				client.log.Errorf("Invalid Json Stat: %s", err)
				out_error(err)
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
			out_error(err)
			return

		}
		if err != process_item(&m) {
			client.log.Errorf("Invalid Json Stat: %s", err)
			out_error(err)
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

func (client *HTTPClient) HttpHandler(w http.ResponseWriter, r *http.Request) {
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

		n_line := bytes.TrimSpace(line)
		if len(n_line) == 0 {
			continue
		}
		client.server.BytesReadCount.Up(uint64(len(line)))
		client.server.AllLinesCount.Up(1)
		splitem, err := client.server.SplitterProcessor.ProcessLine(n_line)
		if err == nil {
			splitem.SetOrigin(splitter.HTTP)
			splitem.SetOriginName(client.server.Name)
			stats.StatsdClient.Incr("incoming.http.lines", 1)
			client.server.ValidLineCount.Up(1)
			client.input_queue <- splitem
			lines += 1
		} else {
			client.server.InvalidLineCount.Up(1)
			stats.StatsdClient.Incr("incoming.http.invalidlines", 1)
			client.log.Warning("Invalid Line: %s (%s)", err, n_line)
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

func (client *HTTPClient) run(out_queue chan splitter.SplitItem) {
	for {
		select {
		case splitem := <-client.input_queue:
			client.server.ProcessSplitItem(splitem, out_queue)
		case <-client.close:
			return
		}
	}
}

// note close_client is not used, but for the interface due to the
// nature of http requests handling in go (different then straight TCP)
func (client HTTPClient) handleRequest(out_queue chan splitter.SplitItem, close_client chan bool) {

	// multi http servers needs new muxers
	// start up the http listens
	pth := client.url.Path
	if len(pth) == 0 {
		pth = "/"
	}
	serverMux := http.NewServeMux()
	json_pth := pth + "json"
	if !strings.HasSuffix(pth, "/") {
		json_pth = pth + "/json"
	}
	serverMux.HandleFunc(json_pth, client.JsonHandler)
	serverMux.HandleFunc(pth, client.HttpHandler)

	go http.Serve(client.Connection, serverMux)

	for w := int64(1); w <= client.server.Workers; w++ {
		go client.run(out_queue)
		go client.run(client.out_queue) // bleed out non-socket inputs
	}
	return
}

func (client HTTPClient) handleSend(out_queue chan splitter.SplitItem) {

	for {
		message := <-out_queue
		if !message.IsValid() {
			break
		}
	}
	client.log.Notice("Closing Http connection")
	//close it out
	client.Close()
	return
}
