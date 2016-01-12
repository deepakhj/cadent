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
flush_time="5s"
to_prereg_group="graphite-proxy"

[[accumulator.tags]]
key="moo"
value="goo"
[[accumulator.tags]]
key="foo"
value="bar"

*/

import (
	"fmt"
	"github.com/BurntSushi/toml"
	logging "github.com/op/go-logging"
	"time"
)

const DEFALT_SECTION_NAME = "accumulator"

var log = logging.MustGetLogger("accumulator")

// for tagging things if the formatter supports them (influx, or other)
type AccumulatorTags struct {
	Key   string `toml:"key"`
	Value string `toml:"value"`
}

type ConfigAccumulator struct {
	ToBackend    string            `toml:"backend"` // once "parsed and flushed" re-inject into another pre-reg group for delegation
	InputFormat  string            `toml:"input_format"`
	OutoutFormat string            `toml:"output_format"`
	FlushTime    string            `toml:"flush_time"`
	Tags         []AccumulatorTags `toml:"tags"`
}

func (cf ConfigAccumulator) GetAccumulator() (*Accumulator, error) {

	ac, err := NewAccumlator(cf.InputFormat, cf.OutoutFormat)
	if err != nil {
		log.Critical("%s", err)
		return nil, err
	}
	if cf.FlushTime == "" {
		cf.FlushTime = "1s"
	}
	dur, err := time.ParseDuration(cf.FlushTime)
	if err != nil {
		log.Critical("%s", err)
		return nil, err
	}
	ac.FlushTime = dur
	if len(cf.ToBackend) == 0 {
		return nil, fmt.Errorf("Need a `backend` for post delegation")
	}
	ac.ToBackend = cf.ToBackend
	ac.Accumulate.SetTags(cf.Tags)
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
