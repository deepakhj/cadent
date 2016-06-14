/*
   The accumulator aggrigator, based on a config

    THe "line" flow (for a given item) goes something like this

    Listen Ingest
    	-> PreReg (for rejection/backend selection)
    	-> Accumulate -> (Flush to OutputQueue of ToBackend)
    		-> BackendInputQueue
    		-> PreReg again (new keys and new backends, i.e. statsd -> graphite)
    			(note: skipping accumulate again if that backend PreReg has an accumulator
    			 i.e. the splitItem has AccumulatedParsed set)
    		-> Backend ConstHash
    		-> Out

    There must be a "splitter.Splitter" for each FormatterItem otherwise there's no way

    If there is an "external writer" you can set the Backend to 'black_hole' and the line "ends"
    there on Flush and the line should hopefully get "written" somewhere like your
    Favorite DB
*/

package accumulator

import (
	repr "cadent/server/repr"
	splitter "cadent/server/splitter"
	stats "cadent/server/stats"
	writers "cadent/server/writers"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const BLACK_HOLE_BACKEND = "BLACKHOLE"

/**** main accumulator object */

type Accumulator struct {

	// these are assigned from the config file in the PreReg config file
	ToBackend         string          `json:"backend"`
	FromBackend       string          `json:"from_backend"`
	FormatterName     string          `json:"formatter"`
	AccumulatorName   string          `json:"accumulator"`
	Name              string          `json:"name"`
	KeepKeys          bool            `json:"keep_keys"` // if true, will not "remove" the keys post flush, just set them to 0
	AccumulateTime    time.Duration   `json:"accumulate_timer"`
	FlushTimes        []time.Duration `json:"flush_time"`
	TTLTimes          []time.Duration `json:"ttl_times"`
	RandomTickerStart bool            `json:"random_ticker_start"`

	Accumulate AccumulatorItem
	Formatter  FormatterItem

	InSplitter  splitter.Splitter
	OutSplitter splitter.Splitter

	mu          sync.Mutex
	timer       *time.Ticker
	LineQueue   chan string
	OutputQueue chan splitter.SplitItem
	Shutdown    chan bool
	shutitdown  bool

	Aggregators *AggregateLoop // writers hook into the main agg flushing loops

	log *logging.Logger
}

func NewAccumlator(inputtype string, outputtype string, keepkeys bool, name string) (*Accumulator, error) {

	fmter, err := NewFormatterItem(outputtype)
	if err != nil {
		return nil, err
	}
	fmter.Init()

	acc, err := NewAccumulatorItem(inputtype)
	if err != nil {
		return nil, err
	}
	acc.Init(fmter)
	acc.SetKeepKeys(keepkeys)
	acc.SetResolution(time.Duration(time.Second))

	ac := &Accumulator{
		Accumulate:        acc,
		Formatter:         fmter,
		AccumulatorName:   inputtype,
		FormatterName:     outputtype,
		KeepKeys:          keepkeys,
		Name:              name,
		FlushTimes:        []time.Duration{time.Duration(time.Second)},
		AccumulateTime:    time.Duration(time.Second),
		Shutdown:          make(chan bool, 0),
		LineQueue:         make(chan string, 10000),
		shutitdown:        false,
		timer:             nil,
		RandomTickerStart: false,
	}

	ac.log = logging.MustGetLogger("accumulator." + name)

	// determine the splitter from the formatter item
	nul_conf := make(map[string]interface{})
	ispl, err := splitter.NewSplitterItem(inputtype, nul_conf)
	if err != nil {
		return nil, err
	}
	ac.InSplitter = ispl

	ospl, err := splitter.NewSplitterItem(outputtype, nul_conf)
	if err != nil {
		return nil, err
	}
	ac.OutSplitter = ospl

	return ac, nil
}

// create the overlord aggregator
func (acc *Accumulator) SetAggregateLoop(conf writers.WriterConfig) (agg *AggregateLoop, err error) {
	acc.Aggregators, err = NewAggregateLoop(acc.FlushTimes, acc.TTLTimes, acc.Name)
	acc.Aggregators.flush_random_ticker = acc.RandomTickerStart

	if err != nil {
		return nil, err
	}
	err = acc.Aggregators.SetWriter(conf)
	if err != nil {
		return nil, err
	}
	return acc.Aggregators, nil

}

func (acc *Accumulator) SetOutputQueue(qu chan splitter.SplitItem) {
	acc.OutputQueue = qu
}

func (acc *Accumulator) ProcessSplitItem(sp splitter.SplitItem) error {
	return acc.ProcessLine(sp.Line())
}

func (acc *Accumulator) ProcessLine(sp string) error {
	stats.StatsdClient.Incr("accumulator.lines.incoming", 1)
	acc.LineQueue <- sp
	return nil
}

// this is a helper function to get things to "start" on nicely "rounded"
// ticker intervals .. i.e. if  duration is 5 seconds .. start on t % 5
func (acc *Accumulator) delayRoundedTicker(duration time.Duration) *time.Ticker {
	//time.Sleep(time.Duration(durSec - (time.Now().UnixNano() % durSec)))
	time.Sleep(time.Now().Truncate(duration).Add(duration).Sub(time.Now()))
	return time.NewTicker(duration)
}

// start the flusher at the time interval
// best to call this in a go routine
func (acc *Accumulator) Start() error {
	if acc.shutitdown {
		acc.log.Warning("Shutting down, will not start `%s`", acc.Name)
		return nil
	}

	if acc.timer == nil {
		// make sure to set the proper resolution in the
		// aggregator matters for those things that have time components on the incoming
		// graphite/carbon
		acc.Accumulate.SetResolution(acc.AccumulateTime)

		if acc.RandomTickerStart {
			acc.log.Notice(
				"Accumulator Loop for %s at random start .. starting: %d",
				acc.AccumulateTime.String(),
				time.Now().Unix(),
			)
			acc.timer = time.NewTicker(acc.AccumulateTime)
		} else {
			acc.log.Notice(
				"Aggregater Loop for %s starting at time %% %s .. starting %d",
				acc.AccumulateTime.String(),
				acc.AccumulateTime.String(),
				time.Now().Unix(),
			)
			acc.timer = acc.delayRoundedTicker(acc.AccumulateTime)
		}
	}
	if acc.LineQueue == nil {
		acc.LineQueue = make(chan string, 10000)
	}

	acc.log.Notice("Starting accumulator loop for `%s`", acc.Name)

	// fire up Aggs
	if acc.Aggregators != nil {
		acc.log.Notice("Starting aggregator loop for `%s`", acc.Name)
		go acc.Aggregators.Start()
	}

	for {

		select {
		case line := <-acc.LineQueue:
			acc.Accumulate.ProcessLine(line)
			stats.StatsdClient.Incr("accumulator.lines.processed", 1)
		case dd := <-acc.timer.C:
			acc.log.Debug("Flushing accumulator %s to: %s at: %v", acc.Name, acc.ToBackend, dd.Unix())
			go func() { acc.FlushAndPost(dd) }()
		case <-acc.Shutdown:
			acc.timer.Stop()
			acc.log.Warning("Shutting down final flush of accumulator `%s`", acc.Name)
			acc.FlushAndPost(time.Now())
			if acc.Aggregators != nil {
				acc.Aggregators.Stop()
			}
			return nil
		}
	}

	close(acc.LineQueue)
	acc.LineQueue = nil
	return nil
}

func (acc *Accumulator) Stop() {
	acc.log.Warning("Initiating shutdown of accumulator `%s`", acc.Name)
	go func() {
		acc.shutitdown = true
		acc.Shutdown <- true
		return
	}()
	return
}

// move back into Main Server loop
func (acc *Accumulator) PushLine(spl splitter.SplitItem) {
	if acc.OutputQueue != nil && acc.ToBackend != BLACK_HOLE_BACKEND {
		acc.OutputQueue <- spl
	}
}

// move into Aggregator land
func (acc *Accumulator) PushStat(spl repr.StatRepr) {
	stats.StatsdClient.Incr("accumulator.stats.outgoing", 1)
	acc.Aggregators.InputChan <- spl
}

func (acc *Accumulator) FlushAndPost(attime time.Time) ([]splitter.SplitItem, error) {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("accumulator.flushpost-time-ns"), time.Now())
	items := acc.Accumulate.Flush()
	//log.Notice("Flush: %s", items)
	//return []splitter.SplitItem{}, nil
	//t := time.Now()
	out_spl := make([]splitter.SplitItem, len(items.Lines), len(items.Lines))
	for idx, item := range items.Lines {
		spl, err := acc.OutSplitter.ProcessLine(item)
		if err != nil {
			acc.log.Error("Invalid Line post flush accumulate `%s` Err:%s", item, err)
			continue
		}
		// this tells the server backends to NOT send to the accumulator anymore
		// otherwise we'd get serious infinite channel loops
		//log.Warning("ACC posted: %v  Len %d", spl.Line(), acc.OutputQueue)
		spl.SetPhase(splitter.AccumulatedParsed)
		spl.SetOrigin(splitter.Other)
		spl.SetOriginName(acc.FromBackend) // where we are from
		out_spl[idx] = spl
		//log.Notice("sending: %s Len:%d", spl.Line(), len(acc.OutputQueue))
		acc.PushLine(spl)
	}

	if acc.Aggregators != nil {
		for _, stat := range items.Stats {
			if stat.Time.IsZero() {
				stat.Time = attime // need to set this as this is the flush time
			}
			acc.PushStat(stat)
		}
		acc.log.Debug("Aggregator Flush: `%s` to `%s` Lines: %d", acc.Name, acc.Aggregators.Name, len(items.Stats))
	}
	stats.StatsdClientSlow.Incr("accumulator.flushesposts", 1)

	acc.log.Debug("Flushed accumulator `%s` to Backend: `%s` Lines: %d", acc.Name, acc.ToBackend, len(out_spl))
	items = nil // GC me
	return out_spl, nil
}

// flush out the accumulator, and "reparse" the lines
func (acc *Accumulator) Flush() ([]splitter.SplitItem, error) {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("accumulator.flush-time-ns"), time.Now())

	items := acc.Accumulate.Flush()
	var out_spl []splitter.SplitItem
	for _, item := range items.Lines {
		spl, err := acc.OutSplitter.ProcessLine(item)
		if err != nil {
			acc.log.Error("Invalid Line post flush accumulate `%s` Err:%s", item, err)
			continue
		}
		// this tells the server backends to NOT send to the accumulator anymore
		// otherwise we'd get serious infinite channel loops
		spl.SetPhase(splitter.AccumulatedParsed)
		spl.SetOrigin(splitter.Other)
		out_spl = append(out_spl, spl)
	}
	items = nil // GC me
	stats.StatsdClientSlow.Incr("accumulator.flushes", 1)
	return out_spl, nil
}

func (acc *Accumulator) LogConfig() {
	acc.log.Debug(" - Accumulator Group: `%s`", acc.Name)
	acc.log.Debug("   - Delgateing to Backend: `%s`", acc.ToBackend)
	acc.log.Debug("   - Accumulator Output format:: `%s`", acc.Formatter.Type())
	acc.log.Debug("   - Accumulator Type:: `%s`", acc.Accumulate.Name())
	acc.log.Debug("   - Accumulator FlushTime:: `%v`", acc.FlushTimes)
	acc.log.Debug("   - Accumulator KeepKeys:: `%v`", acc.KeepKeys)
	if acc.Aggregators != nil {
		acc.log.Debug("   - Accumulator Aggregator:: `%v`", acc.Aggregators.Name)
	}
}

// just grab whats currently in the queue to be flushed
// this is so we can simply "look" into the accumulator from another source
// (i.e. our monitor)

func (acc *Accumulator) CurrentStats() *repr.ReprList {
	s_rep := new(repr.ReprList)
	stats := acc.Accumulate.Stats()
	t := time.Now()

	for idx, stat := range stats {
		rr := stat.Repr()
		rr.StatKey = idx
		rr.Time = t
		rr.Resolution = acc.FlushTimes[0].Seconds()
		s_rep.Add(rr)
	}
	return s_rep
}
