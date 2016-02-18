package accumulator

/*

Accumulator TOML configgy helper

For each accumulator there are 3 basic items to config

 InputFormat (statsd, graphite, etc)
 OutputFormat (statsd, graphite, etc)
 Accumulator Tick (time between flushing the statss)
 OutPutBackend (the name of the PreReg backend to send the flush results to)

This config object is currently used in the "PreReg" world to configure
accumulators to backends

[accumulator]
input_format="statsd"
output_format="graphite"
to_prereg_group="graphite-proxy"
keep_keys = true

[[accumulator.tags]]
key="moo"
value="goo"
[[accumulator.tags]]
key="foo"
value="bar"

# external writer
[accumulator.writer]
driver = "file"
dsn = "/tmp/moo"
 	[accumulator.writer.options]
    max_file_size=10000

# aggregate bin counts
[accumulator.keeper]
times = ["5s", "1m", "1h"]


*/

import (
	"fmt"
	"github.com/BurntSushi/toml"
	logging "github.com/op/go-logging"
	"math"
	"time"
)

const DEFALT_SECTION_NAME = "accumulator"

var log = logging.MustGetLogger("accumulator")

// the accumulator can write to other "socket based" backends if desired
type AccumulatorWriter struct {
	Driver  string                 `toml:"driver"`
	DSN     string                 `toml:"dsn"`
	Options map[string]interface{} `toml:"options"` // option=[ [key, value], [key, value] ...]
}

// eventhough we "flush" every "X" seconds, we may want another level or 2 of flushers
// which means we "force" keep_keys and have N extra lists of aggrigated stats
type AccumulatorKeeper struct {
	Times     []string `toml:"times"`
	Durations []time.Duration
}

// for tagging things if the formatter supports them (influx, or other)
type AccumulatorTags struct {
	Key   string `toml:"key"`
	Value string `toml:"value"`
}

// options for backends

type ConfigAccumulator struct {
	ToBackend    string            `toml:"backend"` // once "parsed and flushed" re-inject into another pre-reg group for delegation
	InputFormat  string            `toml:"input_format"`
	OutoutFormat string            `toml:"output_format"`
	KeepKeys     bool              `toml:"keep_keys"` // keeps the keys on flush  "0's" them rather then removal
	Option       [][]string        `toml:"options"`   // option=[ [key, value], [key, value] ...]
	Tags         []AccumulatorTags `toml:"tags"`
	Writer       AccumulatorWriter `toml:"writer"`
	Keeper       AccumulatorKeeper `toml:"keeper"`
}

func (cf ConfigAccumulator) GetAccumulator() (*Accumulator, error) {

	ac, err := NewAccumlator(cf.InputFormat, cf.OutoutFormat, cf.KeepKeys)
	if err != nil {
		log.Critical("%s", err)
		return nil, err
	}

	if len(cf.ToBackend) == 0 {
		return nil, fmt.Errorf("Need a `backend` for post delegation")
	}
	ac.ToBackend = cf.ToBackend
	ac.Accumulate.SetTags(cf.Tags)
	err = ac.Accumulate.SetOptions(cf.Option)
	if err != nil {
		return nil, err
	}

	if len(cf.Keeper.Times) > 0 {
		last_keeper := time.Duration(0)
		for _, st := range cf.Keeper.Times {
			t_dur, err := time.ParseDuration(st)
			if err != nil {
				return nil, err
			}
			// need to be properly time ordered and MULTIPLES of each other
			if last_keeper.Seconds() > 0 {
				if t_dur.Seconds() < last_keeper.Seconds() {
					return nil, fmt.Errorf("Keeper Durations need to be in smallest to longest order")
				}
				if math.Mod(float64(t_dur.Seconds()), float64(last_keeper.Seconds())) != 0.0 {
					return nil, fmt.Errorf("Keeper Durations need to be in multiples of themselves")
				}
			}
			last_keeper = t_dur
			cf.Keeper.Durations = append(cf.Keeper.Durations, t_dur)
		}
		ac.FlushTimes = cf.Keeper.Durations

	} else {
		ac.FlushTimes = []time.Duration{5 * time.Second}
	}

	// set up the writer it needs to be done AFTER the Keeper times are verified
	if cf.Writer.Driver != "" {
		_, err = ac.SetAggregateLoop(cf.Writer)
		if err != nil {
			return nil, err
		}
	}

	return ac, nil
}

func ParseConfigString(inconf string) (*Accumulator, error) {
	cf := new(ConfigAccumulator)
	if _, err := toml.Decode(inconf, cf); err != nil {
		log.Critical("Error decoding config string: %s", err)
		return nil, err
	}
	return cf.GetAccumulator()
}
