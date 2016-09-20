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
   Holds the current writer and sets up the channel loop to process incoming write requests
*/

package writers

import (
	"cadent/server/broadcast"
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/utils"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	WRITER_DEFAULT_INDEX_QUEUE_LENGTH  = 1024 * 20
	WRITER_DEFAULT_METRIC_QUEUE_LENGTH = 1024 * 10
	WRITER_MAX_WRITE_QUEUE             = 10 * 1024 * 1024
)

var errNeedCacheName = errors.New("`Name` is required")
var ErrCacheOptionRequired = errors.New("Metrics configs need a `cache` option")
var errCacheNotFound = errors.New("Count not find cache with the provided name")

// toml config for Internal Caches
type WriterCacheConfig struct {
	Name            string  `toml:"name" json:"name"`
	SeriesEncoding  string  `toml:"series_encoding" json:"series_encoding"`
	BytesPerMetric  int     `toml:"bytes_per_metric" json:"bytes_per_metric"`
	MaxMetricKeys   int     `toml:"max_metrics" json:"max_metrics"`
	MaxTimeInCache  string  `toml:"max_time_in_cache" json:"max_time_in_cache"`
	CacheOverFlow   string  `toml:"overflow_method" json:"overflow_method"`
	BroadCastLength int     `toml:"broadcast_length" json:"broadcast_length"`
	LowFruitRate    float64 `toml:"low_fruit_rate" json:"low_fruit_rate"`
}

func (wc *WriterCacheConfig) New(resolution uint32) (*metrics.Cacher, error) {
	// grab from signleton list
	if len(wc.Name) == 0 {
		return nil, errNeedCacheName
	}
	c, err := metrics.GetCacherSingleton(fmt.Sprintf("%s:%d", wc.Name, resolution))
	if err != nil {
		return nil, err
	}
	c.Prefix = wc.Name
	if wc.BroadCastLength > 0 {
		c.SetMaxBroadcastLen(wc.BroadCastLength)
	}
	if wc.MaxMetricKeys > 0 {
		c.SetMaxKeys(wc.MaxMetricKeys)
	}
	if len(wc.CacheOverFlow) > 0 {
		c.SetMaxOverFlowMethod(wc.CacheOverFlow)
	}
	if len(wc.SeriesEncoding) > 0 {
		c.SetSeriesEncoding(wc.SeriesEncoding)
	}
	if wc.BytesPerMetric > 0 {
		c.SetMaxBytesPerMetric(wc.BytesPerMetric)
	}
	if len(wc.MaxTimeInCache) > 0 {
		d, err := time.ParseDuration(wc.MaxTimeInCache)
		if err != nil {
			return nil, err
		}
		c.SetMaxTimeInCache(uint32(d.Seconds()))
	}
	if wc.LowFruitRate > 0 {
		c.SetLowFruitRate(wc.LowFruitRate)
	}
	return c, nil
}

// toml config for Metrics
type WriterMetricConfig struct {
	Driver   string `toml:"driver" json:"driver"`
	DSN      string `toml:"dsn" json:"dsn"`
	UseCache string `toml:"cache" json:"cache"`

	QueueLength int                    `toml:"input_queue_length" json:"input_queue_length"` // metric write queue length
	Options     map[string]interface{} `toml:"options" json:"options"`                       // option=[ [key, value], [key, value] ...]
}

func (wc WriterMetricConfig) ResolutionsNeeded() (metrics.WritersNeeded, error) {
	return metrics.ResolutionsNeeded(wc.Driver)
}

func (wc WriterMetricConfig) NewMetrics(duration time.Duration, cache_config []WriterCacheConfig) (metrics.Metrics, error) {

	mets, err := metrics.NewWriterMetrics(wc.Driver)
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

	// a little special case to set the rollupType == "triggered" if we are the triggered driver
	if strings.HasSuffix(wc.Driver, "-triggered") {
		i_ops["rollup_type"] = "triggered"
	}

	// use the defined cacher object
	if len(wc.UseCache) == 0 {
		return nil, ErrCacheOptionRequired
	}

	// find the proper cache to use
	res := uint32(duration.Seconds())
	proper_name := fmt.Sprintf("%s:%d", wc.UseCache, res)
	have := false
	for _, c := range cache_config {
		got, err := c.New(res)
		if err != nil {
			return nil, err
		}
		if got.Name == proper_name {
			i_ops["cache"] = got
			have = true
			break
		}
	}
	if !have {
		return nil, errCacheNotFound
	}

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
	Driver  string                 `toml:"driver" json:"driver"`
	DSN     string                 `toml:"dsn" json:"dsn"`
	Options map[string]interface{} `toml:"options" json:"options"` // option=[ [key, value], [key, value] ...]
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
	Caches []WriterCacheConfig `toml:"caches" json:"caches"`

	Metrics WriterMetricConfig  `toml:"metrics" json:"metrics"`
	Indexer WriterIndexerConfig `toml:"indexer" json:"indexer"`

	// secondary writers i.e. write to multiple spots
	SubMetrics WriterMetricConfig  `toml:"submetrics" json:"submetrics"`
	SubIndexer WriterIndexerConfig `toml:"subindexer" json:"subindexer"`

	MetricQueueLength  int `toml:"metric_queue_length" json:"metric_queue_length"`   // metric write queue length
	IndexerQueueLength int `toml:"indexer_queue_length" json:"indexer_queue_length"` // indexer write queue length
}

type WriterLoop struct {
	name         string
	cache        *metrics.Cacher
	metrics      metrics.Metrics
	indexer      indexer.Indexer
	write_chan   chan *repr.StatRepr
	indexer_chan chan *repr.StatName
	shutdowner   *broadcast.Broadcaster
	write_queue  *WriteQueue
	MetricQLen   int
	IndexerQLen  int
	log          *logging.Logger
	startstop    utils.StartStop
	stopped      bool
}

func New() (loop *WriterLoop, err error) {
	loop = new(WriterLoop)
	loop.MetricQLen = WRITER_DEFAULT_METRIC_QUEUE_LENGTH
	loop.IndexerQLen = WRITER_DEFAULT_INDEX_QUEUE_LENGTH

	loop.shutdowner = broadcast.New(0)
	loop.write_queue = NewWriteQueue(WRITER_MAX_WRITE_QUEUE)
	loop.log = logging.MustGetLogger("writer")
	loop.stopped = false

	return loop, nil
}

func (loop *WriterLoop) SetName(name string) {
	loop.name = name
}

func (loop *WriterLoop) GetName() string {
	return loop.name
}

func (loop *WriterLoop) statTick() {
	shuts := loop.shutdowner.Listen()
	defer shuts.Close()
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-shuts.Ch:
			return
		case <-ticker.C:
			stats.StatsdClientSlow.GaugeAvg(fmt.Sprintf("writer.metricsqueue.%s.length", loop.name), int64(len(loop.write_chan)))
			stats.StatsdClientSlow.GaugeAvg(fmt.Sprintf("writer.indexerqueue.%s.length", loop.name), int64(len(loop.indexer_chan)))
			//log.Printf("Write Queue Length: %s: Metrics: %d Indexer %d", loop.name, len(loop.write_chan), len(loop.indexer_chan))
		}
	}
}

func (loop *WriterLoop) SetMetrics(mets metrics.Metrics) error {
	loop.metrics = mets
	return nil
}

func (loop *WriterLoop) Metrics() metrics.Metrics {
	return loop.metrics
}

func (loop *WriterLoop) SetIndexer(idx indexer.Indexer) error {
	loop.indexer = idx
	return nil
}

func (loop *WriterLoop) Indexer() indexer.Indexer {
	return loop.indexer
}

func (loop *WriterLoop) WriterChan() chan *repr.StatRepr {
	return loop.write_chan
}

func (loop *WriterLoop) indexLoop() {
	shut := loop.shutdowner.Listen()
	for {
		select {
		case stat, more := <-loop.indexer_chan:
			// indexing can be very expensive (at least for cassandra)
			// and should have their own internal queues and smarts for handleing a massive influx of metric names
			if !more {
				return
			}
			loop.indexer.Write(*stat)
		case <-shut.Ch:
			return
		}
	}
}

func (loop *WriterLoop) procLoop() {
	shut := loop.shutdowner.Listen()

	for {
		select {
		case stat, more := <-loop.write_chan:
			if !more {
				return
			}

			// push the stat to the dynamic sized queue (saves oddles of ram on large channels and large metrics dumps)
			err := loop.write_queue.Push(stat)
			if err != nil {
				loop.log.Error("write push error: %s", err)
			}
		case <-shut.Ch:
			loop.metrics.Stop()
			loop.indexer.Stop()
			return
		}
	}
}

func (loop *WriterLoop) processQueue() {
	shut := loop.shutdowner.Listen()

	proc_queue := func() {
		ct := loop.write_queue.Len()
		for i := 0; i < ct; i++ {
			stat := loop.write_queue.Poll()
			switch stat {
			case nil:
				time.Sleep(time.Second) // just pause a bit as the queue is empty
				break

			default:
				// metric writers send to indexer so as to take advantage of it's
				// caching middle layer to prevent pounding the queue
				loop.metrics.Write(*stat)
			}
		}
	}

	for {
		select {
		case <-shut.Ch:
			return
		default:
			proc_queue()
			time.Sleep(time.Second) // so as to not CPU burn
		}
	}
}

func (loop *WriterLoop) Full() bool {
	return len(loop.write_chan) >= loop.MetricQLen || len(loop.indexer_chan) >= loop.IndexerQLen
}

func (loop *WriterLoop) Start() {
	loop.startstop.Start(func() {
		loop.log.Notice("Starting Writer `%s`", loop.name)

		loop.write_chan = make(chan *repr.StatRepr, loop.MetricQLen)
		// indexing is slow, so we'll need to buffer things a bit more
		loop.indexer_chan = make(chan *repr.StatName, loop.IndexerQLen)

		go loop.metrics.Start()
		go loop.indexer.Start()
		go loop.indexLoop()
		go loop.procLoop()
		go loop.processQueue()
		go loop.statTick()
	})
}

func (loop *WriterLoop) Stop() {
	loop.startstop.Stop(func() {
		if loop.stopped {
			return
		}
		loop.stopped = true
		loop.log.Warning("Shutting down writer `%s`", loop.name)
		loop.shutdowner.Send(true)
		if loop.indexer_chan != nil {
			close(loop.indexer_chan)
			loop.indexer_chan = nil
		}
		if loop.write_chan != nil {
			close(loop.write_chan)
			loop.write_chan = nil
		}
		loop.log.Warning("Shutting down metrics writer `%s`", loop.name)
		loop.metrics.Stop()
		loop.log.Warning("Shutting down index writer `%s`", loop.name)
		loop.indexer.Stop()
	})
}

/******************************************************************
writer "queue"

 This may seem a bit "odd" as you may think "why not use a buffered channels"
 well a few reasons ..
 1) sometimes the length of the buffered channel nessesary is "huge" and golang does not clean GC ram if most of the
 time it's not used fully.
 2) in order processing for all buckets per "writer"
 this ensures that inputs are queued up in the order gotten as well as allowing the main aggregator loops
 to continue simply by filling this up
 3) "large influxes handling" (kinda like #1) but when all 10s,1m,10m flush at the same time, inputchannels will block
 hard on some many inputs/outputs.  this quickly dispatches them to a data structure that during the "slower" times
 can easily be consumed quickly
 4) It also keeps things running should cassandra get slow alowing the queue to fill up and hopefully, things don't
 run out of ram before that happens so there is a risk so make sure to properly set the "max queue size"

**/

type WriteNode struct {
	data *repr.StatRepr
	next *WriteNode
}

// Fifo queage
type WriteQueue struct {
	head     *WriteNode
	tail     *WriteNode
	count    int
	queuemax int
	lock     *sync.Mutex
}

//	Creates a new pointer to a new queue.
func NewWriteQueue(max int) *WriteQueue {
	q := &WriteQueue{queuemax: max}
	q.lock = &sync.Mutex{}
	return q
}

func (q *WriteQueue) SetMaxLen(max int) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.queuemax = max
}

//	Returns the number of elements in the queue (i.e. size/length)
func (q *WriteQueue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.count
}

//	Pushes/inserts a value at the end/tail of the queue.
//	Note: this function does mutate the queue.
func (q *WriteQueue) Push(item *repr.StatRepr) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.count >= q.queuemax {
		return fmt.Errorf("Max write queue hit (max: %d) .. cannot push", q.queuemax)
	}
	n := &WriteNode{data: item}

	if q.tail == nil {
		q.tail = n
		q.head = n
	} else {
		q.tail.next = n
		q.tail = n
	}
	q.count++
	return nil
}

//	Returns the value at the front of the queue.
//	i.e. the oldest value in the queue.
//	Note: this function does mutate the queue.
func (q *WriteQueue) Poll() *repr.StatRepr {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.head == nil {
		return nil
	}

	n := q.head
	q.head = n.next

	if q.head == nil {
		q.tail = nil
	}
	q.count--

	return n.data
}

//	Returns a read value at the front of the queue.
//	i.e. the oldest value in the queue.
//	Note: this function does NOT mutate the queue.
//	go-routine safe.
func (q *WriteQueue) Peek() *repr.StatRepr {
	q.lock.Lock()
	defer q.lock.Unlock()

	n := q.head
	if n == nil {
		return nil
	}

	return n.data
}
