/*
   Holds the current writer and sets up the channel loop to process incoming write requests
*/

package writers

import (
	"consthash/server/broadcast"
	"consthash/server/repr"
	"consthash/server/stats"
	"consthash/server/writers/indexer"
	"consthash/server/writers/metrics"
	"fmt"
	//"log"
	"time"
)

const (
	WRITER_DEFAULT_INDEX_QUEUE_LENGTH  = 1024 * 200
	WRITER_DEFAULT_METRIC_QUEUE_LENGTH = 1024 * 100
)

// toml config for Metrics
type WriterMetricConfig struct {
	Driver      string                 `toml:"driver"`
	DSN         string                 `toml:"dsn"`
	QueueLength int                    `toml:"queue_length"` // metric write queue length
	Options     map[string]interface{} `toml:"options"`      // option=[ [key, value], [key, value] ...]
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

	if wc.QueueLength <= 0 {
		wc.QueueLength = WRITER_DEFAULT_METRIC_QUEUE_LENGTH
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
	Metrics            WriterMetricConfig  `toml:"metrics"`
	Indexer            WriterIndexerConfig `toml:"indexer"`
	MetricQueueLength  int                 `toml:"metric_queue_length"`  // metric write queue length
	IndexerQueueLength int                 `toml:"indexer_queue_length"` // indexer write queue length

}

type WriterLoop struct {
	name         string
	metrics      metrics.Metrics
	indexer      indexer.Indexer
	write_chan   chan repr.StatRepr
	indexer_chan chan repr.StatRepr
	shutdowner   *broadcast.Broadcaster
	MetricQLen   int
	IndexerQLen  int
}

func New() (loop *WriterLoop, err error) {
	loop = new(WriterLoop)
	loop.MetricQLen = WRITER_DEFAULT_METRIC_QUEUE_LENGTH
	loop.IndexerQLen = WRITER_DEFAULT_INDEX_QUEUE_LENGTH

	loop.shutdowner = broadcast.New(1)
	return loop, nil
}

func (loop *WriterLoop) SetName(name string) {
	loop.name = name
}

func (loop *WriterLoop) statTick() {
	for {
		time.Sleep(time.Second)
		stats.StatsdClientSlow.GaugeAvg(fmt.Sprintf("writer.metricsqueue.%s.length", loop.name), int64(len(loop.write_chan)))
		stats.StatsdClientSlow.GaugeAvg(fmt.Sprintf("writer.indexerqueue.%s.length", loop.name), int64(len(loop.indexer_chan)))
		//log.Printf("Write Queue Length: %s: Metrics: %d Indexer %d", loop.name, len(loop.write_chan), len(loop.indexer_chan))
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
	shut := loop.shutdowner.Listen()
	for {
		select {
		case stat := <-loop.indexer_chan:
			// indexing can be very expensive (at least for cassandra)
			// and should have their own internal queues and smarts for handleing a massive influx of metric names
			loop.indexer.Write(stat.Key)
		case <-shut.Ch:
			shut.Close()
			return
		}
	}
	return
}

func (loop *WriterLoop) procLoop() {
	shut := loop.shutdowner.Listen()

	for {
		select {
		case stat := <-loop.write_chan:
			go func() {
				loop.metrics.Write(stat)
				loop.indexer_chan <- stat
				return
			}() // non-blocking indexer loop
		case <-shut.Ch:
			shut.Close()
			return
		}
	}
	return
}

func (loop *WriterLoop) Full() bool {
	return len(loop.write_chan) >= loop.MetricQLen || len(loop.indexer_chan) >= loop.IndexerQLen
}

func (loop *WriterLoop) Start() {
	loop.write_chan = make(chan repr.StatRepr, loop.MetricQLen)
	loop.indexer_chan = make(chan repr.StatRepr, loop.IndexerQLen) // indexing is slow, so we'll need to buffer things a bit more

	go loop.indexLoop()
	go loop.procLoop()
	go loop.statTick()
	return
}

func (loop *WriterLoop) Stop() {
	go func() {
		loop.shutdowner.Send(true)
		return
	}()
}
