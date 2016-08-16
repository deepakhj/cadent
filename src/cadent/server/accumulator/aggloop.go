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
	broadcast "cadent/server/broadcast"
	dispatch "cadent/server/dispatch"
	repr "cadent/server/repr"
	stats "cadent/server/stats"
	writers "cadent/server/writers"
	metrics "cadent/server/writers/metrics"
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
	mus []*sync.Mutex

	// these are assigned from the config file in the PreReg config file
	FlushTimes []time.Duration `json:"flush_time"`
	TTLTimes   []time.Duration `json:"ttl_times"`

	Aggregators *repr.MultiAggregator
	Name        string

	Shutdown   *broadcast.Broadcaster
	shutitdown bool
	InputChan  chan repr.StatRepr

	OutWriters []*writers.WriterLoop // write to a DB of some kind on flush

	OutReader *writers.ApiLoop

	// dispathers

	stat_write_queue    chan dispatch.IJob
	stat_dispatch_queue chan chan dispatch.IJob
	stat_dispatcher     *dispatch.Dispatch

	// if true will set the flusher to basically started at "now" time otherwise it will use time % duration
	// use case:
	// statsd flushes to a "none-writer" should be more or less randomized to keep everything
	// from hammering the pre-writer listeners ever tick
	// things that are writers (especially graphite style) should be flushed on
	// time % duration intervals
	// defaults to false
	flush_random_ticker bool

	log *logging.Logger
}

func NewAggregateLoop(flushtimes []time.Duration, ttls []time.Duration, name string) (*AggregateLoop, error) {

	agg := &AggregateLoop{
		Name:                name,
		FlushTimes:          flushtimes,
		TTLTimes:            ttls,
		mus:                 make([]*sync.Mutex, len(flushtimes)),
		Shutdown:            broadcast.New(1),
		Aggregators:         repr.NewMulti(flushtimes),
		InputChan:           make(chan repr.StatRepr, AGGLOOP_DEFAULT_QUEUE_LENGTH),
		flush_random_ticker: false,
	}

	agg.log = logging.MustGetLogger("aggloop." + name)

	return agg, nil
}

func (agg *AggregateLoop) getResolutionArray() [][]int {
	// get the [time, ttl] array for use in the Metrics Writers
	var res [][]int
	for idx, dur := range agg.FlushTimes {
		res = append(res, []int{
			int(dur.Seconds()),
			int(agg.TTLTimes[idx].Seconds()),
		})
	}
	return res
}

// config the HTTP interface if desired
func (agg *AggregateLoop) SetReader(conf writers.ApiConfig) error {
	rl := new(writers.ApiLoop)

	// set the resolution bits
	res := agg.getResolutionArray()

	// grab the first resolution as that's the one the main "reader" will be on
	err := rl.Config(conf, float64(res[0][0]))
	if err != nil {
		return err
	}

	rl.SetResolutions(res)
	rl.SetBasePath(conf.BasePath)
	agg.OutReader = rl
	return nil

}

// set the metrics and index writers types.  Based on the writer type
// the number of actuall aggregator loops needed may change
// for instance RRD file DBs typically "self rollup" so there's no need
// to deal with the aggregation of longer times, but DBs (cassandra) cannot do that
// automatically, so we set things appropriately
func (agg *AggregateLoop) SetWriter(conf writers.WriterConfig) error {

	// need only one indexer
	idx, err := conf.Indexer.NewIndexer()
	if err != nil {
		return err
	}

	num_writers, err := conf.Metrics.ResolutionsNeeded()

	// need to "reset" the Aggregators to just be the FIRST one if num_writers == FirstResolution
	if num_writers == metrics.FirstResolution {
		agg.Aggregators = repr.NewMulti([]time.Duration{agg.FlushTimes[0]})
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
		mets.SetResolutions(agg.getResolutionArray())
		wr.SetMetrics(mets)
		wr.SetIndexer(idx)

		agg.OutWriters = append(agg.OutWriters, wr)
		agg.log.Notice("Set Aggregator writer @ %s", dur.String())
		if num_writers == metrics.FirstResolution {
			agg.log.Notice("Only one writer needed for this writer driver")
			break
		}

	}
	agg.log.Notice("Set %d Aggregator writers", len(agg.OutWriters))
	return nil
}

func (agg *AggregateLoop) startInputLooper() {
	shut := agg.Shutdown.Listen()

	if agg.stat_write_queue == nil {
		workers := AGGLOOP_DEFAULT_WORKERS
		agg.stat_write_queue = make(chan dispatch.IJob, AGGLOOP_DEFAULT_QUEUE_LENGTH)
		agg.stat_dispatch_queue = make(chan chan dispatch.IJob, workers)
		agg.stat_dispatcher = dispatch.NewDispatch(workers, agg.stat_dispatch_queue, agg.stat_write_queue)
		agg.stat_dispatcher.Name = "aggloop"
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
}

// this is a helper function to get things to "start" on nicely "rounded"
// ticker intervals .. i.e. if  duration is 5 seconds .. start on t % 5
// this is approximate of course .. there will be some error, but it will be "close"
func (agg *AggregateLoop) delayRoundedTicker(duration time.Duration) *time.Ticker {
	time.Sleep(time.Now().Truncate(duration).Add(duration).Sub(time.Now()))
	return time.NewTicker(duration)
}

// start the looper for each flush time as well as the writers
func (agg *AggregateLoop) startWriteLooper(duration time.Duration, ttl time.Duration, writer *writers.WriterLoop, mu *sync.Mutex) {
	if agg.shutitdown {
		agg.log.Warning("Got shutdown signal, not starting writers")
		return
	}

	shut := agg.Shutdown.Listen()

	_dur := duration
	_ttl := ttl

	//_mu := mu
	// start up the writers listeners
	writer.Start()

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
			// agg.log.Critical("FLUSH POST: %s", stat)
			stat.Name.Resolution = uint32(_dur.Seconds())
			stat.Name.TTL = uint32(_ttl.Seconds()) // need to add in the TTL
			writer.WriterChan() <- stat
		}
		//agg.log.Critical("CHAN WRITE: LEN: %d Items: %d", len(writer.WriterChan()), m_items)
		//agg.Aggregators.Clear(duration) // clear now before any "new" things get added
		// need to clear out the Agg
		stats.StatsdClientSlow.Incr(fmt.Sprintf("aggregator.%s.writesloops", _dur.String()), 1)
		return
	}

	agg.log.Notice("Starting Aggregater Loop for %s", _dur.String())
	var ticker *time.Ticker
	// either flush at a random "duration interval" or flush at time % duration interval
	if agg.flush_random_ticker {
		agg.log.Notice("Aggregater Loop for %s at random start .. starting: %d", _dur.String(), time.Now().Unix())
		ticker = time.NewTicker(_dur)
	} else {
		agg.log.Notice("Aggregater Loop for %s starting at time %% %s .. starting %d", _dur.String(), _dur.String(), time.Now().Unix())
		ticker = agg.delayRoundedTicker(_dur)
	}
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
}

// For every flus time, start a new timer loop to perform writes
func (agg *AggregateLoop) Start() error {
	agg.log.Notice("Starting Aggregator Loop for `%s`", agg.Name)

	if agg.shutitdown {
		agg.log.Warning("Got shutdown signal, not starting loop")
		return nil
	}

	//start the input loop acceptor
	go agg.startInputLooper()
	for idx, writ := range agg.OutWriters {
		go agg.startWriteLooper(agg.FlushTimes[idx], agg.TTLTimes[idx], writ, agg.mus[idx])
	}

	// fire up the reader if around
	if agg.OutReader != nil {
		go agg.OutReader.Start()
	}
	return nil
}

func (agg *AggregateLoop) Stop() {
	if agg.shutitdown {
		return
	}
	agg.log.Warning("Initiating shutdown of aggregator for `%s`", agg.Name)

	if agg.stat_dispatcher != nil {
		agg.stat_dispatcher.Shutdown()
	}

	agg.Shutdown.Send(true)
	agg.shutitdown = true

	if agg.OutReader != nil {
		agg.OutReader.Stop()
	}

	for idx := range agg.OutWriters {
		agg.log.Warning("Starting Shutdown of writer `%s:%s`", agg.Name, agg.OutWriters[idx].GetName())
		agg.OutWriters[idx].Stop()
	}
	agg.log.Warning("Shutdown of aggregator `%s`", agg.Name)
}
