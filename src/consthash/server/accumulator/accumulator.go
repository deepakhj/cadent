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
	FlushTime       time.Duration `json:"flush_time"`

	Accumulate AccumulatorItem
	Formatter  FormatterItem

	Splitter splitter.Splitter

	mu          sync.Mutex
	timer       *time.Timer
	OutputQueue chan splitter.SplitItem
	Shutdown    chan bool
}

func NewAccumlator(formatter string, accume_name string) (*Accumulator, error) {

	fmter, err := NewFormatterItem(formatter)
	if err != nil {
		return nil, err
	}
	fmter.Init()

	acc, err := NewAccumulatorItem(accume_name)
	if err != nil {
		return nil, err
	}
	acc.Init(fmter)

	ac := &Accumulator{
		Accumulate:      acc,
		Formatter:       fmter,
		AccumulatorName: accume_name,
		FormatterName:   formatter,
		Name:            fmt.Sprintf("<accumulator> %s -> %s", accume_name, formatter),
		FlushTime:       time.Second,
	}

	// determine the splitter from the formatter item
	nul_conf := make(map[string]interface{})
	spl, err := splitter.NewSplitterItem(formatter, nul_conf)
	if err != nil {
		return nil, err
	}
	ac.Splitter = spl

	return ac, nil
}

func (acc *Accumulator) ProcessSplitItem(sp splitter.SplitItem) error {
	return acc.Accumulate.ProcessLine(sp.Line())
}

func (acc *Accumulator) ProcessLine(sp string) error {
	return acc.Accumulate.ProcessLine(sp)
}

// start the flusher at the time interval
// best to call this in a go routine
func (acc *Accumulator) Start() error {
	acc.mu.Lock()
	if acc.timer != nil {
		return fmt.Errorf("Accumulator %s already starter", acc.Name)
	}
	acc.timer = time.NewTimer(acc.FlushTime)
	acc.Shutdown = make(chan bool)
	acc.mu.Unlock()

	do_flush := func() {
		data, err := acc.Flush()
		if err != nil {
			log.Error("Flushing accumulator FAILED %s", err)
			return
		}
		for _, item := range data {
			acc.OutputQueue <- item
		}
	}
	for {
		select {
		case <-acc.timer.C:
			log.Debug("Flushing accumulator %s", acc.Name)
			do_flush()
		case <-acc.Shutdown:
			log.Info("Shutting down final flush of accumulator %s", acc.Name)
			do_flush()
			acc.timer = nil
			close(acc.Shutdown)
			return nil
		}
	}
	return nil
}

func (acc *Accumulator) Stop() {
	if acc.timer != nil {
		acc.Shutdown <- true
	}
}

// flush out the accumulator, and "reparse" the lines
func (acc *Accumulator) Flush() ([]splitter.SplitItem, error) {
	items := acc.Accumulate.Flush()
	var out_spl []splitter.SplitItem
	for _, item := range items {
		spl, err := acc.Splitter.ProcessLine(item)
		if err != nil {
			log.Error("Invalid Line post flush accumulate `%s`", item)
			continue
		}
		// this tells the server backends to NOT send to the accumulator anymore
		// otherwise we'd get serious infinite channel loops
		spl.SetPhase(splitter.AccumulatedParsed)
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
}
