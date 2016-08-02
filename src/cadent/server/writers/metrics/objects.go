/*
   Base objects to simulate a graphite API response
*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/writers/series"
	"fmt"
	"math"
)

/******************  a simple union of series.TimeSeries and repr.StatName *********************/
type TotalTimeSeries struct {
	Name   *repr.StatName
	Series series.TimeSeries
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

/*** Raw renderer **/

type RawDataPoint struct {
	Time  uint32  `json:"time"`
	Sum   float64 `json:"sum"`
	Mean  float64 `json:"mean"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	First float64 `json:"first"`
	Last  float64 `json:"last"`
	Count int64   `json:"count"`
}

func NullRawDataPoint(time uint32) RawDataPoint {
	return RawDataPoint{
		Time:  time,
		Sum:   math.NaN(),
		Mean:  math.NaN(),
		Min:   math.NaN(),
		Max:   math.NaN(),
		First: math.NaN(),
		Last:  math.NaN(),
		Count: math.MinInt64,
	}
}

func (r *RawDataPoint) IsNull() bool {
	return r.Count == math.MinInt64 && math.IsNaN(r.Sum) && math.IsNaN(r.Mean) && math.IsNaN(r.First) && math.IsNaN(r.Last) && math.IsNaN(r.Min) && math.IsNaN(r.Max)
}

func (r *RawDataPoint) String() string {
	return fmt.Sprintf("RawDataPoint: T: %d Mean: %f", r.Time, r.Mean)
}

// merge two data points into one .. this is a nice merge that will add counts, etc to the pieces
// the "time" is then the greater of the two
func (r *RawDataPoint) Merge(d *RawDataPoint) {
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

	if d.Time != 0 && d.Time < r.Time && !math.IsNaN(d.First) {
		r.First = d.First
	}

	if d.Time != 0 && d.Time > r.Time && !math.IsNaN(d.Last) {
		r.Last = d.Last
	}

	if r.Count == math.MinInt64 && d.Count > math.MaxInt64 {
		r.Count = d.Count
	} else if r.Count != math.MinInt64 && d.Count != math.MinInt64 {
		r.Count += d.Count
	}

	if !math.IsNaN(r.Sum) && r.Count != math.MinInt64 && r.Count > 0 {
		r.Mean = r.Sum / float64(r.Count)
	} else if !math.IsNaN(r.Mean) && !math.IsNaN(d.Mean) {
		// if no sum/count .. need to do a crude time weighted mean of the two points
		// this is probably not anywhere near "statistically" accurate
		// but it's kinda like a linear interoplation
		r.Mean = (float64(r.Time)*r.Mean + float64(d.Time)*d.Mean) / float64(r.Time+d.Time)
	} else if math.IsNaN(r.Mean) && !math.IsNaN(d.Mean) {
		r.Mean = d.Mean // can only use this one
	}

	if r.Time < d.Time {
		r.Time = d.Time
	}

}

type RawRenderItem struct {
	Metric    string         `json:"metric"`
	RealStart uint32         `json:"data_from"`
	RealEnd   uint32         `json:"data_end"`
	Start     uint32         `json:"from"`
	End       uint32         `json:"to"`
	Step      uint32         `json:"step"`
	Data      []RawDataPoint `json:"data"`
}

func (r *RawRenderItem) Len() int {
	return len(r.Data)
}

func (r *RawRenderItem) String() string {
	return fmt.Sprintf("RawRenderItem: Start: %d End: %d, Points: %d", r.Start, r.End, r.Len())
}

// is True if the start and end times are contained in this data blob
func (r *RawRenderItem) DataInRange(start uint32, end uint32) bool {
	return start >= r.RealStart && end <= r.RealEnd
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
	return len(data)
}

// a crude "resampling" of incoming data .. if the incoming is non-uniform in time
// and we need to smear the resulting set into a uniform vector, since things like
// counts, sums, etc are easily just added together, we just do that, the "Mean" is
// more trouble (if no Sum and Counts are given) as we then need to do a time weighted
// mean (see Merge above)
//
// If moving from a "large" time step to a "smaller" one you WILL GET NULL values for
// time slots that do not match anything .. we cannot (will not) interpolate data like that
// as it's 99% statistically "wrong"
func (r *RawRenderItem) Resample(step uint32) {

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
		// there's no data in this Step so just bail (too small a step)
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
	r.Step = step
	r.End = endTime
	r.Data = data
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
		return fmt.Errorf("Cannot quantize: to a '0' step size ...")
	}

	// make sure the start/ends are nicely divisible by the Step
	start := r.Start
	left := r.Start % step
	if left != 0 {
		start = r.Start + step - left
	}

	endTime := (r.End - 1) - ((r.End - 1) % step)

	if endTime < start {
		// there's no data in this Step so just bail (too small a step)
		return fmt.Errorf("Cannot quantize to step: too little data")
	}

	// data length of the new data array
	data := make([]RawDataPoint, (endTime-start)/step+1)
	cur_len := uint32(len(r.Data))

	// 'i' iterates Original Data. 'o' iterates OutGoing Data.
	// t is the current time we need to fill/merge
	for t, i, o := start, uint32(0), -1; t <= endTime; t += step {
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
	r.Step = step
	r.End = endTime
	r.Data = data
	return nil
}

// merges 2 series into the current one .. it will quantize them first
// if the 'r' data point is Null, it will use the 'm' data point
// otherwise it will just continue using "r" as we assume that's the source of truth
func (r *RawRenderItem) Merge(m *RawRenderItem) error {

	// steps sizes need to be the same
	if r.Step != m.Step {
		return fmt.Errorf("To merge 2 RawRenderItems, the step size needs to be the same")
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
	r.Quantize()
	m.Quantize()

	// find the "longest" one
	cur_len := len(m.Data)
	for i := 0; i < cur_len; i++ {
		if r.Data[i].IsNull() {
			r.Data[i] = m.Data[i]
		}
	}
	return nil
}
