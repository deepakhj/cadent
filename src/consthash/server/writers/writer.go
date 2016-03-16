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
	"sync"
	//"log"
	"time"
)

const (
	WRITER_DEFAULT_INDEX_QUEUE_LENGTH  = 1024 * 20
	WRITER_DEFAULT_METRIC_QUEUE_LENGTH = 1024 * 10
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
	write_queue  *WriteQueue
	MetricQLen   int
	IndexerQLen  int
}

func New() (loop *WriterLoop, err error) {
	loop = new(WriterLoop)
	loop.MetricQLen = WRITER_DEFAULT_METRIC_QUEUE_LENGTH
	loop.IndexerQLen = WRITER_DEFAULT_INDEX_QUEUE_LENGTH

	loop.shutdowner = broadcast.New(1)
	loop.write_queue = NewWriteQueue()
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
			/*
				//go func() {
				loop.metrics.Write(stat)
				//loop.indexer.Write(stat.Key)
				loop.indexer_chan <- stat
				//return
				//}() // non-blocking indexer loop
			*/
			loop.write_queue.Push(stat)
		case <-shut.Ch:
			shut.Close()
			return
		}
	}
	return
}

func (loop *WriterLoop) processQueue() {

	proc_queue := func() {
		ct := loop.write_queue.Len()
		for i := 0; i < ct; i++ {
			stat := loop.write_queue.Poll()
			switch stat {
			case nil:
				break
			default:
				loop.metrics.Write(*stat)
				loop.indexer.Write((*stat).Key)
			}
		}

	}
	for {
		proc_queue()
		time.Sleep(time.Millisecond)
	}
}

func (loop *WriterLoop) Full() bool {
	return len(loop.write_chan) >= loop.MetricQLen || len(loop.indexer_chan) >= loop.IndexerQLen
}

func (loop *WriterLoop) Start() {
	loop.write_chan = make(chan repr.StatRepr, loop.MetricQLen)
	loop.indexer_chan = make(chan repr.StatRepr, loop.IndexerQLen) // indexing is slow, so we'll need to buffer things a bit more

	go loop.indexLoop()
	go loop.procLoop()
	go loop.processQueue()
	go loop.statTick()
	return
}

func (loop *WriterLoop) Stop() {
	go func() {
		loop.shutdowner.Send(true)
		return
	}()
}

/**** writer "queue" **/

type WriteNode struct {
	data *repr.StatRepr
	next *WriteNode
}

//	A go-routine safe FIFO (first in first out) data stucture.
type WriteQueue struct {
	head  *WriteNode
	tail  *WriteNode
	count int
	lock  *sync.Mutex
}

//	Creates a new pointer to a new queue.
func NewWriteQueue() *WriteQueue {
	q := &WriteQueue{}
	q.lock = &sync.Mutex{}
	return q
}

//	Returns the number of elements in the queue (i.e. size/length)
//	go-routine safe.
func (q *WriteQueue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.count
}

//	Pushes/inserts a value at the end/tail of the queue.
//	Note: this function does mutate the queue.
//	go-routine safe.
func (q *WriteQueue) Push(item repr.StatRepr) {
	q.lock.Lock()
	defer q.lock.Unlock()

	n := &WriteNode{data: &item}

	if q.tail == nil {
		q.tail = n
		q.head = n
	} else {
		q.tail.next = n
		q.tail = n
	}
	q.count++
}

//	Returns the value at the front of the queue.
//	i.e. the oldest value in the queue.
//	Note: this function does mutate the queue.
//	go-routine safe.
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
