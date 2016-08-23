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


	triggered "rollups"

	Since Blob series for "long flush windows" (like 60+seconds) can take a _very_ long time
	to flush if the blob size is in the kb range, basically they will take for ever to be persisted
	and we don't want to persist "just a few points" in the storage, so for the "smallest" flush window
	any time we do an overflow write we trigger a "rollup" event which will

	This also means we only need to store in the cache just the smallest time which will save lots of
	ram if you have lots of flush windows.

	The process is as follows

	1. the min Resolution writer triggers a "rollup" once it writes data
	2. Based on the other resolutions grab all the data from the min resolution
	   that fits in the resolution window (start - reesoltuion, end + resoltuion)
	3. Resample that list to the large resolution
	4. if there is data in the range already, just update the row, otherwise insert a new one

	find the current UinqueId already persisted for the longer flush windows
	If found: "MERGE" the new data into it (if the blob size is still under the window)
		"START" a new blob if too big
	If not found:
		"Start" a new blob

*/

package metrics

import (
	"cadent/server/utils/shutdown"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/dispatch"
	"cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils"
	"time"
)

const (
	ROLLUP_QUEUE_LENGTH = 2048
	ROLLUP_NUM_WORKERS  = 8
	ROLLUP_NUM_RETRIES  = 2
)

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// rollup job queue workers
type RollupJob struct {
	Rollup *RollupMetric
	Ts     TotalTimeSeries
	Retry  int
}

func (j *RollupJob) IncRetry() int {
	j.Retry++
	return j.Retry
}
func (j *RollupJob) OnRetry() int {
	return j.Retry
}

func (j *RollupJob) DoWork() error {
	err := j.Rollup.DoRollup(j.Ts)
	return err
}

/****************** Interfaces *********************/
type RollupMetric struct {
	writer        DBMetrics
	resolutions   [][]int
	minResolution int

	dispatcher *dispatch.DispatchQueue

	series_encoding string
	blobMaxBytes    int

	startstop utils.StartStop

	log *logging.Logger
}

func NewRollupMetric(writer DBMetrics, maxBytes int) *RollupMetric {
	rl := new(RollupMetric)
	rl.writer = writer
	rl.blobMaxBytes = maxBytes
	rl.log = logging.MustGetLogger("writers.rollup")
	return rl
}

// the resolution/ttl pairs we wish to rollup .. you should not include
// the resolution that is the foundation for the rollup
// if the resolutions are [ [ 5, 100], [ 60, 720 ]] just include the [[60, 720]]
func (rl *RollupMetric) SetResolutions(res [][]int) {
	rl.resolutions = res
}

func (rl *RollupMetric) SetMinResolution(res int) {
	rl.minResolution = res
}

func (rl *RollupMetric) Start() {
	rl.startstop.Start(func() {
		rl.log.Notice("Starting rollup engine writer at %d bytes per series", rl.blobMaxBytes)
		rl.dispatcher = dispatch.NewDispatchQueue(
			ROLLUP_NUM_WORKERS,
			ROLLUP_QUEUE_LENGTH,
			ROLLUP_NUM_RETRIES,
		)
		rl.dispatcher.Start()
	})
}

func (rl *RollupMetric) Stop() {
	rl.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		rl.log.Warning("Starting Shutdown of rollup engine")

		if rl.dispatcher != nil {
			rl.dispatcher.Stop()
		}
	})
}

func (rl *RollupMetric) Add(ts *TotalTimeSeries) {
	rl.dispatcher.Add(&RollupJob{Rollup: rl, Ts: *ts})
}

/*
 for each resolution we have find the latest item in the DB

 There are 4 main conditions we need to satisfy

  Old: S------------------E
  NEW:                      S--------------E
  1. NewData.Start > OldData.End --> merge and write (this should be 99% of cases, where OldE == NewS +/- resolution step)

  OLD: S------------------E
  NEW:           S-----------------E
  2. NewData.Start >= OldData.Start && NewData.End >= OldData.End -> Merge and write

  OLD:          S-----------------E
  NEW: S---------------E
  3. NewData.Start < OldData.Start && NewData.End < OldData.End && NewData.End > OldData.Start -> we have a "backwards" in time merge which

  OLD:                      S-----------E
  NEW: S---------------E
  4. NewData.End < OldData.Start -> Merge

  The flow:

  We know New.Start + New.End but not what the Old start and ends are.

  case #1 and #2 should be most of the cases, unless things are being back filled

  To start we need to get the Latest from the DB to check on conditions 1 -> 2

 	If there is no latest, then we know we are brand new and should just resample and write it out

  	Otherwise, merge the 2 streams, make a new series that are split into the proper chunk sizes.

  	If the resulting series is less then the chunk size, we simply "update" the DB row w/ the new data and end time

  	If not, we update the old row w/ the first chunk, and add new rows for each of the other chunks

  Fore cases 3 + 4 we need to ask for any data from the DB that spans the New Start and End

  OLD: S---------E S---------E ...... S---------E
  NEW     S-------------E

  The merge the series again into one

       S---------------------E

   And then update those N rows w/ the new starts/ends and data

*/

func (rl *RollupMetric) DoRollup(tseries TotalTimeSeries) error {
	defer stats.StatsdSlowNanoTimeFunc("writer.rollup.process-time-ns", time.Now())

	// make the series into our resampler object
	new_data, err := NewRawRenderItemFromSeries(&tseries)
	if err != nil {
		rl.log.Errorf("Rollup Failure: %v", err)
		return err
	}

	rl.log.Debug("Rollup Triggered for %s (%s)", tseries.Name.Key, tseries.Name.UniqueIdString())
	writeOne := func(rawd *RawRenderItem, old_data DBSeriesList, resolution int, ttl int) error {
		defer stats.StatsdSlowNanoTimeFunc("reader.rollup.write-time-ns", time.Now())
		// make the new series
		n_opts := series.NewDefaultOptions()
		n_opts.HighTimeResolution = tseries.Series.HighResolution()
		nseries, err := series.NewTimeSeries(tseries.Series.Name(), int64(rawd.RealStart), n_opts)
		if err != nil {
			return err
		}

		// we may need to add multiple chunks based on the chunk size we have
		m_tseries := make([]series.TimeSeries, 0)
		//rl.log.Debug("WRITE: %d %d %s Data: %d", rawd.RealStart, rawd.RealEnd, tseries.Name.Key, len(rawd.Data))
		for _, point := range rawd.Data {
			// skip it if not a real point
			if point.IsNull() {
				continue
			}
			// raw data pts time are in seconds
			use_t := time.Unix(int64(point.Time), 0).UnixNano()
			err := nseries.AddPoint(use_t, point.Min, point.Max, point.Last, point.Sum, point.Count)
			if err != nil {
				rl.log.Error("Add point failed: %v", err)
				continue
			}
			// need a new series
			if nseries.Len() > rl.blobMaxBytes {
				m_tseries = append(m_tseries, nseries)

				n_opts := series.NewDefaultOptions()
				n_opts.HighTimeResolution = tseries.Series.HighResolution()
				nseries, err = series.NewTimeSeries(tseries.Series.Name(), use_t, n_opts)
				if err != nil {
					return err
				}
			}
		}
		if nseries.Count() > 0 {
			m_tseries = append(m_tseries, nseries)
		}
		new_name := tseries.Name
		new_name.TTL = uint32(ttl)
		new_name.Resolution = uint32(resolution)

		// We walk through the old data points and see if we need to "update" the rows
		// or add new ones
		// we effectively update the series until we're out of the old ones, and then add new ones for the
		// remaining series
		l_old := len(old_data)
		for idx, ts := range m_tseries {
			if l_old > idx {
				rl.log.Debug("Update Series: %s %d->%d", old_data[idx].Uid, old_data[idx].Start, old_data[idx].End)
				rl.writer.UpdateDBSeries(old_data[idx], ts)
			} else {
				rl.log.Debug("Insert New Series: %s (%s) %d->%d", new_name.Key, new_name.UniqueIdString(), ts.StartTime(), ts.LastTime())
				rl.writer.InsertDBSeries(new_name, ts, uint32(resolution))
			}
		}

		return nil
	}

	// now march through the higher resolutions .. the resolution objects
	// is [ resolution, ttl ]
	nano_s := int64(time.Second)
	for _, res := range rl.resolutions {
		t_new_data := *new_data
		t_new_data.Resample(uint32(res[0]))
		// use nicely truncated blocks
		t_start := uint32(TruncateTimeTo(int64(new_data.Start), res[0]))
		t_end := uint32(TruncateTimeTo(int64(new_data.End)+int64(res[0]), res[0]))

		// if the start|End time is 0 then there is trouble w/ the series itself
		// and attempting a rollup is next to suicide
		if t_start == 0 || t_end == 0 {
			rl.log.Errorf("Rollup failure: Start|End time is 0, the series is corrupt cannot rollup")
			continue
		}

		// first see if the "latest" series does our case 1 or 2
		r_stats, err := rl.writer.GetLatestFromDB(tseries.Name, uint32(res[0]))

		if len(r_stats) > 0 && (r_stats.Start() == 0 || r_stats.End() == 0) {
			rl.log.Errorf("Rollup failure: Start|End time is 0 in the DB, the series is corrupt cannot rollup")
			continue
		}

		// we have a "new" series to write
		if len(r_stats) == 0 {
			t_new_data.Start = t_start
			t_new_data.RealStart = t_start
			t_new_data.RealEnd = t_end
			t_new_data.End = t_end

			err = writeOne(&t_new_data, nil, res[0], res[1])
			if err != nil {
				rl.log.Errorf("Rollup failure: %v", err)
			}
			continue
		}

		t_start_nano := nano_s * int64(t_new_data.Start)
		t_end_nano := nano_s * int64(t_new_data.End)

		if t_start_nano >= r_stats.End() || (t_start_nano >= r_stats.Start() && t_end_nano >= r_stats.End()) {
			old_data, err := r_stats.ToRawRenderItem()
			old_data.Id = t_new_data.Id
			old_data.Step = uint32(res[0])
			old_data.Metric = t_new_data.Metric

			if err != nil {
				rl.log.Errorf("Rollup failure: %v", err)
				continue
			}
			//fmt.Println("CASE1: Old Points:", old_data)
			//old_data.PrintPoints()
			//fmt.Println("Pre Merge Points")
			//t_new_data.PrintPoints()
			err = t_new_data.MergeAndAggregate(old_data)
			if err != nil {
				rl.log.Errorf("Rollup failure: %v", err)
				continue
			}
			//fmt.Println("New Points:", t_new_data.String())
			//t_new_data.PrintPoints()
			err = writeOne(&t_new_data, r_stats, res[0], res[1])
			if err != nil {
				rl.log.Errorf("Rollup failure: %v", err)
			}
			continue
		}

		// need to get more data in our range
		r_stats, err = rl.writer.GetRangeFromDB(tseries.Name, t_start, t_end, uint32(res[0]))
		if err != nil {
			rl.log.Errorf("Rollup failure: %v", err)
			continue
		}

		// we need to make "one big series" from the DB items
		old_data, err := r_stats.ToRawRenderItem()
		old_data.Id = t_new_data.Id
		old_data.Step = uint32(res[0])
		old_data.Metric = t_new_data.Metric
		if err != nil {
			rl.log.Errorf("Rollup failure: %v", err)
			continue
		}

		// the "new" data will win over any older ones
		//fmt.Println("Old Points")
		//old_data.PrintPoints()
		//fmt.Println("Pre Merge Points")
		//t_new_data.PrintPoints()
		t_new_data.MergeAndAggregate(old_data)
		t_new_data.Start = t_start
		t_new_data.End = t_end
		//fmt.Println("New Points")
		//t_new_data.PrintPoints()
		//now simply either replace the old data with new ones
		err = writeOne(&t_new_data, r_stats, res[0], res[1])
		if err != nil {
			rl.log.Errorf("Rollup failure: %v", err)
		}

	}
	return nil
}
