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
   Base objects for our API responses and internal communications
*/

package metrics

import (
	"bytes"
	"cadent/server/repr"
	"cadent/server/series"
	"cadent/server/utils"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

const MAX_QUANTIZE_POINTS = 100000

/** errors **/
var errMergeStepSizeError = errors.New("To merge 2 RawRenderItems, the step size needs to be the same")
var errQuantizeTooLittleData = errors.New("Cannot quantize to step: too little data")
var errQuantizeStepTooSmall = fmt.Errorf("Cannot quantize: to a '0' step size ...")
var errQuantizeStepTooManyPoints = fmt.Errorf("Cannot quantize: Over the 10,000 point threashold, please try a smaller time window")

/******************  a simple union of series.TimeSeries and repr.StatName *********************/
type TotalTimeSeries struct {
	Name   *repr.StatName
	Series series.TimeSeries
}

/******************  structs for the raw Database query (for blob series only) *********************/

type DBSeries struct {
	Id         interface{} // don't really know what the ID format for a DB is
	Uid        string
	Start      int64
	End        int64
	Ptype      uint8
	Pbytes     []byte
	Resolution uint32
	TTL        uint32
}

func (d *DBSeries) Iter() (series.TimeSeriesIter, error) {
	s_name := series.NameFromId(d.Ptype)
	return series.NewIter(s_name, d.Pbytes)
}

type DBSeriesList []*DBSeries

func (mb DBSeriesList) Start() int64 {
	t_s := int64(0)
	for idx, d := range mb {
		if idx == 0 || (d.Start < t_s && d.Start != 0) {
			t_s = d.Start
		}
	}
	return t_s
}

func (mb DBSeriesList) End() int64 {
	t_s := int64(0)
	for idx, d := range mb {
		if idx == 0 || d.End > t_s {
			t_s = d.End
		}
	}
	return t_s
}

func (mb DBSeriesList) ToRawRenderItem() (*RawRenderItem, error) {
	rawd := new(RawRenderItem)
	for _, d := range mb {
		iter, err := d.Iter()
		if err != nil {
			return nil, err
		}
		for iter.Next() {
			to, mi, mx, ls, su, ct := iter.Values()
			t := uint32(time.Unix(0, to).Unix())

			rawd.Data = append(rawd.Data, RawDataPoint{
				Count: ct,
				Sum:   su,
				Max:   mx,
				Min:   mi,
				Last:  ls,
				Time:  t,
			})
		}
	}
	sort.Sort(rawd.Data)
	if len(rawd.Data) > 0 {
		rawd.RealStart = rawd.Data[0].Time
		rawd.Start = rawd.RealStart

		rawd.End = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealEnd = rawd.End
	}

	return rawd, nil

}

/****************** Output structs for the graphite API*********************/

type DataPoint struct {
	Time  uint32
	Value *float64 // need nils for proper "json none"
}

func NewDataPoint(time uint32, val float64) DataPoint {
	d := DataPoint{Time: time, Value: new(float64)}
	d.SetValue(&val)
	return d
}

func (d DataPoint) MarshalJSON() ([]byte, error) {
	if d.Value == nil || math.IsNaN(*d.Value) {
		return []byte(fmt.Sprintf("[null, %d]", d.Time)), nil
	}

	return []byte(fmt.Sprintf("[%f, %d]", *d.Value, d.Time)), nil
}
func (d DataPoint) SetValue(val *float64) {
	d.Value = val
}

// the basic metric json blob for find
type RenderItem struct {
	Target     string      `json:"target"`
	Datapoints []DataPoint `json:"datapoints"`
}

type RenderItems []RenderItem

// the basic whisper metric json blob for find

type WhisperRenderItem struct {
	RealStart uint32                 `json:"data_from"`
	RealEnd   uint32                 `json:"data_end"`
	Start     uint32                 `json:"from"`
	End       uint32                 `json:"to"`
	Step      uint32                 `json:"step"`
	Series    map[string][]DataPoint `json:"series"`
}

/****************** Output structs for the internal API*********************/

type RawDataPoint struct {
	Time  uint32  `json:"time"`
	Sum   float64 `json:"sum"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Last  float64 `json:"last"`
	Count int64   `json:"count"`
}

func (d RawDataPoint) floatToJson(buf *bytes.Buffer, name string, end string, val float64) {
	if math.IsNaN(val) {
		fmt.Fprintf(buf, name+":null"+end)
		return
	}
	fmt.Fprintf(buf, name+":%f"+end, val)
}

func (d RawDataPoint) MarshalJSON() ([]byte, error) {
	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)

	fmt.Fprintf(buf, "{\"time\": %d,", d.Time)
	d.floatToJson(buf, "\"sum\"", repr.COMMA_SEPARATOR, d.Sum)
	d.floatToJson(buf, "\"min\"", repr.COMMA_SEPARATOR, d.Min)
	d.floatToJson(buf, "\"max\"", repr.COMMA_SEPARATOR, d.Max)
	d.floatToJson(buf, "\"last\"", repr.COMMA_SEPARATOR, d.Last)
	fmt.Fprintf(buf, "\"count\": %d}", d.Count)
	return buf.Bytes(), nil
}

func NullRawDataPoint(time uint32) RawDataPoint {
	return RawDataPoint{
		Time:  time,
		Sum:   math.NaN(),
		Min:   math.NaN(),
		Max:   math.NaN(),
		Last:  math.NaN(),
		Count: 0,
	}
}

func (r *RawDataPoint) IsNull() bool {
	return math.IsNaN(r.Sum) && math.IsNaN(r.Last) && math.IsNaN(r.Min) && math.IsNaN(r.Max)
}

func (r *RawDataPoint) String() string {
	return fmt.Sprintf("RawDataPoint: T: %d Mean: %f", r.Time, r.AggValue(repr.MEAN))
}

func (r *RawDataPoint) AggValue(aggfunc repr.AggType) float64 {

	// if the count is 1 there is only but one real value
	if r.Count == 1 {
		return r.Sum
	}

	switch aggfunc {
	case repr.SUM:
		return r.Sum
	case repr.MIN:
		return r.Min
	case repr.MAX:
		return r.Max
	case repr.LAST:
		return r.Last
	case repr.COUNT:
		return float64(r.Count)
	default:
		if r.Count > 0 {
			return r.Sum / float64(r.Count)
		}
		return math.NaN()
	}
}

// merge two data points into one .. this is a nice merge that will add counts, etc to the pieces
// the "time" is then the greater of the two
func (r *RawDataPoint) Merge(d *RawDataPoint) {
	if d.IsNull() {
		return // just skip as nothing to merge
	}
	if math.IsNaN(r.Max) || r.Max < d.Max {
		r.Max = d.Max
	}
	if math.IsNaN(r.Min) || r.Min > d.Min {
		r.Min = d.Min
	}

	if math.IsNaN(r.Sum) && !math.IsNaN(d.Sum) {
		r.Sum = d.Sum
	} else if !math.IsNaN(r.Sum) && !math.IsNaN(d.Sum) {
		r.Sum += d.Sum
	}

	if d.Time != 0 && d.Time > r.Time && !math.IsNaN(d.Last) {
		r.Last = d.Last
	}
	if math.IsNaN(r.Last) && !math.IsNaN(d.Last) {
		r.Last = d.Last
	}

	if r.Count == 0 && d.Count > 0 {
		r.Count = d.Count
	} else if r.Count != 0 && d.Count != 0 {
		r.Count += d.Count
	}

	if r.Time < d.Time {
		r.Time = d.Time
	}

}

type RawDataPointList []RawDataPoint

func (v RawDataPointList) Len() int           { return len(v) }
func (v RawDataPointList) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v RawDataPointList) Less(i, j int) bool { return v[i].Time < v[j].Time }

type RawRenderItem struct {
	Metric    string           `json:"metric"`
	Id        string           `json:"id"`
	Tags      repr.SortingTags `json:"tags"`
	MetaTags  repr.SortingTags `json:"meta_tags"`
	RealStart uint32           `json:"data_from"`
	RealEnd   uint32           `json:"data_end"`
	Start     uint32           `json:"from"`
	End       uint32           `json:"to"`
	Step      uint32           `json:"step"`
	AggFunc   repr.AggType     `json:"aggfunc"`
	Data      RawDataPointList `json:"data"`
}

func NewRawRenderItemFromSeriesIter(iter series.TimeSeriesIter) (*RawRenderItem, error) {
	rawd := new(RawRenderItem)
	for iter.Next() {
		to, mi, mx, ls, su, ct := iter.Values()
		t := uint32(time.Unix(0, to).Unix())

		rawd.Data = append(rawd.Data, RawDataPoint{
			Count: ct,
			Sum:   su,
			Max:   mx,
			Min:   mi,
			Last:  ls,
			Time:  t,
		})
	}
	if len(rawd.Data) > 0 {
		sort.Sort(rawd.Data)

		rawd.RealStart = rawd.Data[0].Time
		rawd.Start = rawd.RealStart

		rawd.End = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealEnd = rawd.End
	}

	return rawd, nil
}

func NewRawRenderItemFromSeries(ts *TotalTimeSeries) (*RawRenderItem, error) {
	s_iter, err := ts.Series.Iter()
	if err != nil {
		return nil, err
	}
	rawd, err := NewRawRenderItemFromSeriesIter(s_iter)
	if err != nil {
		return nil, err
	}

	rawd.Id = ts.Name.UniqueIdString()
	rawd.MetaTags = ts.Name.MetaTags
	rawd.Tags = ts.Name.Tags
	rawd.Metric = ts.Name.Key
	rawd.Step = ts.Name.Resolution

	return rawd, nil
}

func (r *RawRenderItem) Len() int {
	return len(r.Data)
}

func (r *RawRenderItem) String() string {
	return fmt.Sprintf("RawRenderItem: Start: %d End: %d, Points: %d", r.Start, r.End, r.Len())
}

func (r *RawRenderItem) PrintPoints() {
	fmt.Printf("RawRenderItem: %s (%s) Start: %d End: %d, Points: %d\n", r.Metric, r.Id, r.Start, r.End, r.Len())
	for idx, d := range r.Data {
		fmt.Printf("%d: %d %f %f %f %f %d\n", idx, d.Time, d.Min, d.Max, d.Last, d.Sum, d.Count)
	}
}

// is True if the start and end times are contained in this data blob
func (r *RawRenderItem) DataInRange(start uint32, end uint32) bool {
	return r.RealStart >= start && end <= r.RealEnd
}

func (r *RawRenderItem) ToDataPoint() []DataPoint {
	dpts := make([]DataPoint, len(r.Data), len(r.Data))
	for idx, d := range r.Data {
		dpts[idx].Time = d.Time
		use_v := d.AggValue(r.AggFunc)
		dpts[idx].SetValue(&use_v)
	}
	return dpts
}

// somethings like read caches, are hot data we most of the time
// only need to make sure the "start" is in range as the end may not have happened yet
func (r *RawRenderItem) StartInRange(start uint32) bool {
	return start >= r.RealStart
}

// trim the data array so that it fit's w/i the start and end times
func (r *RawRenderItem) TrunctateTo(start uint32, end uint32) int {
	data := make([]RawDataPoint, 0)
	for _, d := range r.Data {
		if d.Time >= start && d.Time <= end {
			data = append(data, d)
		}
	}
	r.Data = data
	d_l := len(r.Data)
	if d_l > 0 {
		r.RealEnd = data[len(data)-1].Time
		r.RealStart = data[0].Time
	}
	return len(data)
}

// a crude "resampling" of incoming data .. if the incoming is non-uniform in time
// and we need to smear the resulting set into a uniform vector
//
// If moving from a "large" time step to a "smaller" one you WILL GET NULL values for
// time slots that do not match anything .. we cannot (will not) interpolate data like that
// as it's 99% statistically "wrong"
// this will "quantize" things the Start/End set times, not the RealStart/End
//
func (r *RawRenderItem) ResampleAndQuantize(step uint32) {

	cur_len := uint32(len(r.Data))
	// nothing we can do here
	if cur_len <= 1 {
		return
	}

	// figure out the lengths for the new vector (note the difference in this
	// start time from the Quantize .. we need to go below things
	start := r.Start
	left := r.Start % step
	if left != 0 {
		start = r.Start - step + left
	}

	endTime := (r.End - step) - ((r.End - step) % step)

	if endTime < start {
		// basically the resampling makes "one" data point so we merge all the
		// incoming points into this one
		endTime = start + step
		data := make([]RawDataPoint, 1)
		data[0] = NullRawDataPoint(start)
		for _, d := range r.Data {
			data[0].Merge(&d)
		}
		r.Start = start
		r.RealStart = start
		r.Step = step
		r.End = endTime
		r.RealEnd = endTime
		r.Data = data
		return
	}

	// data length of the new data array
	data := make([]RawDataPoint, (endTime-start)/step+1)

	//log.Error("RESAMPLE\n\n")
	// 'i' iterates Original Data.
	// 'o' iterates OutGoing Data.
	// t is the current time we need to merge
	for t, i, o := start, uint32(0), -1; t <= endTime; t += step {
		o++

		// start at null
		if !data[o].IsNull() {
			data[o] = NullRawDataPoint(t)
		}

		// past any valid points
		if i >= cur_len {
			data[o] = NullRawDataPoint(t)
			continue
		}
		// need to collect all the points that fit into the step
		// remembering that the "first" bin may include points "behind" it (by a step) from the original list
		// list
		p := r.Data[i]
		//log.Errorf("Resample: %d: cur T: %d to T: %d -- DataP: %v", i, t, (t + step), r.Data[i])

		// if the point is w/i [start-step, start)
		// then these belong to the current Bin
		if p.Time < start && p.Time >= (t-step) && p.Time < t {
			if data[o].IsNull() {
				data[o] = p
			} else {
				data[o].Merge(&p)
			}
			data[o].Time = t
			i++
			p = r.Data[i]
		}

		// if the points is w/i [t, t+step)
		// grab all the points in the step range and "merge"
		if p.Time >= t && p.Time < (t+step) {
			//log.Errorf("Start Merge FP: cur T: %d to T: %d -- DataP: %v :: [t, t+step): %v [t-step, t): %v ", t, (t + step), p, p.Time >= t && p.Time < (t+step), p.Time >= (t-step) && p.Time < t)
			// start at current point and merge up
			if data[o].IsNull() {
				data[o] = p
			} else {
				data[o].Merge(&p)
			}

			//log.Errorf("Start Merge FP: cur T: %d to T: %d -- DataP: %v", t, (t + step), p)
			for {
				i++
				if i >= cur_len {
					break
				}
				np := r.Data[i]
				if np.Time >= (t + step) {
					break
				}
				//log.Errorf("Merging: cur T: %d to T: %d -- DataP: %v", t, (t + step), np)
				data[o].Merge(&np)
			}
			data[o].Time = t
		}
	}

	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	r.Data = data
}

// reample but "skip" and nils
func (r *RawRenderItem) Resample(step uint32) error {
	// based on the start, stop and step.  Fill in the gaps in missing
	// slots (w/ nulls) as graphite does not like "missing times" (expects things to have a constant
	// length over the entire interval)
	// You should Put in an "End" time of "ReadData + Step" to avoid loosing the last point as things
	// are  [Start, End) not [Start, End]

	if step <= 0 {
		return errQuantizeStepTooSmall
	}

	// make sure the start/ends are nicely divisible by the Step
	start := r.RealStart

	left := start % step
	if left != 0 {
		start = start + step - left
	}

	end := r.RealEnd

	endTime := (end - 1) - ((end - 1) % step) + step

	// make sure in time order
	sort.Sort(r.Data)

	// 'i' iterates Original Data.
	// 'o' iterates Incoming Data.
	// 'n' iterates New Data.
	// t is the current time we need to fill/merge
	var t uint32
	var i, n int

	i_len := len(r.Data)
	data := make([]RawDataPoint, 0)

	for t, i, n = start, 0, -1; t <= endTime; t += step {
		dp := NullRawDataPoint(t)
		// loop through the orig data until we hit a time > then the current one
		for ; i < i_len; i++ {
			if r.Data[i].Time > t {
				break
			}
			dp.Merge(&r.Data[i])
		}

		if !dp.IsNull() {
			data = append(data, dp)
			n++
		}
	}

	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	r.Data = data
	return nil

}

func (r *RawRenderItem) Quantize() error {
	return r.QuantizeToStep(r.Step)
}

// based on the start, stop and step.  Fill in the gaps in missing
// slots (w/ nulls) as graphite does not like "missing times" (expects things to have a constant
// length over the entire interval)
// You should Put in an "End" time of "ReadData + Step" to avoid loosing the last point as things
// are  [Start, End) not [Start, End]
func (r *RawRenderItem) QuantizeToStep(step uint32) error {

	if step <= 0 {
		return errQuantizeStepTooSmall
	}

	// make sure the start/ends are nicely divisible by the Step
	start := r.Start
	left := r.Start % step
	if left != 0 {
		start = r.Start + step - left
	}

	endTime := (r.End - 1) - ((r.End - 1) % step) + step

	// make sure in time order
	sort.Sort(r.Data)

	// basically all the data fits into one point
	if endTime < start {
		data := make([]RawDataPoint, 1)
		r.Step = step
		r.Start = start
		r.End = start + step
		r.RealEnd = start + step
		r.RealStart = start
		data[0] = NullRawDataPoint(start)
		for _, d := range r.Data {
			data[0].Merge(&d)

		}
		r.Data = data
		return nil
	}

	// trap too many points
	num_pts := ((endTime - start) / step) + 1
	if num_pts >= MAX_QUANTIZE_POINTS {
		return errQuantizeStepTooManyPoints
	}

	// data length of the new data array
	data := make([]RawDataPoint, num_pts)
	cur_len := uint32(len(r.Data))

	// 'i' iterates Original Data.
	// 'o' iterates OutGoing Data.
	// t is the current time we need to fill/merge
	var t, i uint32
	var o int32
	for t, i, o = start, uint32(0), -1; t <= endTime; t += step {
		o += 1

		// No more data in the original list
		if i >= cur_len {
			data[o] = NullRawDataPoint(t)
			continue
		}

		p := r.Data[i]
		if p.Time == t {
			// perfect match
			data[o] = p
			i++
		} else if p.Time > t {
			// data is too recent, so we need to "skip" the slot and move on
			// unless it as merged already
			if data[o].Time == 0 {
				data[o] = NullRawDataPoint(t)
			}

			if p.Time >= endTime {
				data[o].Merge(&p)
			}

		} else if p.Time > t-step && p.Time < t {
			// data fits in a slot,
			// but may need a merge w/ another point(s) in the parent list
			// so we advance "i"
			p.Time = t
			if data[o].Time != 0 && !data[0].IsNull() {
				data[o].Merge(&p)
			} else {
				data[o] = p
			}
			i++
		} else if p.Time <= t-step {
			// point is too old. move on until we find one that's not
			// but we need to redo the above logic to put it in the proper spot (and put nulls where
			// needed (thus the -= 1 bits)
			for p.Time <= t-step && i < cur_len-1 {
				i++
				p = r.Data[i]
			}
			if p.Time <= t-step {
				i++
			}
			t -= step
			o -= 1
		}
	}

	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	r.Data = data
	return nil
}

// merges 2 series into the current one .. it will quantize them first
// if the 'r' data point is Null, it will use the 'm' data point
// otherwise it will just continue using "r" as we assume that's the source of truth
func (r *RawRenderItem) Merge(m *RawRenderItem) error {

	// steps sizes need to be the same
	if r.Step != m.Step {
		return errMergeStepSizeError
	}

	if m.Start < r.Start {
		r.Start = m.Start
	} else {
		m.Start = r.Start
	}
	if m.End > r.End {
		r.End = m.End
	} else {
		m.End = r.End
	}
	if m.RealStart < r.RealStart {
		r.RealStart = m.RealStart
	} else {
		m.RealStart = r.RealStart
	}
	if m.RealEnd > r.RealEnd {
		r.RealEnd = m.RealEnd
	} else {
		m.RealEnd = r.RealEnd
	}

	// both series should be the same size after this step
	err := r.Quantize()
	if err != nil {
		return err
	}
	err = m.Quantize()
	if err != nil {
		return err
	}

	// find the "longest" one
	cur_len := len(m.Data)
	for i := 0; i < cur_len; i++ {
		if r.Data[i].IsNull() {
			r.Data[i] = m.Data[i]
		}
	}

	r.Tags = r.Tags.Merge(m.Tags)
	r.MetaTags = r.MetaTags.Merge(m.MetaTags)
	return nil
}

// when we "merge" the two series we also "aggregate" the overlapping
// data points
func (r *RawRenderItem) MergeAndAggregate(m *RawRenderItem) error {
	// steps sizes need to be the same
	if r.Step != m.Step {
		return errMergeStepSizeError
	}

	if m.Start < r.Start {
		r.Start = m.Start
	} else {
		m.Start = r.Start
	}
	if m.End > r.End {
		r.End = m.End
	} else {
		m.End = r.End
	}
	if m.RealStart < r.RealStart {
		r.RealStart = m.RealStart
	} else {
		m.RealStart = r.RealStart
	}
	if m.RealEnd > r.RealEnd {
		r.RealEnd = m.RealEnd
	} else {
		m.RealEnd = r.RealEnd
	}

	// both series should be the same size after this step
	err := r.Quantize()
	if err != nil {
		return err
	}

	err = m.Quantize()
	if err != nil {
		return err
	}

	// find the "longest" one
	cur_len := len(m.Data)
	for i := 0; i < cur_len; i++ {
		if r.Data[i].IsNull() {
			r.Data[i] = m.Data[i]
		} else {
			r.Data[i].Merge(&m.Data[i])
		}
	}
	r.Tags = r.Tags.Merge(m.Tags)
	r.MetaTags = r.MetaTags.Merge(m.MetaTags)

	return nil
}

// this is similar to the MergeAndAgg but it will NOT create nulls
// for missing points in a time sequence, and Resample the data at the same time
func (r *RawRenderItem) MergeWithResample(d *RawRenderItem, step uint32) error {
	// based on the start, stop and step.  Fill in the gaps in missing
	// slots (w/ nulls) as graphite does not like "missing times" (expects things to have a constant
	// length over the entire interval)
	// You should Put in an "End" time of "ReadData + Step" to avoid loosing the last point as things
	// are  [Start, End) not [Start, End]

	if step <= 0 {
		return errQuantizeStepTooSmall
	}

	// make sure the start/ends are nicely divisible by the Step
	start := r.RealStart
	if d.RealStart < start {
		start = d.RealStart
	}

	left := start % step
	if left != 0 {
		start = start + step - left
	}

	end := r.RealEnd
	if d.RealEnd > r.RealEnd {
		end = d.RealEnd
	}

	endTime := (end - 1) - ((end - 1) % step) + step

	// make sure in time order
	sort.Sort(r.Data)
	sort.Sort(d.Data)

	// 'i' iterates Original Data.
	// 'o' iterates Incoming Data.
	// 'n' iterates New Data.
	// t is the current time we need to fill/merge
	var t uint32
	var i, o, n int

	i_len := len(r.Data)
	o_len := len(d.Data)
	data := make([]RawDataPoint, 0)

	for t, i, o, n = start, 0, 0, -1; t <= endTime; t += step {
		dp := NullRawDataPoint(t)
		// loop through the orig data until we hit a time > then the current one
		for ; i < i_len; i++ {

			if r.Data[i].Time > t {
				break
			}
			dp.Merge(&r.Data[i])
		}

		// loop through the incoming data until we hit a time > then the current one
		for ; o < o_len; o++ {

			if d.Data[o].Time > t {
				break
			}
			dp.Merge(&d.Data[o])
		}
		if !dp.IsNull() {
			data = append(data, dp)
			n++
		}
	}

	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	r.Data = data
	r.Tags = r.Tags.Merge(d.Tags)
	r.MetaTags = r.MetaTags.Merge(d.MetaTags)

	return nil

}
