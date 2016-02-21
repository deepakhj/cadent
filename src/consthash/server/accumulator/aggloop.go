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

func (agg *AggregateLoop) SetWriter(conf AccumulatorWriter) error {

	// need a writer for each timer loop
	for _, dur := range agg.FlushTimes {
		wr, err := writers.NewLoop(conf.Driver)
		if err != nil {
			agg.log.Error("Writer error:: %s", err)
			return err
		}
		ops := conf.Options
		if ops == nil {
			ops = make(map[string]interface{})
		}
		ops["dsn"] = conf.DSN
		ops["prefix"] = fmt.Sprintf("_%0.0fs", dur.Seconds())
		err = wr.Config(ops)
		if err != nil {
			return err
		}
		agg.OutWriters = append(agg.OutWriters, wr)
	}
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
	_mu := mu
	// start up the writers listeners
	go writer.Start()

	post := func() {
		defer stats.StatsdNanoTimeFunc(fmt.Sprintf("aggregator.postwrite-time-ns"), time.Now())
		for _, stat := range agg.Aggregators.Get(duration).Items {
			stat.Resolution = _dur.Seconds()
			stat.TTL = int64(_ttl.Seconds()) // need to add in the TTL
			writer.WriteChan() <- stat
		}
		// need to clear out the Agg
		agg.Aggregators.Clear(duration)
		stats.StatsdClient.Incr(fmt.Sprintf("aggregator.%s.writesloops", duration.String()), 1)
	}

	ticker := time.NewTicker(_dur)
	for {
		select {
		case <-ticker.C:
			_mu.Lock()
			agg.log.Debug("Flushing %d stats in bin %s to writer", len(agg.Aggregators.Get(duration).Items), _dur.String())
			post() // for stats
			_mu.Unlock()

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
	return nil
}

func (agg *AggregateLoop) Stop() {
	agg.log.Notice("Initiating shutdown of aggregator for `%s`", agg.Name)
	agg.Shutdown.Send(true)
	for idx, _ := range agg.FlushTimes {
		agg.OutWriters[idx].Stop()
	}
	return
}
