/*
   Writers/Readers of stats

   We are attempting to mimic the Graphite API json blobs throughout the process here
   such that we can hook this in directly to either graphite-api/web

   NOTE: this is not a full graphite DSL, just paths and metrics, we leave the fancy functions inside
   graphite-api/web to work their magic .. one day we'll impliment the full DSL, but until then ..
*/

package readers

import (
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"strconv"
	"strings"
	"time"
)

var log = logging.MustGetLogger("readers")

/****************** Output structs *********************/

// the basic metric json blob for find
type MetricFindItem struct {
	Text          string `json:"text"`
	Expandable    int    `json:"expandable"`
	Leaf          int    `json:"leaf"`
	Id            string `json:"id"`
	AllowChildren int    `json:"allowChildren"`
}

type MetricFindItems []MetricFindItem

// the basic metric json blob for find
type MetricExpandItem struct {
	Results []string `json:"results"`
}

type DataPoint struct {
	Time  int
	Value float64
}

func (s DataPoint) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%f, %d]", s.Value, s.Time)), nil
}

// the basic metric json blob for find
type RenderItem struct {
	Target     string      `json:"target"`
	Datapoints []DataPoint `json:"datapoints"`
}

type RenderItems []RenderItem

/****************** Helpers *********************/
// parses things like "-23h" etc
// only does "second" precision which is all graphite can do currently
func ParseTime(st string) (int64, error) {
	st = strings.Trim(strings.ToLower(st), " \n\t")
	unix_t := int64(time.Now().Unix())
	if st == "now" {
		return unix_t, nil
	}

	if strings.HasSuffix(st, "s") || strings.HasSuffix(st, "sec") || strings.HasSuffix(st, "second") {
		items := strings.Split(st, "s")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i, nil
	}

	if strings.HasSuffix(st, "m") || strings.HasSuffix(st, "min") {
		items := strings.Split(st, "m")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60, nil
	}

	if strings.HasSuffix(st, "h") || strings.HasSuffix(st, "hour") {
		items := strings.Split(st, "h")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60, nil
	}

	if strings.HasSuffix(st, "d") || strings.HasSuffix(st, "day") {
		items := strings.Split(st, "d")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60*24, nil
	}
	if strings.HasSuffix(st, "y") || strings.HasSuffix(st, "year") {
		items := strings.Split(st, "y")
		i, err := strconv.ParseInt(items[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unix_t + i*60*60*24*365, nil
	}

	// if it's an int already, we're good
	i, err := strconv.ParseInt(st, 10, 64)
	if err != nil {
		return i, nil
	}

	return 0, fmt.Errorf("Time `%s` could not be parsed", st)

}

/****************** Data readers *********************/

type Reader interface {
	Config(map[string]interface{}) error

	// /metrics/find/?query=stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.*
	/*
			[
		{
		text: "accumulator",
		expandable: 1,
		leaf: 0,
		id: "stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.accumulator",
		allowChildren: 1
		}
		]
	*/
	Find(metric string) (MetricFindItems, error)

	// /metric/expand?query=stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.*
	/*
			{
		results: [
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.send",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.sent-bytes",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.valid-lines",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.valid-lines-sent-to-workers"
		]
		}
	*/
	Expand(metric string) (MetricExpandItem, error)

	// /render?target=XXXX&from=-24h&to=now
	/*
			{
		target: "scaleToSeconds(stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.writer.cassandra.noncached-writes-path,1)",
		datapoints: [
		[
		0,
		1456087430
		],...
		]}
	*/
	Render(path string, from string, to string) (RenderItems, error)
}
