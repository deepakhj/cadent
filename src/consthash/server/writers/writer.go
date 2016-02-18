/*
   Holds the current writer and sets up the channel loop to process incoming write requests
*/

package writers

import (
	"consthash/server/repr"
)

type WriterLoop struct {
	writer     Writer
	write_chan chan repr.StatRepr
	stop_chan  chan bool
}

func NewLoop(name string) (loop *WriterLoop, err error) {
	loop = new(WriterLoop)
	loop.writer, err = NewWriter(name)
	if err != nil {
		return nil, err
	}

	loop.write_chan = make(chan repr.StatRepr, 10000)
	loop.stop_chan = make(chan bool, 1)
	return loop, nil
}

func (loop *WriterLoop) Config(conf map[string]interface{}) error {
	return loop.writer.Config(conf)
}

func (loop *WriterLoop) WriteChan() chan repr.StatRepr {
	return loop.write_chan
}

func (loop *WriterLoop) procLoop() {
	for {
		select {
		case stat := <-loop.write_chan:
			loop.writer.Write(stat)
		case <-loop.stop_chan:
			return
		}
	}
	return
}

func (loop *WriterLoop) Start() {
	go loop.procLoop()
	return
}

func (loop *WriterLoop) Stop() {
	loop.stop_chan <- true
}
