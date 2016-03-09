/*
   Holds the current writer and sets up the channel loop to process incoming write requests
*/

package writers

import (
	"consthash/server/repr"
	"consthash/server/stats"
	"consthash/server/writers/indexer"
	"consthash/server/writers/metrics"
	"fmt"
	"log"
	"time"
)

// toml config for Metrics
type WriterMetricConfig struct {
	Driver  string                 `toml:"driver"`
	DSN     string                 `toml:"dsn"`
	Options map[string]interface{} `toml:"options"` // option=[ [key, value], [key, value] ...]
}

func (wc WriterMetricConfig) NewMetrics(duration time.Duration) (metrics.Metrics, error) {

	mets, err := metrics.NewMetrics(wc.Driver)
	if err != nil {
		return nil, err
	}
	i_ops := wc.Options
	if i_ops == nil {
		i_ops = make(map[string]interface{})
	}
	i_ops["dsn"] = wc.DSN
	i_ops["prefix"] = fmt.Sprintf("_%0.0fs", duration.Seconds())
	i_ops["resolution"] = duration.Seconds()

	err = mets.Config(i_ops)
	if err != nil {
		return nil, err
	}
	return mets, nil
}

// toml config for Indexer
type WriterIndexerConfig struct {
	Driver  string                 `toml:"driver"`
	DSN     string                 `toml:"dsn"`
	Options map[string]interface{} `toml:"options"` // option=[ [key, value], [key, value] ...]
}

func (wc WriterIndexerConfig) NewIndexer() (indexer.Indexer, error) {
	idx, err := indexer.NewIndexer(wc.Driver)
	if err != nil {
		return nil, err
	}
	i_ops := wc.Options
	if i_ops == nil {
		i_ops = make(map[string]interface{})
	}
	i_ops["dsn"] = wc.DSN
	err = idx.Config(i_ops)
	if err != nil {
		return nil, err
	}
	return idx, nil
}

type WriterConfig struct {
	Metrics WriterMetricConfig  `toml:"metrics"`
	Indexer WriterIndexerConfig `toml:"indexer"`
}

type WriterLoop struct {
	name         string
	metrics      metrics.Metrics
	indexer      indexer.Indexer
	write_chan   chan repr.StatRepr
	indexer_chan chan repr.StatRepr
	stop_chan    chan bool
}

func New() (loop *WriterLoop, err error) {
	loop = new(WriterLoop)
	loop.write_chan = make(chan repr.StatRepr, 100000)
	loop.indexer_chan = make(chan repr.StatRepr, 1000000) // indexing is slow, so we'll need to buffer things a bit more
	loop.stop_chan = make(chan bool, 1)
	return loop, nil
}

func (loop *WriterLoop) SetName(name string) {
	loop.name = name
}

func (loop *WriterLoop) statTick() {
	for {
		time.Sleep(time.Second)
		stats.StatsdClientSlow.Incr(fmt.Sprintf("writer.inputqueue.%s.length", loop.name), int64(len(loop.write_chan)))
		log.Printf("Write Queue Length %s: %d", loop.name, len(loop.write_chan))
	}
	return
}

func (loop *WriterLoop) SetMetrics(mets metrics.Metrics) error {
	loop.metrics = mets
	return nil
}

func (loop *WriterLoop) SetIndexer(idx indexer.Indexer) error {
	loop.indexer = idx
	return nil
}

func (loop *WriterLoop) WriterChan() chan repr.StatRepr {
	return loop.write_chan
}

func (loop *WriterLoop) indexLoop() {
	for {
		select {
		case stat := <-loop.indexer_chan:
			// indexing can be very expensive (at least for cassandra)
			// and should have their own internal queues and smarts for handleing a massive influx of metric names
			loop.indexer.Write(stat.Key)
		//log.Printf("Stat: %v", stat.Key)
		case <-loop.stop_chan:
			return
		}
	}
}

func (loop *WriterLoop) procLoop() {
	for {
		select {
		case stat := <-loop.write_chan:
			// indexing can be very expensive (at least for cassandra)
			// and should have their own internal queues and smarts for handleing a massive influx of metric names
			loop.metrics.Write(stat)
			go func() { loop.indexer_chan <- stat }() // non-blocking indexer loop
			//log.Printf("Stat: %v", stat.Key)
		case <-loop.stop_chan:
			return
		}
	}
	return
}

func (loop *WriterLoop) Start() {
	go loop.indexLoop()
	go loop.procLoop()
	go loop.statTick()
	return
}

func (loop *WriterLoop) Stop() {
	go func() {
		loop.stop_chan <- true
		return
	}()
}
