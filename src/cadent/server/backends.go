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
   This is basically just a map of all the servers we have running
   based on their name in the config (the toml section
   so we can look them up and push things into their respective input quees
   from an external process

   It maintains a singleton that is the list o backends

*/

package cadent

import (
	splitter "cadent/server/splitter"
	"fmt"
)

type Backend struct {
	Queue *Server
	Name  string
}

// basic alias to add a line to a backend's input queue
// where things will start "all over again"
func (bk *Backend) Send(line splitter.SplitItem) {
	bk.Queue.InputQueue <- line
}

type Backends map[string]*Backend

// a singleton
var SERVER_BACKENDS Backends = make(Backends)

func (bk Backends) Get(name string) (*Server, error) {
	srv, ok := bk[name]
	if !ok {
		return nil, fmt.Errorf("backend %s not found", name)
	}
	return srv.Queue, nil
}

func (bk Backends) Add(name string, server *Server) {
	toadd := &Backend{Name: name, Queue: server}
	bk[name] = toadd
}

// send a line to the backend
func (bk Backends) Send(name string, line splitter.SplitItem) (err error) {

	got, ok := bk[name]
	if !ok {
		return fmt.Errorf("Backend `%s` not found", name)
	}
	got.Send(line)
	return nil
}
