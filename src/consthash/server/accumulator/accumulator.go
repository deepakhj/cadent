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
    To
*/

package accumulator

import (
	splitter "consthash/server/splitter"
	"fmt"
	"sync"
	"time"
)

/**** main accumulator object */

type Accumulator struct {

	// these are assigned from the config file in the PreReg config file
	ToBackend       string        `json:"backend"`
	FormatterName   string        `json:"formatter"`
	AccumulatorName string        `json:"accumulator"`
	Name            string        `json:"name"`
	KeepKeys        bool          `json:"keep_keys"` // if true, will not "remove" the keys post flush, just set them to 0
	FlushTime       time.Duration `json:"flush_time"`

	Accumulate AccumulatorItem
	Formatter  FormatterItem

	InSplitter  splitter.Splitter
	OutSplitter splitter.Splitter

	mu          sync.Mutex
	timer       *time.Ticker
	LineQueue   chan string
	OutputQueue chan splitter.SplitItem
	Shutdown    chan bool
}

func NewAccumlator(inputtype string, outputtype string, keepkeys bool) (*Accumulator, error) {

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

	ac := &Accumulator{
		Accumulate:      acc,
		Formatter:       fmter,
		AccumulatorName: inputtype,
		FormatterName:   outputtype,
		KeepKeys:        keepkeys,
		Name:            fmt.Sprintf("%s -> %s", inputtype, outputtype),
		FlushTime:       time.Second,
		Shutdown:        make(chan bool, 1),
		LineQueue:       make(chan string, 10000),
		timer:           nil,
	}

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

func (acc *Accumulator) SetOutputQueue(qu chan splitter.SplitItem) {
	acc.OutputQueue = qu
}

func (acc *Accumulator) ProcessSplitItem(sp splitter.SplitItem) error {
	return acc.ProcessLine(sp.Line())
}

func (acc *Accumulator) ProcessLine(sp string) error {
	acc.LineQueue <- sp
	return nil
}

// start the flusher at the time interval
// best to call this in a go routine
func (acc *Accumulator) Start() error {
	acc.mu.Lock()
	if acc.timer == nil {
		acc.timer = time.NewTicker(acc.FlushTime)
	}
	if acc.LineQueue == nil {
		acc.LineQueue = make(chan string, 10000)
	}
	acc.mu.Unlock()
	log.Notice("Starting accumulator loop for `%s`", acc.Name)

	for {
		select {
		case <-acc.timer.C:
			log.Debug("Flushing accumulator %s", acc.Name)
			acc.FlushAndPost()
		case line := <-acc.LineQueue:
			acc.Accumulate.ProcessLine(line)

		case <-acc.Shutdown:
			acc.timer.Stop()
			log.Notice("Shutting down final flush of accumulator `%s`", acc.Name)
			acc.FlushAndPost()
			break
		}
	}

	//bleed
	for {
		for i := 0; i < len(acc.LineQueue); i++ {
			_ = <-acc.LineQueue
		}
		if len(acc.LineQueue) == 0 {
			break
		}
	}
	close(acc.LineQueue)
	acc.LineQueue = nil
	return nil
}

func (acc *Accumulator) Stop() {
	log.Notice("Initiating shutdown of accumulator `%s`", acc.Name)
	acc.Shutdown <- true
	return
}

func (acc *Accumulator) FlushAndPost() ([]splitter.SplitItem, error) {
	items := acc.Accumulate.Flush()
	//log.Notice("Flush: %s", items)
	//return []splitter.SplitItem{}, nil
	var out_spl []splitter.SplitItem
	for _, item := range items {
		spl, err := acc.OutSplitter.ProcessLine(item)
		if err != nil {
			log.Error("Invalid Line post flush accumulate `%s` Err:%s", item, err)
			continue
		}
		// this tells the server backends to NOT send to the accumulator anymore
		// otherwise we'd get serious infinite channel loops
		//log.Warning("ACC posted: %v  Len %d", spl.Line(), acc.OutputQueue)
		spl.SetPhase(splitter.AccumulatedParsed)
		spl.SetOrigin(splitter.Other)
		out_spl = append(out_spl, spl)
		//log.Notice("sending: %s Len:%d", spl.Line(), len(acc.OutputQueue))
		acc.OutputQueue <- spl
	}
	log.Debug("Flushed accumulator %s Lines: %d", acc.Name, len(out_spl))

	return out_spl, nil
}

// flush out the accumulator, and "reparse" the lines
func (acc *Accumulator) Flush() ([]splitter.SplitItem, error) {
	items := acc.Accumulate.Flush()
	var out_spl []splitter.SplitItem
	for _, item := range items {
		spl, err := acc.OutSplitter.ProcessLine(item)
		if err != nil {
			log.Error("Invalid Line post flush accumulate `%s` Err:%s", item, err)
			continue
		}
		// this tells the server backends to NOT send to the accumulator anymore
		// otherwise we'd get serious infinite channel loops
		spl.SetPhase(splitter.AccumulatedParsed)
		spl.SetOrigin(splitter.Other)
		out_spl = append(out_spl, spl)
	}
	return out_spl, nil
}

func (acc *Accumulator) LogConfig() {
	log.Debug(" - Accumulator Group: `%s`", acc.Name)
	log.Debug("   - Delgateing to Backend: `%s`", acc.ToBackend)
	log.Debug("   - Accumulator Output format:: `%s`", acc.Formatter.Type())
	log.Debug("   - Accumulator Type:: `%s`", acc.Accumulate.Name())
	log.Debug("   - Accumulator FlushTime:: `%v`", acc.FlushTime)
	log.Debug("   - Accumulator KeepKeys:: `%v`", acc.KeepKeys)
}

// just grab whats currently in the queue to be flushed
// this is so we can simply "look" into the accumulator from another sorce
// (i.e. our monitor)

func (acc *Accumulator) CurrentStats() []StatRepr {
	var s_rep []StatRepr
	stats := acc.Accumulate.Stats()
	t := time.Now().UnixNano()
	for idx, stat := range stats {
		rr := stat.Repr()
		rr.StatKey = idx
		rr.Time = t
		s_rep = append(s_rep, rr)
	}
	return s_rep
}
