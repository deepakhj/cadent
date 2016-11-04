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
	The Cassandra Metric Log

	This is an attempt to mend the issue w/ the "series" metric writers.  That issue is what happens when things
	crash, restart, etc.  Any of the series in Ram get lost.  So this is a slightly different table
	where per "now" timestamp we take the recent metric arrivals, and add them in one big blob of zstd compressed
	json.

	On start up, the series writer then simply reads back in this log, and maintains its RAM profile
	for querying and eventual writing into the time-series tables

	This fundamentally changes "overflow" writing into a time chunk based method.

	the ram series cache is then flushed every "chunk" (configurable, default of 10m) and we keep "N" chunks (configurable,
	default of 6) for each metric in Ram for fast queries.  Once we "hit #7" we flush out the oldest one to cassandra
	Once that entire chunk is flushed out, we drop that "sequence" we are on (a sequence is just the counter
	of each chunk we've completed)

	Thus if we crash, we still have the most (yes some data will be lost depending on the crash
	scenario, but we will have much more then if we had just the RAM series).

	Each writer needs to have its own log table, (if there are 3 cadents writing/reading from the same DB
	each one will be at a different checkpoint) so there MUST be an "index" for them otherwise on crash
	and restart (or even node loss and restore) it needs to know where to pick up from.


	CREATE TABLE metric_log_{cadentindex} (
    		stamp bigint,
    		sequence bigint,
    		points blob,
    		PRIMARY KEY (stamp, sequence)
	) WITH COMPACT STORAGE AND CLUSTER ORDERING stamp DESC;



*/

package metrics

import (
	"bytes"
	"cadent/server/broadcast"
	"cadent/server/dispatch"
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/indexer"
	"encoding/json"
	"fmt"
	"github.com/DataDog/zstd"
	"sort"
	"strings"
	"time"
)

const (
	// how many chunks to keep in RAM
	CASSANDRA_LOG_CHUNKS = 6

	// hold CASSANDRA_LOG_TIME_CHUNKS duration per timeseries
	CASSANDRA_LOG_TIME_CHUNKS = "10m"

	// every CASSANDRA_LOG_FLUSH duration, drop the current set of points into the log table
	CASSANDRA_LOG_FLUSH = "10s"

	// max bytes for the rollup triggers
	CASSANDRA_LOG_ROLLLUP_MAXBYTES = 2048

	CASSANDRA_LOG_TTL = int64(24 * time.Hour) // expire stuff after a day

)

/****************** Metrics Writer *********************/
type CassandraLogMetric struct {
	CassandraMetric

	cacher     *CacherChunk
	loggerChan *broadcast.Listener // log writes
	slicerChan *broadcast.Listener // current slice writes

	okToWrite   chan bool // temp channel to block as we may need to pull in the cache first
	writerIndex int
}

func NewCassandraLogMetrics() *CassandraLogMetric {
	cass := new(CassandraLogMetric)
	cass.driver = "cassandra-log"
	cass.isPrimary = false
	cass.rollupType = "cached"
	cass.okToWrite = make(chan bool)
	return cass
}

func NewCassandraLogTriggerMetrics() *CassandraLogMetric {
	cass := new(CassandraLogMetric)
	cass.driver = "cassandra-log-triggered"
	cass.isPrimary = false
	cass.rollupType = "triggered"
	return cass
}

// Config setups all the options for the writer
func (cass *CassandraLogMetric) Config(conf *options.Options) (err error) {

	/*err = cass.CassandraMetric.Config(conf)
	if err != nil {
		return err
	}*/
	_, err = conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	// only need one real "writer DB" here as we are writing to the same metrics table
	gots, err := _get_cass_singleton(conf)
	if err != nil {
		return err
	}

	cass.writer = gots

	// even if not used, it best be here for future goodies
	_, err = conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("resolution needed for cassandra writer: %v", err)
	}

	// need writer index for the log table
	i, err := conf.Int64Required("writer_index")
	if err != nil {
		return fmt.Errorf("writer_index is required for cassandra-log writer: %v", err)
	}
	cass.writerIndex = int(i)

	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		cass.staticTags = repr.SortingTagsFromString(_tgs)
	}

	// tweak queus and worker sizes
	cass.numWorkers = int(conf.Int64("write_workers", CASSANDRA_DEFAULT_METRIC_WORKERS))
	cass.queueLen = int(conf.Int64("write_queue_length", CASSANDRA_DEFAULT_METRIC_QUEUE_LEN))
	cass.dispatchRetries = int(conf.Int64("write_queue_retries", CASSANDRA_DEFAULT_METRIC_RETRIES))
	cass.tablePerResolution = conf.Bool("table_per_resolution", false)

	rdur, err := time.ParseDuration(CASSANDRA_DEFAULT_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	cass.renderTimeout = rdur

	// rolluptype
	if cass.rollupType == "" {
		cass.rollupType = conf.String("rollup_type", CASSANDRA_DEFAULT_ROLLUP_TYPE)
	}

	_cache, err := conf.ObjectRequired("cache")
	if _cache == nil {
		return errMetricsCacheRequired
	}
	// need to cast this to the proper cache type
	var ok bool
	cass.cacher, ok = _cache.(*CacherChunk)
	if !ok {
		return ErrorMustBeChunkedCacheType
	}

	// deal w/ options later XXX TODO

	// for the overflow cached items::
	// these caches can be shared for a given writer set, and the caches may provide the data for
	// multiple writers, we need to specify that ONE of the writers is the "main" one otherwise
	// the Metrics Write function will add the points over again, which is not a good thing
	// when the accumulator flushes things to the multi writers
	// The Writer needs to know it's "not" the primary writer and thus will not "add" points to the
	// cache .. so the cache basically gets "one" primary writer pointed (first come first serve)
	cass.isPrimary = cass.cacher.SetPrimaryWriter(cass)
	if cass.isPrimary {
		cass.writer.log.Notice("Cassandra series writer is the primary writer to write back cache %s", cass.cacher.Name)
	}

	if cass.rollupType == "triggered" {
		cass.driver = "cassandra-log-triggered" // reset the name
		cass.rollup = NewRollupMetric(cass, CASSANDRA_LOG_ROLLLUP_MAXBYTES)
	}

	if cass.tablePerResolution {
		cass.selQ = "SELECT ptype, points FROM %s WHERE id=? AND etime >= ? AND etime <= ?"
	} else {
		cass.selQ = "SELECT ptype, points FROM %s WHERE mid={id: ?, res: ?} AND etime >= ? AND etime <= ?"
	}

	return nil
}

func (cass *CassandraLogMetric) Driver() string {
	return cass.driver
}

func (cass *CassandraLogMetric) Start() {
	cass.startstop.Start(func() {

		cass.okToWrite = make(chan bool)

		/**** dispatcher queue ***/
		cass.writer.log.Notice("Starting cassandra-log series writer for %s at %d time chunks", cass.writer.db.MetricTable(), cass.cacher.maxTime)

		cass.writer.log.Notice("Adding metric tables ...")

		schems := NewCassandraMetricsSchema(
			cass.writer.conn,
			cass.writer.db.Keyspace(),
			cass.writer.db.MetricTable(),
			cass.resolutions,
			"blob",
			cass.tablePerResolution,
			cass.writer.db.GetCassandraVersion(),
		)

		// table names is {base}_{windex}_{res}s
		schems.LogTable = cass.writer.db.LogTableBase()
		schems.WriteIndex = cass.writerIndex
		schems.Resolution = cass.currentResolution
		err := schems.AddMetricsTable()

		if err != nil {
			panic(err)
		}
		cass.writer.log.Notice("Adding metric log tables")
		err = schems.AddMetricsLogTable()
		if err != nil {
			panic(err)
		}

		cass.shutitdown = false

		cass.loggerChan = cass.cacher.GetLogChan()
		cass.slicerChan = cass.cacher.GetSliceChan()

		// if the resolutions list is just "one" there is no triggered rollups
		if len(cass.resolutions) == 1 {
			cass.rollupType = "cached"
		}

		cass.writer.log.Notice("Rollup Type: %s on resolution: %d (min resolution: %d)", cass.rollupType, cass.currentResolution, cass.resolutions[0][0])
		cass.doRollup = cass.rollupType == "triggered" && cass.currentResolution == cass.resolutions[0][0]
		// start the rollupper if needed
		if cass.doRollup {
			cass.writer.log.Notice("Starting rollup machine")
			// all but the lowest one
			cass.rollup.blobMaxBytes = CASSANDRA_LOG_ROLLLUP_MAXBYTES
			cass.rollup.SetResolutions(cass.resolutions[1:])
			go cass.rollup.Start()
		}

		//start up the dispatcher
		cass.dispatcher = dispatch.NewDispatchQueue(
			cass.numWorkers,
			cass.queueLen,
			cass.dispatchRetries,
		)
		cass.dispatcher.Start()

		go cass.writeLog()
		go cass.writeSlice()

		//slurp the old logs if any sequence reads will force a write for each distinct
		// sequence ID so the rest of the manifold needs to be set up
		cass.getSequences()

		// now we can write
		close(cass.okToWrite)
		cass.okToWrite = nil

		cass.cacher.Start()

	})
}

func (cass *CassandraLogMetric) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		cass.writer.log.Warning("Starting Shutdown of cassandra series writer")

		if cass.shutitdown {
			return // already did
		}
		cass.shutitdown = true

		cass.cacher.Stop()

		// we only need to write any metrics in the
		cItems := cass.cacher.CurrentLog()
		cass.writer.log.Warning("Shutting down %s and writing final log (%d metrics)", cass.cacher.Name, len(cItems))

		if cass.doRollup {
			cass.rollup.Stop()
		}
		if cass.dispatcher != nil {
			cass.dispatcher.Stop()
		}

		cass.writer.log.Warning("Shutdown finished ... quiting cassandra series writer")
		return
	})
}

// for sorting below
type intList []int64

func (p intList) Len() int { return len(p) }
func (p intList) Less(i, j int) bool {
	return p[i] < p[j]
}
func (p intList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// on startup yank the current sequences we may have missed due to crash/restart reassingment
func (cass *CassandraLogMetric) getSequences() {

	cass.writer.log.Notice("Backfilling internal caches from log")

	// due to sorting restrictions in cassandra, we need to get the sequence numbers first then sort them
	// to get things in time order
	Q := fmt.Sprintf(
		"SELECT DISTINCT seq FROM %s_%d_%ds",
		cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution,
	)
	seqiter := cass.writer.conn.Query(Q).Iter()
	defer seqiter.Close()

	sequences := make(intList, 0)
	var tSeq int64
	for seqiter.Scan(&tSeq) {
		sequences = append(sequences, tSeq)
	}
	sort.Sort(sequences)
	curSeq := int64(0)
	added := 0

	if len(sequences) > 0 {
		// set the current sequence number 'higher"
		curSeq = sequences[len(sequences)-1]
		cass.cacher.curSequenceId = curSeq
	}

	// now purge out the sequence number as we've writen our data
	Q = fmt.Sprintf(
		"SELECT seq, pts FROM %s_%d_%ds WHERE seq = ? ORDER BY ts ASC",
		cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution,
	)

	for _, s := range sequences {

		iter := cass.writer.conn.Query(Q, s).Iter()
		defer iter.Close()

		var sequence int64
		var pts []byte
		didChunks := 0
		for iter.Scan(&sequence, &pts) {
			didChunks++
			cass.writer.log.Notice("Parsing chunk %d into cache for sequence # %d", didChunks, s)

			// unzstd the punk
			decomp := zstd.NewReader(bytes.NewBuffer(pts))

			var data []byte
			_, err := decomp.Read(data)
			if err != nil {
				cass.writer.log.Errorf("Error reading compressed slice log data: %v", err)
				decomp.Close()
				continue
			}

			slice := make(map[repr.StatId][]*repr.StatRepr, 0)
			dec := json.NewDecoder(decomp)
			err = dec.Decode(&slice)
			if err != nil {
				cass.writer.log.Errorf("Error reading slice log data: %v", err)
				decomp.Close()
				continue
			}

			// now we need to add the points to the chunks
			// this can mean our first chunk is longer then the "slice time" but that should be ok
			for _, item := range slice {
				for _, stat := range item {
					cass.cacher.BackFill(stat.Name, stat, curSeq)
					added++
				}
			}
			decomp.Close()
		}

	}

	// set our sequence version
	cass.writer.log.Notice("Backfilled %d data points from the log.  Current Sequence is %d", added, curSeq)
}

// listen for the broad cast chan an flush out a log entry
func (cass *CassandraLogMetric) writeLog() {
	for {
		clog, more := <-cass.loggerChan.Ch
		if !more {
			return
		}
		if clog == nil {
			continue
		}
		tn := time.Now()

		toLog, ok := clog.(*CacheChunkLog)
		if !ok {
			stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.errors", 1)

			cass.writer.log.Critical("Not a CacheChunkLog type cannot write log")
			continue
		}

		// needs to be in seconds
		ttl := CASSANDRA_LOG_TTL / int64(time.Second)
		Q := fmt.Sprintf(
			"INSERT INTO %s_%d_%ds (seq, ts, pts) VALUES (?,?,?) USING TTL %d",
			cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution, ttl,
		)

		// zstd the punk
		buf := bytes.NewBuffer(nil)
		compressed := zstd.NewWriter(buf)
		enc := json.NewEncoder(compressed)
		err := enc.Encode(toLog.Slice)
		compressed.Close()

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.errors", 1)
			cass.writer.log.Critical("Json encode error: type cannot write log: %v", err)
			continue
		}

		err = cass.writer.conn.Query(
			Q,
			toLog.SequenceId,
			time.Now().UnixNano(),
			buf.Bytes(),
		).Exec()
		if err != nil {
			stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.errors", 1)
			cass.writer.log.Critical("Cassandra writer error: cannot write log: %v", err)
			continue
		}
		stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.success", 1)
		stats.StatsdSlowNanoTimeFunc("writer.cassandralog.log.write-time-ns", tn)

	}
}

// listen for the broadcast chan an flush out the newest slice in the mix
func (cass *CassandraLogMetric) writeSlice() {
	for {
		var err error

		clog, more := <-cass.slicerChan.Ch
		if !more {
			return
		}

		if clog == nil {
			continue
		}

		tn := time.Now()

		toLog, ok := clog.(*CacheChunkSlice)
		if !ok {
			stats.StatsdClientSlow.Incr("writer.cassandralog.slice.writes.errors", 1)
			cass.writer.log.Critical("Not a CacheChunkSlice type cannot write log")
			continue
		}

		hadErrors := int64(0)
		hadHappy := int64(0)
		for _, item := range toLog.Slice.AllSeries() {
			err = cass.doInsert(&TotalTimeSeries{Series: item.Series, Name: item.Name})
			if err != nil {
				cass.writer.log.Error("Series Insert Error: %v", err)
				hadErrors++
			} else {
				hadHappy++
			}
		}

		// now purge out the sequence number as we've writen our data
		// note that we don't really care about the "tombstone" issue here as we will never read from
		// this table unless we crash or restart, and the tombstone performance hit is not the
		// end of the world, compaction will eventually take care of these
		Q := fmt.Sprintf(
			"DELETE FROM %s_%d_%ds WHERE seq = ?",
			cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution,
		)
		err = cass.writer.conn.Query(
			Q,
			toLog.SequenceId,
		).Exec()

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.cassandralog.log.purge.errors", 1)
			cass.writer.log.Critical("Cassandra writer error: cannot purge sequence %d: %v", toLog.SequenceId, err)
		} else {
			cass.writer.log.Info(
				"Purged log sequence %d for writer %d and resolution %d",
				toLog.SequenceId,
				cass.writerIndex,
				cass.currentResolution,
			)
		}
		stats.StatsdClientSlow.Incr("writer.cassandralog.slice.writes.errors", hadErrors)
		stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.success", hadHappy)
		stats.StatsdNanoTimeFunc("writer.cassandralog.slice.write-time-ns", tn)
	}
}

// simple proxy to the cacher
func (cass *CassandraLogMetric) Write(stat repr.StatRepr) error {

	// block until ready
	if cass.okToWrite != nil {
		<-cass.okToWrite
	}

	if cass.shutitdown {
		return nil
	}

	stat.Name.MergeMetric2Tags(cass.staticTags)
	// only need to do this if the first resolution
	if cass.currentResolution == cass.resolutions[0][0] {
		cass.indexer.Write(*stat.Name)
	}

	// not primary writer .. move along
	if !cass.isPrimary {
		return nil
	}

	if cass.rollupType == "triggered" {
		if cass.currentResolution == cass.resolutions[0][0] {
			return cass.cacher.Add(stat.Name, &stat)
		}
	} else {
		return cass.cacher.Add(stat.Name, &stat)
	}
	return nil
}

/************************ READERS ****************/

// after the "raw" render we need to yank just the "point" we need from the data which
// will make the read-cache much smaller (will compress just the Mean value as the count is 1)
func (cass *CassandraLogMetric) RawDataRenderOne(metric *indexer.MetricFindItem, start int64, end int64, resample uint32) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandralog.rawrenderone.get-time-ns", time.Now())
	rawd := new(RawRenderItem)

	//figure out the best res
	resolution := cass.GetResolution(start, end)
	outResolution := resolution

	//obey the bigger
	if resample > resolution {
		outResolution = resample
	}

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	uStart := uint32(start)
	uEnd := uint32(end)

	statName := metric.StatName()

	rawd.Step = outResolution
	rawd.Metric = metric.Path
	rawd.Id = metric.UniqueId
	rawd.RealEnd = uEnd
	rawd.RealStart = uStart
	rawd.Start = rawd.RealStart
	rawd.End = rawd.RealEnd
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.AggFunc = statName.AggType()

	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return rawd, ErrorNotADataNode
	}

	b_len := (uEnd - uStart) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, ErrorTimeTooSmall
	}

	inflight, err := cass.GetFromWriteCache(metric, uStart, uEnd, resolution)

	// need at LEAST 2 points to get the proper step size
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// all the data we need is in the inflight
		// if all the data is in this list we don't need to go any further
		if inflight.RealStart <= uStart {
			// move the times to the "requested" ones and quantize the list
			if inflight.Step != outResolution {
				inflight.Resample(outResolution)
			}
			return inflight, err
		}
	}
	if err != nil {
		cass.writer.log.Error("Cassandra: Erroring getting inflight data: %v", err)
	}

	// and now for the mysql Query otherwise
	cassData, err := cass.GetFromDatabase(metric, resolution, start, end, resample)
	if err != nil {
		cass.writer.log.Error("Cassandra: Error getting from DB: %v", err)
		return rawd, err
	}

	cassData.Step = outResolution
	cassData.Start = uStart
	cassData.End = uEnd
	cassData.Tags = metric.Tags
	cassData.MetaTags = metric.MetaTags

	if inflight == nil || len(inflight.Data) == 0 {
		return cassData, nil
	}

	if len(cassData.Data) > 0 && len(inflight.Data) > 0 {
		inflight.MergeWithResample(cassData, outResolution)
		return inflight, nil
	}
	if inflight.Step != outResolution {
		inflight.Resample(outResolution)
	}
	return inflight, nil
}

func (cass *CassandraLogMetric) RawRender(path string, start int64, end int64, tags repr.SortingTags, resample uint32) ([]*RawRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandralog.rawrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	renderWg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(renderWg)

	for _, pth := range paths {
		mets, err := cass.indexer.Find(pth, tags)

		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*RawRenderItem, 0)

	procs := CASSANDRA_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan indexer.MetricFindItem, len(metrics))
	results := make(chan *RawRenderItem, len(metrics))

	renderOne := func(met indexer.MetricFindItem) *RawRenderItem {
		_ri, err := cass.RawDataRenderOne(&met, start, end, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.cassandralog.rawrender.errors", 1)
			cass.writer.log.Errorf("Read Error for %s (%d->%d) : %v", path, start, end, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan indexer.MetricFindItem, resultqueue chan<- *RawRenderItem) {
		rec_chan := make(chan *RawRenderItem, 1)
		for met := range taskqueue {
			go func() { rec_chan <- renderOne(met) }()
			select {
			case <-time.After(cass.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.cassandralog.rawrender.timeouts", 1)
				cass.writer.log.Errorf("Render Timeout for %s (%d->%d)", path, start, end)
				resultqueue <- nil
			case res := <-rec_chan:
				resultqueue <- res
			}
		}
	}

	for i := 0; i < procs; i++ {
		go jobWorker(i, jobs, results)
	}

	for _, metric := range metrics {
		jobs <- metric
	}
	close(jobs)

	for i := 0; i < len(metrics); i++ {
		res := <-results
		if res != nil {
			rawd = append(rawd, res)
		}
	}
	close(results)
	stats.StatsdClientSlow.Incr("reader.cassandralog.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}

func (cass *CassandraLogMetric) GetFromWriteCache(metric *indexer.MetricFindItem, start uint32, end uint32, resolution uint32) (*RawRenderItem, error) {

	// grab data from the write inflight cache
	// need to pick the "proper" cache
	cache_db := fmt.Sprintf("%s:%d", cass.cacherPrefix, resolution)
	use_res := resolution
	if cass.rollupType == "triggered" {
		cache_db = fmt.Sprintf("%s:%d", cass.cacherPrefix, cass.resolutions[0][0])
		use_res = uint32(cass.resolutions[0][0])
	}

	use_cache := GetCacherByName(cache_db)
	if use_cache == nil {
		use_cache = cass.cacher
	}
	statName := metric.StatName()
	inflight, err := use_cache.GetAsRawRenderItem(statName)

	if err != nil {
		return nil, err
	}
	if inflight == nil {
		return nil, nil
	}
	inflight.Metric = metric.Path
	inflight.Id = metric.UniqueId
	inflight.Step = use_res
	inflight.Start = start
	inflight.End = end
	inflight.Tags = metric.Tags
	inflight.MetaTags = metric.MetaTags
	inflight.AggFunc = statName.AggType()
	return inflight, nil
}

func (cass *CassandraLogMetric) CachedSeries(path string, start int64, end int64, tags repr.SortingTags) (series *TotalTimeSeries, err error) {

	return nil, errNotImplimented
	/*
		defer stats.StatsdSlowNanoTimeFunc("reader.cassandralog.seriesrender.get-time-ns", time.Now())

		paths := strings.Split(path, ",")
		if len(paths) > 1 {
			return series, errMultiTargetsNotAllowed
		}

		metric := &repr.StatName{Key: path}
		metric.MergeMetric2Tags(tags)
		metric.MergeMetric2Tags(cass.staticTags)

		resolution := cass.GetResolution(start, end)
		cache_db := fmt.Sprintf("%s:%v", cass.cacherPrefix, resolution)
		use_cache := GetChunkCacherByName(cache_db)
		if use_cache == nil {
			use_cache = cass.cacher
		}
		name, inflight, err := use_cache.GetSeries(metric)
		if err != nil {
			return nil, err
		}
		if inflight == nil {
			// try the the path as unique ID
			gotsInt := metric.StringToUniqueId(path)
			if gotsInt != 0 {
				name, inflight, err = use_cache.GetSeriesById(gotsInt)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, nil
			}
		}

		return &TotalTimeSeries{Name: name, Series: inflight}, nil
	*/
}
