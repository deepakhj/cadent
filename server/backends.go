/*
   This is basically just a map of all the servers we have running
   based on their name in the config (the toml section
   so we can look them up and push things into their respective input quees
   from an external process

*/

package consthash

import (
	"fmt"
)

type Backend struct {
	Queue *Server
	Name  string
}

// basic alias to add a line to the backends input queue
func (bk *Backend) Send(line string) {
	bk.Queue.InputQueue <- line
}

type Backends map[string]*Backend

func (bk Backends) Add(name string, server *Server) {
	toadd := Backend{Name: name, Queue: server}
	bk[name] = &toadd
}

// send a line to the backend
func (bk Backends) Send(name string, line string) (err error) {

	got, ok := bk[name]
	if !ok {
		return fmt.Errorf("Backend `%s` not found", name)
	}
	got.Send(line)
	return nil
}
