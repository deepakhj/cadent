/*
   The accumulators first pass is to do the first flush time for incoming stats
   after that it's past to this object where it manages the multiple "keeper" loops
   for various time snapshots (i.e. 10s, 1m, 10m, etc)

   The Main Accumulator's job is simply to pass things to this object
   which can then use the Writers to dump data out as nessesary .. if there is no "writer" defined
   this object is not used as otherwise it does nothing useful except take up ram

   Based on the flusher times for a writer, it will prefix things by the string rep

   for instance if things are files the output will be "/path/to/file_[10s|60s|600s]"
   for Databases, the tables will be assumed "basetable_[10s|60s|600s]"

*/

package accumulator

import (
	broadcast "consthash/server/broadcast"
	dispatch "consthash/server/dispatch"
	repr "consthash/server/repr"
	stats "consthash/server/stats"
	writers "consthash/server/writers"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	AGGLOOP_DEFAULT_QUEUE_LENGTH = 1024 * 100
	AGGLOOP_DEFAULT_WORKERS      = 32
)

/** dispatcher job **/
type StatJob struct {
	Aggregators *repr.MultiAggregator
	Stat        repr.StatRepr
}

func (j StatJob) IncRetry() int {
	return 0
}

func (j StatJob) OnRetry() int {
	return 0
}

func (t StatJob) DoWork() error {
	t.Aggregators.Add(t.Stat)
	return nil
}

type AggregateLoop struct {
	mus []sync.Mutex

	// these are assigned from the config file in the PreReg config file
	FlushTimes []time.Duration `json:"flush_time"`
	TTLTimes   []time.Duration `json:"ttl_times"`

	Aggregators *repr.MultiAggregator
	Name        string

	Shutdown  *broadcast.Broadcaster
	InputChan chan repr.StatRepr

	OutWriters []*writers.WriterLoop // write to a DB of some kind on flush

	OutReader *writers.ApiLoop

	// dispathers

	stat_write_queue    chan dispatch.IJob
	stat_dispatch_queue chan chan dispatch.IJob
	stat_dispatcher     *dispatch.Dispatch

	log *logging.Logger
}

func NewAggregateLoop(flushtimes []time.Duration, ttls []time.Duration, name string) (*AggregateLoop, error) {

	agg := &AggregateLoop{
		Name:        name,
		FlushTimes:  flushtimes,
		TTLTimes:    ttls,
		mus:         make([]sync.Mutex, len(flushtimes)),
		Shutdown:    broadcast.New(1),
		Aggregators: repr.NewMulti(flushtimes),
		InputChan:   make(chan repr.StatRepr, AGGLOOP_DEFAULT_QUEUE_LENGTH),
	}

	agg.log = logging.MustGetLogger("aggregatorloop")

	return agg, nil
}

// config the HTTP interface if desired
func (agg *AggregateLoop) SetReader(conf writers.ApiConfig) error {
	rl := new(writers.ApiLoop)
	err := rl.Config(conf)
	if err != nil {
		return err
	}
	// set the resolution bits
	var res [][]int
	for idx, dur := range agg.FlushTimes {
		res = append(res, []int{
			int(dur.Seconds()),
			int(agg.TTLTimes[idx].Seconds()),
		})
	}

	rl.SetResolutions(res)
	rl.SetBasePath(conf.BasePath)
	agg.OutReader = rl
	return nil

}

func (agg *AggregateLoop) SetWriter(conf writers.WriterConfig) error {

	// need only one indexer
	idx, err := conf.Indexer.NewIndexer()
	if err != nil {
		return err
	}

	// need a writer for each timer loop
	for _, dur := range agg.FlushTimes {
		wr, err := writers.New()
		if conf.MetricQueueLength > 0 {
			wr.MetricQLen = conf.MetricQueueLength
		}
		if conf.IndexerQueueLength > 0 {
			wr.IndexerQLen = conf.IndexerQueueLength
		}
		if err != nil {
			agg.log.Error("Writer error:: %s", err)
			return err
		}
		wr.SetName(dur.String())
		mets, err := conf.Metrics.NewMetrics(dur)
		if err != nil {
			return err
		}
		mets.SetIndexer(idx)
		wr.SetMetrics(mets)
		wr.SetIndexer(idx)

		agg.OutWriters = append(agg.OutWriters, wr)
		agg.log.Notice("Started duration %s Aggregator writer", dur.String())

	}
	agg.log.Notice("Started %d Aggregator writers", len(agg.OutWriters))
	return nil
}

func (agg *AggregateLoop) startInputLooper() {
	shut := agg.Shutdown.Listen()

	if agg.stat_write_queue == nil {
		workers := AGGLOOP_DEFAULT_WORKERS
		agg.stat_write_queue = make(chan dispatch.IJob, AGGLOOP_DEFAULT_QUEUE_LENGTH)
		agg.stat_dispatch_queue = make(chan chan dispatch.IJob, workers)
		agg.stat_dispatcher = dispatch.NewDispatch(workers, agg.stat_dispatch_queue, agg.stat_write_queue)
		agg.stat_dispatcher.Run()
	}

	for {
		select {
		case stat := <-agg.InputChan:
			agg.stat_write_queue <- StatJob{Aggregators: agg.Aggregators, Stat: stat}
			//agg.Aggregators.Add(stat)
		case <-shut.Ch:
			shut.Close()
			return
		}
	}
	return
}

// this is a helper function to get things to "start" on nicely "rounded"
// ticker intervals .. i.e. if  duration is 5 seconds .. start on t % 5
// this is approximate of course .. there will be some error, but it will be "close"
func (agg *AggregateLoop) delayRoundedTicker(duration time.Duration) *time.Ticker {
	time.Sleep(time.Now().Truncate(duration).Add(duration).Sub(time.Now()))
	return time.NewTicker(duration)
}

// start the looper for each flush time as well as the writers
func (agg *AggregateLoop) startWriteLooper(duration time.Duration, ttl time.Duration, writer *writers.WriterLoop, mu sync.Mutex) {
	shut := agg.Shutdown.Listen()

	_dur := duration
	_ttl := ttl
	//_mu := mu
	// start up the writers listeners
	go writer.Start()

	post := func(items map[string]repr.StatRepr) {
		defer stats.StatsdSlowNanoTimeFunc("aggregator.postwrite-time-ns", time.Now())

		//_mu.Lock()
		//defer _mu.Unlock()

		if writer.Full() {
			agg.log.Critical(
				"Saddly the write queue is full, if we continue adding to it, the entire world dies, we have to bail this write tick (metric queue: %d, indexer queue: %d)",
				writer.MetricQLen,
				writer.IndexerQLen,
			)
			return
		}
		for _, stat := range items {
			//agg.log.Critical("CHAN WRITE: LEN: %d", len(writer.WriteChan()))
			go func() {
				stat.Resolution = _dur.Seconds()
				stat.TTL = int64(_ttl.Seconds()) // need to add in the TTL
				writer.WriterChan() <- stat
				return
			}()
		}
		//agg.Aggregators.Clear(duration) // clear now before any "new" things get added
		// need to clear out the Agg
		stats.StatsdClientSlow.Incr(fmt.Sprintf("aggregator.%s.writesloops", _dur.String()), 1)
		return
	}

	agg.log.Notice("Starting Aggregater Loop for %s", _dur.String())
	ticker := agg.delayRoundedTicker(_dur) // time.NewTicker(_dur)
	for {
		select {
		case dd := <-ticker.C:
			items := agg.Aggregators.Get(duration).GetAndClear()
			i_len := len(items)
			agg.log.Debug(
				"Flushing %d stats in bin %s to writer at: %d",
				i_len,
				_dur.String(),
				dd.Unix(),
			)
			if i_len == 0 {
				agg.log.Warning(
					"No stats to send to writer in bin %s at: %d",
					_dur.String(),
					dd.Unix(),
				)
				continue
			}
			go post(items)

		case <-shut.Ch:
			ticker.Stop()
			shut.Close()
			return
		}
	}
	return
}

// For every flus time, start a new timer loop to perform writes
func (agg *AggregateLoop) Start() error {
	agg.log.Notice("Starting Aggregator Loop for `%s`", agg.Name)
	//start the input loop acceptor
	go agg.startInputLooper()
	for idx, dur := range agg.FlushTimes {
		go agg.startWriteLooper(dur, agg.TTLTimes[idx], agg.OutWriters[idx], agg.mus[idx])
	}

	// fire up the reader if around
	if agg.OutReader != nil {
		go agg.OutReader.Start()
	}
	return nil
}

func (agg *AggregateLoop) Stop() {
	agg.log.Notice("Initiating shutdown of aggregator for `%s`", agg.Name)
	go func() {
		agg.Shutdown.Send(true)
		for idx := range agg.FlushTimes {
			agg.OutWriters[idx].Stop()
		}
		return
	}()
	return
}
