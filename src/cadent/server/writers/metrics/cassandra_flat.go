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
	The Cassandra - Flat Metric Reader/Writer

	The table should have this schema to match the repr item
	The same as the writer items

	By "flat" we mean we don't store "chunks" of data in a blob, but a row, by row per time

	this one is much nicer for other system that need to access the data, but is more "space heavy"
	and not nearly as efficient as the data blob representation

	this one ALSO LETS YOU UPDATE/MERGE data points in case times come in the past for older data points (which can
	be a performance penalty)

	the blob version does NOT allow upserts

	A Schema for one to use

	CREATE TYPE metric_point (
            max double,
            min double,
            sum double,
            last double,
            count int
        );


        CREATE TYPE metric_path (
            path text,
            resolution int
        );

        CREATE TABLE metric.metric (
            id varchar,
            mpath frozen<metric_path>,
            time bigint,
            point frozen<metric_point>,
            PRIMARY KEY (id, mpath, time)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (mpath ASC, time ASC)
            AND compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '0.083',
                'base_time_seconds': '50'
            }
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
            AND dclocal_read_repair_chance = 0.1
            AND default_time_to_live = 0
            AND gc_grace_seconds = 864000
            AND max_index_interval = 2048
            AND memtable_flush_period_in_ms = 0
            AND min_index_interval = 128
            AND read_repair_chance = 0.0
            AND speculative_retry = '99.0PERCENTILE';

	CONFIG options::

	[graphite-cassandra-flat.accumulator.writer.metrics]
	driver="cassandra-flat"
	dsn="my.cassandra.com"
	[graphite-cassandra-flat.accumulator.writer.metrics.options]
		keyspace="metric"
		metric_table="metric"
		path_table="path"
		segment_table="segment"
		write_consistency="one"
		read_consistency="one"
		port=9042
		cache_metric_size=102400  # the "internal carbon-like-cache" size (ram is your friend)
		cache_byte_size=1024 # number of bytes
		cache_low_fruit_rate=0.25 # every 1/4 of the time write "low count" metrics to at least persist them
		writes_per_second=5000 # allowed insert queries per second

		numcons=5  # cassandra connection pool size
		timeout="30s" # query timeout
		user: ""
		pass: ""
		write_workers=32  # dispatch workers to write
		write_queue_length=102400  # buffered queue size before we start blocking

		# NOPE: batch_count: batch this many inserts for much faster insert performance (default 1000)
		# NOPE: periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/dbs"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"
	"os"
	"os/signal"
	"syscall"

	"cadent/server/broadcast"
	"cadent/server/dispatch"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/indexer"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	CASSANDRA_FLAT_METRIC_WORKERS    = 32
	CASSANDRA_FLAT_METRIC_QUEUE_LEN  = 1024 * 100
	CASSANDRA_FLAT_WRITES_PER_SECOND = 5000
	CASSANDRA_FLAT_WRITE_UPSERT      = true
	CASSANDRA_FLAT_RENDER_TIMEOUT    = "5s" // 5 second time out on any render

)

var errNotImplimented = errors.New("Method not implimented")

/** Being Cassandra we need some mappings to match the schemas **/

/**
	CREATE TYPE metric_point (
        max double,
        min double,
        sum double,
        last double,
        count int
    );
*/

type CassMetricPoint struct {
	Max   float64
	Min   float64
	Sum   float64
	Last  float64
	Count int
}

/*
	CREATE TYPE metric_id (
        path text,
        resolution int
    );
*/

type CassMetricID struct {
	Path       string
	Resolution int
}

/*
 CREATE TABLE metric (
        id frozen<metric_id>,
        time bigint,
        point frozen<metric_point>
 )
*/
type CassMetric struct {
	Id         CassMetricID
	Time       int64
	Resolution CassMetricPoint
}

/*** set up "one" real writer (per dsn) .. and writer queue .. no
  no need to get multiqueues/channel/etc of these per resolution
  as we are all sharing the same DB pool and should write things in as they come in
  multiple writer pools tend to lead to bad lock contention behavior on input channels
  and output channels as well as the cassandra writer (gocql) itself.
  Having a "single" real writer for all resolutions saves all of that.

  this, and the "non-channel" Queue in "writer.go", comes from ALOT of performance testing and tuning
  while not the "go'est" way of doing things.  It works with many 100s of thousands of metrics being flushed
  one a single machine.

  We don't need to do this for the "indexer" portion of the cassandra writer, as there is only "one" instance
  of that per DSN and it also maintains it's own "hot" cache check, which after 1-3 flushes will fill up and
  basically never write anymore

*/

// the singleton
var _CASS_FLAT_WRITER_SINGLETON map[string]*CassandraFlatWriter
var _cass_flat_set_mutex sync.Mutex

func _get_flat_signelton(conf map[string]interface{}) (*CassandraFlatWriter, error) {
	_cass_flat_set_mutex.Lock()
	defer _cass_flat_set_mutex.Unlock()
	gots := conf["dsn"]
	if gots == nil {
		return nil, fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	dsn := gots.(string)
	if val, ok := _CASS_FLAT_WRITER_SINGLETON[dsn]; ok {
		return val, nil
	}

	writer, err := NewCassandraFlatWriter(conf)
	if err != nil {
		return nil, err
	}
	_CASS_FLAT_WRITER_SINGLETON[dsn] = writer
	return writer, nil
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type CassandraFlatMetricJob struct {
	Cass  *CassandraFlatWriter
	Name  *repr.StatName
	Stats repr.StatReprSlice // where the point list live
	Retry int
}

func (j *CassandraFlatMetricJob) IncRetry() int {
	j.Retry++
	return j.Retry
}
func (j *CassandraFlatMetricJob) OnRetry() int {
	return j.Retry
}

func (j *CassandraFlatMetricJob) DoWork() error {
	_, err := j.Cass.InsertMulti(j.Name, j.Stats)
	return err
}

type CassandraFlatWriter struct {
	// juse the writer connections for this
	db   *dbs.CassandraDB
	conn *gocql.Session

	dispatcher *dispatch.DispatchQueue

	cacher                *Cacher
	cacheOverFlowListener *broadcast.Listener // on byte overflow of cacher force a write

	shutitdown bool // just a flag
	startstop  utils.StartStop

	writes_per_second int // allowed writes per second
	num_workers       int
	queue_len         int
	max_write_size    int           // size of that buffer before a flush
	max_idle          time.Duration // either max_write_size will trigger a write or this time passing will
	write_lock        sync.Mutex
	log               *logging.Logger

	// upsert (true) or select -> merge -> update (false)
	// either squish metrics that have the same time windowe as a previious insert
	// or try to "update" the data point if exists
	// note upsert is WAY faster and should handle most of the cases
	insert_mode bool

	_insert_query      string //render once
	_select_time_query string //render once
	_get_query         string //render once
}

func NewCassandraFlatWriter(conf map[string]interface{}) (*CassandraFlatWriter, error) {
	cass := new(CassandraFlatWriter)
	cass.log = logging.MustGetLogger("metrics.cassandraflat")
	cass.shutitdown = false

	gots := conf["dsn"]
	if gots == nil {
		return nil, fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	conn_key := fmt.Sprintf("%v:%v/%v/%v", gots, conf["port"], conf["keyspace"], conf["metrics_table"])
	cass.log.Notice("Connecting Metrics to Cassandra (%s)", conn_key)

	db, err := dbs.NewDB("cassandra", conn_key, conf)
	if err != nil {
		return nil, err
	}
	// need to cast for real usage
	cass.db = db.(*dbs.CassandraDB)
	cass.conn = db.Connection().(*gocql.Session)

	_wr_buffer := conf["batch_count"]
	if _wr_buffer == nil {
		cass.max_write_size = 50
	} else {
		// toml things generic ints are int64
		cass.max_write_size = int(_wr_buffer.(int64))
	}
	if gocql.BatchSizeMaximum < cass.max_write_size {
		cass.log.Warning("Cassandra Driver: Setting batch size to %d, as it's the largest allowed", gocql.BatchSizeMaximum)
		cass.max_write_size = gocql.BatchSizeMaximum
	}

	_pr_flush := conf["periodic_flush"]
	cass.max_idle = time.Duration(time.Second)
	if _pr_flush != nil {
		dur, err := time.ParseDuration(_pr_flush.(string))
		if err == nil {
			cass.max_idle = dur
		} else {
			cass.log.Error("Cassandra Driver: Invalid Duration `%v`", _pr_flush)
		}
	}

	// tweak queus and worker sizes
	_workers := conf["write_workers"]
	cass.num_workers = CASSANDRA_FLAT_METRIC_WORKERS
	if _workers != nil {
		cass.num_workers = int(_workers.(int64))
	}

	_qs := conf["write_queue_length"]
	cass.queue_len = CASSANDRA_FLAT_METRIC_QUEUE_LEN
	if _qs != nil {
		cass.queue_len = int(_qs.(int64))
	}

	_rs := conf["writes_per_second"]
	cass.writes_per_second = CASSANDRA_FLAT_WRITES_PER_SECOND
	if _rs != nil {
		cass.writes_per_second = int(_rs.(int64))
	}

	cass.insert_mode = CASSANDRA_FLAT_WRITE_UPSERT
	_up := conf["write_upsert"]
	if _up != nil {
		cass.insert_mode = _up.(bool)
	}

	cass._insert_query = fmt.Sprintf(
		"INSERT INTO %s (id, mpath, time, point) VALUES  (?, {path: ?, resolution: ?}, ?, {sum: ?, min: ?, max: ?, last: ?, count: ?})",
		cass.db.MetricTable(),
	)

	cass._select_time_query = fmt.Sprintf(
		"SELECT point.max, point.min, point.sum, point.last, point.count, time FROM %s WHERE id=? and mpath={path: ?, resolution: ?} AND time <= ? and time >= ?",
		cass.db.MetricTable(),
	)

	cass._get_query = fmt.Sprintf(
		"SELECT point.max, point.min, point.sum, point.last, point.count, time FROM %s WHERE id=? and mpath={path: ?, resolution: ?} and time = ?",
		cass.db.MetricTable(),
	)

	return cass, nil
}

func (cass *CassandraFlatWriter) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		cass.log.Warning("Starting Shutdown of writer")
		if cass.shutitdown {
			return // already did
		}
		cass.shutitdown = true

		cass.cacher.Stop()

		if cass.dispatcher != nil {
			cass.dispatcher.Stop()
		}

		mets := cass.cacher.Queue
		mets_l := len(mets)
		cass.log.Warning("Shutting down, exhausting the queue (%d items) and quiting", mets_l)
		// full tilt write out
		did := 0
		for _, queueitem := range mets {
			if did%100 == 0 {
				cass.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
			}
			name, points, _ := cass.cacher.GetById(queueitem.metric)
			if points != nil {
				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandraflat.write.send-to-writers"), 1)
				cass.InsertMulti(name, points)
			}
			did++
		}
		cass.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
		cass.log.Warning("Shutdown finished ... quiting cassandra writer")
	})
}

func (cass *CassandraFlatWriter) Start() {
	/**** dispatcher queue ***/
	cass.startstop.Start(func() {
		workers := cass.num_workers
		retries := 2
		cass.dispatcher = dispatch.NewDispatchQueue(workers, cass.queue_len, retries)
		cass.dispatcher.Start()

		cass.cacher.Start()
		go cass.sendToWriters() // the dispatcher
	})
}

func (cass *CassandraFlatWriter) TrapExit() {
	//trap kills to flush queues and close connections
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func(ins *CassandraFlatWriter) {
		s := <-sc
		cass.log.Warning("Caught %s: Flushing remaining points out before quit ", s)

		cass.Stop()
		signal.Stop(sc)
		close(sc)

		// re-raise it
		process, _ := os.FindProcess(os.Getpid())
		process.Signal(s)
		return
	}(cass)
	return
}

// is not doing a straight upsert, we need to select then update
func (cass *CassandraFlatWriter) mergeWrite(stat *repr.StatRepr) *repr.StatRepr {
	if cass.insert_mode { // true means upsert
		return stat
	}

	time := stat.Time.UnixNano()

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	iter := cass.conn.Query(
		cass._select_time_query,
		stat.Name.Key, stat.Name.Resolution, time,
	).Iter()

	var t, count int64
	var min, max, sum, last float64

	for iter.Scan(&max, &min, &sum, &last, &count, &t) {
		// only one here
		n_stat := &repr.StatRepr{
			Time:  stat.Time,
			Last:  repr.JsonFloat64(last),
			Count: count,
			Sum:   repr.JsonFloat64(sum),
			Min:   repr.JsonFloat64(min),
			Max:   repr.JsonFloat64(max),
		}
		return stat.Merge(n_stat)
	}
	return stat
}

// listen to the overflow chan from the cache and attempt to write "now"
func (cass *CassandraFlatWriter) overFlowWrite() {
	for {
		select {
		case item, more := <-cass.cacheOverFlowListener.Ch:

			// bail
			if cass.shutitdown || !more {
				return
			}
			statitem := item.(*TotalTimeSeries)
			// need to make a list of points from the series
			iter, err := statitem.Series.Iter()
			if err != nil {
				cass.log.Error("error in overflow writer %v", err)
				continue
			}
			pts := make(repr.StatReprSlice, 0)
			for iter.Next() {
				pts = append(pts, iter.ReprValue())
			}
			if iter.Error() != nil {
				cass.log.Error("error in overflow iterator %v", iter.Error())
			}
			cass.log.Debug("Cache overflow force write for %s you may want to do something about that", statitem.Name.Key)
			cass.InsertMulti(statitem.Name, pts)
		}
	}
}

// we can use the batcher effectively for single metric multi point writes as they share the
// the same token
func (cass *CassandraFlatWriter) InsertMulti(name *repr.StatName, points repr.StatReprSlice) (int, error) {

	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandraflat.batch.metric-time-ns"), time.Now())

	l := len(points)
	if l == 0 {
		return 0, nil
	}
	/*if l == 1 {
		return cass.InsertOne(points[0]) // should be faster then the batcher logic
	}*/

	batch := cass.conn.NewBatch(gocql.LoggedBatch)

	for _, stat := range points {
		DO_Q := cass._insert_query
		if stat.Name.TTL > 0 {
			DO_Q += fmt.Sprintf(" USING TTL %d", stat.Name.TTL)
		}
		batch.Query(
			DO_Q,
			name.UniqueIdString(),
			name.Key,
			int64(stat.Name.Resolution),
			stat.Time.UnixNano(),
			float64(stat.Sum),
			float64(stat.Min),
			float64(stat.Max),
			float64(stat.Last),
			stat.Count,
		)
	}
	err := cass.conn.ExecuteBatch(batch)
	if err != nil {
		cass.log.Error("Cassandra Driver:Batch Metric insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandraflat.batch.metric-failures", 1)
		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandraflat.batch.writes", 1)
	stats.StatsdClientSlow.GaugeAvg("writer.cassandraflat.batch.metrics-per-writes", int64(l))

	return l, nil
}

func (cass *CassandraFlatWriter) InsertOne(name *repr.StatName, stat *repr.StatRepr) (int, error) {

	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandraflat.write.metric-time-ns"), time.Now())

	ttl := uint32(0)
	if stat.Name.TTL > 0 {
		ttl = stat.Name.TTL
	}

	Q := cass._insert_query
	if ttl > 0 {
		Q += " USING TTL ?"
	}

	write_stat := cass.mergeWrite(stat)
	err := cass.conn.Query(Q,
		name.UniqueIdString(),
		name.Key,
		int64(stat.Name.Resolution),
		stat.Time.UnixNano(),
		float64(write_stat.Sum),
		float64(write_stat.Min),
		float64(write_stat.Max),
		float64(write_stat.Last),
		write_stat.Count,
		ttl,
	).Exec()

	//cass.log.Critical("METRICS WRITE %d: %v", ttl, stat)
	if err != nil {
		cass.log.Error("Cassandra Driver: insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandraflat.metric-failures", 1)

		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandraflat.metric-writes", 1)

	return 1, nil
}

// pop from the cache and send to actual writers
func (cass *CassandraFlatWriter) sendToWriters() error {
	// this may not be the "greatest" rate-limiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backed-up
	// it will block on the write_queue stage

	//ye old unlimited
	if cass.writes_per_second <= 0 {
		cass.log.Notice("Starting metric writer: No Write limiter")

		for {
			if cass.shutitdown {
				return nil
			}

			name, points := cass.cacher.Pop()
			switch points {
			case nil:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandraflat.write.send-to-writers"), 1)
				cass.dispatcher.Add(&CassandraFlatMetricJob{Cass: cass, Stats: points, Name: name})
			}
		}
	} else {

		sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(cass.writes_per_second))
		cass.log.Notice("Starting metric writer: limiter every %f nanoseconds (%d writes per second)", sleep_t, cass.writes_per_second)
		dur := time.Duration(int(sleep_t))

		for {
			if cass.shutitdown {
				return nil
			}

			name, points := cass.cacher.Pop()

			switch points {
			case nil:
				time.Sleep(time.Second)
			default:

				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandraflat.write.send-to-writers"), 1)
				cass.dispatcher.Add(&CassandraFlatMetricJob{Cass: cass, Stats: points, Name: name})
				time.Sleep(dur)
			}

		}
	}
}

func (cass *CassandraFlatWriter) Write(stat repr.StatRepr) error {

	//cache keys needs metric + resolution
	// turning off
	if !cass.shutitdown {
		cass.cacher.Add(&stat.Name, &stat)
	}

	return nil

}

/****************** Metrics Writer *********************/
type CassandraFlatMetric struct {
	resolutions       [][]int
	currentResolution int
	static_tags       repr.SortingTags
	indexer           indexer.Indexer
	writer            *CassandraFlatWriter

	render_timeout time.Duration
}

func NewCassandraFlatMetrics() *CassandraFlatMetric {
	return new(CassandraFlatMetric)
}

func (cass *CassandraFlatMetric) Driver() string {
	return "cassandra-flat"
}

func (cass *CassandraFlatMetric) Start() {
	cass.writer.Start()
}

func (cass *CassandraFlatMetric) Stop() {
	cass.writer.Stop()
}

func (cass *CassandraFlatMetric) SetIndexer(idx indexer.Indexer) error {
	cass.indexer = idx
	return nil
}

// Resolutions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (cass *CassandraFlatMetric) SetResolutions(res [][]int) int {
	cass.resolutions = res
	return len(res) // need as many writers as bins
}

func (cass *CassandraFlatMetric) SetCurrentResolution(res int) {
	cass.currentResolution = res
}

func (cass *CassandraFlatMetric) Config(conf map[string]interface{}) (err error) {
	gots, err := _get_flat_signelton(conf)
	if err != nil {
		return err
	}
	cass.writer = gots

	resolution := conf["resolution"]
	if resolution == nil {
		return fmt.Errorf("Resulotuion needed for cassandra writer")
	}

	cache_key := fmt.Sprintf("cassandraflat:cache:%s:%v", conf["dsn"], resolution)

	gots.cacher, err = GetCacherSingleton(cache_key)

	if err != nil {
		return err
	}

	rdur, err := time.ParseDuration(CASSANDRA_FLAT_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	cass.render_timeout = rdur

	g_tag, ok := conf["tags"]
	if ok {
		cass.static_tags = repr.SortingTagsFromString(g_tag.(string))
	}

	// prevent a reader from squshing this cacher
	if !gots.cacher.started && !gots.cacher.inited {
		gots.cacher.inited = true
		// set the cacher bits
		_ms := conf["cache_metric_size"]
		if _ms != nil {
			gots.cacher.maxKeys = int(_ms.(int64))
		}

		_ps := conf["cache_byte_size"]
		if _ps != nil {
			gots.cacher.maxBytes = int(_ps.(int64))
		}

		_lf := conf["cache_low_fruit_rate"]
		if _lf != nil {
			gots.cacher.lowFruitRate = _lf.(float64)
		}

		_st := conf["cache_series_type"]
		if _st != nil {
			gots.cacher.seriesType = _st.(string)
		}

		_ov := conf["cache_overflow_method"]
		if _ov != nil {
			gots.cacher.overFlowMethod = _ov.(string)
			// set th overflow chan, and start the listener for that channel
			if gots.cacher.overFlowMethod == "chan" {
				gots.cacheOverFlowListener = gots.cacher.GetOverFlowChan()
				go gots.overFlowWrite()
			}
		}
	}

	return nil
}

// simple proxy to the cacher
func (cass *CassandraFlatMetric) Write(stat repr.StatRepr) error {
	// write the index from the cache as indexing can be slooowwww
	// keep note of this, when things are not yet "warm" (the indexer should
	// keep tabs on what it's already indexed for speed sake,
	// the push "push" of stats will cause things to get pretty slow for a while
	stat.Name.MergeMetric2Tags(cass.static_tags)
	cass.indexer.Write(stat.Name)
	return cass.writer.Write(stat)
}

/************************ READERS ****************/

// based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
// from and to needs to be > then the TTL as well
func (cass *CassandraFlatMetric) getResolution(from int64, to int64) uint32 {
	diff := int(math.Abs(float64(to - from)))
	n := int(time.Now().Unix())
	back_f := n - int(from)
	back_t := n - int(to)
	for _, res := range cass.resolutions {
		if diff < res[1] && back_f < res[1] && back_t < res[1] {
			return uint32(res[0])
		}
	}
	return uint32(cass.resolutions[len(cass.resolutions)-1][0])
}

func (cass *CassandraFlatMetric) RawRenderOne(metric indexer.MetricFindItem, start int64, end int64) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandraflat.renderraw.get-time-ns", time.Now())

	rawd := new(RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("Cassandra: RawRenderOne: Not a data node")
	}

	//figure out the best res
	resolution := cass.getResolution(start, end)

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	b_len := uint32(end-start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, fmt.Errorf("Cassandra: RawRenderOne: time too narrow")
	}

	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nano_end := end * nano
	nano_start := start * nano

	first_t := uint32(start)
	last_t := uint32(end)

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	iter := cass.writer.conn.Query(
		cass.writer._select_time_query,
		metric.UniqueId, metric.Id, resolution, nano_end, nano_start,
	).Iter()

	var t, count int64
	var min, max, sum, last float64

	m_key := metric.Id

	var d_points []RawDataPoint

	ct := uint32(0)
	// sorting order for the table is time ASC (i.e. first_t == first entry)

	for iter.Scan(&max, &min, &sum, &last, &count, &t) {
		on_t := uint32(t / nano) // back convert to seconds

		d_points = append(d_points, RawDataPoint{
			Count: count,
			Sum:   sum,
			Max:   max,
			Min:   min,
			Last:  last,
			Time:  on_t,
		})
		//cass.log.Critical("POINT %s time:%d data:%f", metric.Id, on_t, sum)
		ct++
		last_t = on_t
	}

	if err := iter.Close(); err != nil {
		cass.writer.log.Error("RawRender: Failure closing iterator: %v", err)
	}

	if ct > 0 && d_points[0].Time > 0 {
		first_t = d_points[0].Time
	}

	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, first_t, last_t, len(d_points), ct)

	rawd.RealEnd = uint32(last_t)
	rawd.RealStart = uint32(first_t)
	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	rawd.Step = resolution
	rawd.Metric = m_key
	rawd.Id = metric.UniqueId
	rawd.Data = d_points
	rawd.AggFunc = repr.GuessReprValueFromKey(m_key)

	return rawd, nil
}

func (cass *CassandraFlatMetric) RenderOne(metric indexer.MetricFindItem, from int64, to int64) (WhisperRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandraflat.renderone.get-time-ns", time.Now())

	var whis WhisperRenderItem

	rawd, err := cass.RawRenderOne(metric, from, to)

	if err != nil {
		return whis, err
	}
	whis.RealEnd = rawd.RealEnd
	whis.RealStart = rawd.RealStart
	whis.Start = rawd.Start
	whis.End = rawd.End
	whis.Step = rawd.Step
	whis.Series = make(map[string][]DataPoint)

	// which value to actually return
	use_metric := metric.SelectValue()

	m_key := metric.Id
	stat_name := metric.StatName()
	stat_name.Resolution = rawd.Step
	b_len := (rawd.End - rawd.Start) / rawd.Step //"proper" length of the metric

	// Since graphite does not care about the actual time stamp, but assumes
	// a "constant step" in time. Since data may not necessarily "be there" for a given
	// interval we need to "insert nils" for steps that don't really exist
	// as basically (start - end) / resolution needs to match
	// the vector length, in effect we need to "interpolate" the vector to match sizes

	// now for the interpolation bit .. basically leaving "times that have no data as nulls"
	// XXX HOPEFULLY there are usually FEWER or as much "real" data then "wanted" by the resolution
	// if there's not :boom: and you should really keep tabs on who is messing with your data in the DB
	interp_vec := make([]DataPoint, b_len)
	cur_step_time := rawd.Start
	d_points := rawd.Data
	ct := uint32(len(d_points))
	var last_got_t uint32
	var last_got_index uint32

	// grab from cache too if not yet written
	_, inflight, err := cass.writer.cacher.GetById(stat_name.UniqueId())

	// debuggers
	//cass.writer.log.Critical("%s", s_key)
	//cass.writer.cacher.DumpPoints(inflight)
	inflight_len := uint32(len(inflight))

	if ct > 0 { // got something from cassandra, make sure to fill any "missing times" w/ nils
		j := uint32(0)
		for i := uint32(0); i < b_len; i++ {

			interp_vec[i] = DataPoint{Time: cur_step_time, Value: nil}

			for ; j < ct; j++ {
				d := d_points[j]
				if d.Time <= cur_step_time {

					// cass.writer.log.Critical("ONPS %v : %v Len %d :I %d, j %d, iLen: %v SUM: %v", d_points[j], interp_vec[i], ct, i, j, b_len, d_points[j].Sum,)
					use_value := d.AggValue(use_metric)
					interp_vec[i].Value = &use_value
					interp_vec[i].Time = d.Time //this is the "real" time, graphite does not care, but something might
					last_got_t = d.Time
					last_got_index = j
					j++
				}
				break
			}
			cur_step_time += rawd.Step
		}

		// now attempt to merge the inflight data
		if len(inflight) > 0 && err == nil && last_got_t <= cur_step_time {
			j := uint32(0)
			for i := last_got_index; i < b_len; i++ {
				for ; j < inflight_len; j++ {
					d := inflight[j]
					if uint32(d.Time.Unix()) <= interp_vec[i].Time {
						use_value := d.AggValue(use_metric)
						interp_vec[i].Value = &use_value
						j++
					}
					break
				}
			}
		}
	} else if len(inflight) > 0 && err == nil { // no data in cassandra yet, use inflight
		//fill it up
		j := uint32(0)
		for i := uint32(0); i < b_len; i++ {
			interp_vec[i] = DataPoint{Time: cur_step_time, Value: nil}
			for ; j < inflight_len; j++ {
				d := inflight[j]
				if uint32(d.Time.Unix()) <= cur_step_time {

					// cass.writer.log.Critical("ONPS %v : %v Len %d :I %d, j %d, iLen: %v SUM: %v", d_points[j], interp_vec[i], ct, i, j, b_len, d_points[j].Sum,)
					use_value := d.AggValue(use_metric)
					interp_vec[i].Value = &use_value
					interp_vec[i].Time = uint32(d.Time.Unix()) //this is the "real" time, graphite does not care, but something might
					j++
				}
				break
			}
			cur_step_time += rawd.Step
		}
	}
	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, first_t, last_t, len(d_points), ct)

	whis.Series[m_key] = interp_vec

	return whis, nil
}

func (cass *CassandraFlatMetric) RawRender(path string, from int64, to int64) ([]*RawRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandraflat.rawrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	for _, pth := range paths {
		mets, err := cass.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*RawRenderItem, len(metrics), len(metrics))

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	// ye old fan out technique
	render_one := func(metric indexer.MetricFindItem, idx int) {
		defer render_wg.Done()
		_ri, err := cass.RawRenderOne(metric, from, to)
		if err != nil {
			return
		}
		rawd[idx] = _ri
		return
	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(metric, idx)
	}
	render_wg.Wait()
	return rawd, nil
}

func (cass *CassandraFlatMetric) CacheRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error) {
	return nil, fmt.Errorf("CassandraFlatMetric: CacheRender: NOT YET IMPLIMNETED")
}
func (cass *CassandraFlatMetric) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*TotalTimeSeries, error) {
	return nil, fmt.Errorf("CassandraFlatMetric: CachedSeries: NOT YET IMPLIMNETED")
}
