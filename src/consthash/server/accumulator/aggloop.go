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
	repr "consthash/server/repr"
	stats "consthash/server/stats"
	writers "consthash/server/writers"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

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
		InputChan:   make(chan repr.StatRepr, 10000),
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
	for {
		select {
		case stat := <-agg.InputChan:
			agg.Aggregators.Add(stat)
		case <-shut.Ch:
			shut.Close()
			return
		}
	}
	return
}

// start the looper for each flush time as well as the writers
func (agg *AggregateLoop) startWriteLooper(duration time.Duration, ttl time.Duration, writer *writers.WriterLoop, mu sync.Mutex) {
	shut := agg.Shutdown.Listen()

	_dur := duration
	_ttl := ttl
	//_mu := mu
	// start up the writers listeners
	go writer.Start()

	post := func() {
		defer stats.StatsdSlowNanoTimeFunc("aggregator.postwrite-time-ns", time.Now())

		//_mu.Lock()
		//defer _mu.Unlock()

		items := agg.Aggregators.Get(duration).Items
		agg.Aggregators.Clear(duration) // clear now before any "new" things get added
		for _, stat := range items {
			stat.Resolution = _dur.Seconds()
			stat.TTL = int64(_ttl.Seconds()) // need to add in the TTL
			//agg.log.Critical("CHAN WRITE: LEN: %d", len(writer.WriteChan()))
			writer.WriterChan() <- stat
		}
		// need to clear out the Agg
		stats.StatsdClientSlow.Incr(fmt.Sprintf("aggregator.%s.writesloops", _dur.String()), 1)
		return
	}
	agg.log.Notice("Starting Aggregater Loop for %s", _dur.String())
	ticker := time.NewTicker(_dur)
	for {
		select {
		case <-ticker.C:
			agg.log.Debug(
				"Flushing %d stats in bin %s to writer",
				len(agg.Aggregators.Get(duration).Items),
				_dur.String(),
			)
			go post()

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
