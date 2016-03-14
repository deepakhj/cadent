/*
	The Cassandra Metric Reader/Writer

	The table should have this schema to match the repr item
	The same as the writer items


		keyspace: base keyspace name (default: metric)
		metric_table: base table name (default: metric)
		write_consistency: "one"
		read_consistency: "one"
		port: 9042
		numcons: 5  (connection pool size)
		timeout: "30s"
		user: ""
		pass: ""
		# NOPE: batch_count: batch this many inserts for much faster insert performance (default 1000)
		# NOPE: periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package metrics

import (
	"consthash/server/lrucache"
	"consthash/server/repr"
	"consthash/server/stats"
	"consthash/server/writers/dbs"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"consthash/server/dispatch"
	"consthash/server/writers/indexer"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	CASSANDRA_RESULT_CACHE_SIZE = 1024 * 1024 * 100
	CASSANDRA_RESULT_CACHE_TTL  = 10 * time.Second
	CASSANDRA_METRIC_WORKERS    = 128
	CASSANDRA_METRIC_QUEUE_LEN  = 1024 * 100
)

/** Being Cassandra we need some mappings to match the schemas **/

/**
	CREATE TYPE metric_point (
        max double,
        mean double,
        min double,
        sum double,
        count int
    );
*/

type CassMetricPoint struct {
	Max   float64
	Mean  float64
	Min   float64
	Sum   float64
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

/****************** Writer *********************/
type CassandraMetric struct {
	// juse the writer connections for this
	db          *dbs.CassandraDB
	conn        *gocql.Session
	resolutions [][]int
	indexer     indexer.Indexer

	render_wg sync.WaitGroup
	render_mu sync.Mutex

	write_list       []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	write_queue      chan dispatch.IJob
	dispatch_queue   chan chan dispatch.IJob
	write_dispatcher *dispatch.Dispatch
	max_write_size   int           // size of that buffer before a flush
	max_idle         time.Duration // either max_write_size will trigger a write or this time passing will
	write_lock       sync.Mutex

	log *logging.Logger

	datacache *lrucache.TTLLRUCache
}

func NewCassandraMetrics() *CassandraMetric {
	cass := new(CassandraMetric)
	cass.log = logging.MustGetLogger("reader.cassandra")
	cass.datacache = lrucache.NewTTLLRUCache(CASSANDRA_RESULT_CACHE_SIZE, CASSANDRA_RESULT_CACHE_TTL)
	return cass
}

func (cass *CassandraMetric) SetIndexer(idx indexer.Indexer) error {
	cass.indexer = idx
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (cass *CassandraMetric) SetResolutions(res [][]int) {
	cass.resolutions = res
}

func (cass *CassandraMetric) Config(conf map[string]interface{}) (err error) {

	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}
	dsn := gots.(string)

	db, err := dbs.NewDB("cassandra", dsn, conf)
	if err != nil {
		return err
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

	//go cass.PeriodFlush()

	return nil
}

func (cass *CassandraMetric) PeriodFlush() {
	for {
		time.Sleep(cass.max_idle)
		cass.Flush()
	}
	return
}

/*
func (cass *CassandraMetric) consumeWriter() {
	for {
		select {
		case stat := <-cass.write_queue:
			cass.InsertOne(stat)
		}
	}
	return
}
*/

// note this is not really used.  Batching in cassandra is not always a good idea
// since the tocken-aware insert will choose the proper server set, where as in batch mode
// that is not the case
func (cass *CassandraMetric) Flush() (int, error) {
	cass.write_lock.Lock()
	defer cass.write_lock.Unlock()
	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.flush.metric-time-ns"), time.Now())

	l := len(cass.write_list)
	if l == 0 {
		return 0, nil
	}

	batch := cass.conn.NewBatch(gocql.LoggedBatch)
	Q := fmt.Sprintf(
		"INSERT INTO %s (id, time, point) VALUES  ({path: ?, resolution: ?}, ?, {sum: ?, mean: ?, min: ?, max: ?, count: ?})",
		cass.db.MetricTable(),
	)

	for _, stat := range cass.write_list {
		DO_Q := Q
		if stat.TTL > 0 {
			DO_Q += fmt.Sprintf(" USING TTL %d", stat.TTL)
		}
		batch.Query(
			DO_Q,
			stat.Key,
			int64(stat.Resolution),
			stat.Time.UnixNano(),
			float64(stat.Sum),
			float64(stat.Mean),
			float64(stat.Min),
			float64(stat.Max),
			stat.Count,
		)
	}
	err := cass.conn.ExecuteBatch(batch)
	if err != nil {
		cass.log.Error("Cassandra Driver:Batch Metric insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandra.metric-failures", 1)
		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandra.metric-writes", int64(l))
	stats.StatsdClientSlow.Incr("writer.cassandra.batch-writes", 1)

	cass.write_list = nil
	cass.write_list = []repr.StatRepr{}
	return l, nil
}

/**** WRITER ****/

func (cass *CassandraMetric) InsertOne(stat repr.StatRepr) (int, error) {

	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.write.metric-time-ns"), time.Now())

	ttl := int64(0)
	if stat.TTL > 0 {
		ttl = stat.TTL
	}

	Q := fmt.Sprintf(
		"INSERT INTO %s (id, time, point) VALUES  ({path: ?, resolution: ?}, ?, {sum: ?, mean: ?, min: ?, max: ?, count: ?})",
		cass.db.MetricTable(),
	)
	if ttl > 0 {
		Q += fmt.Sprintf(" USING TTL ?", ttl)
	}

	err := cass.conn.Query(Q,
		stat.Key,
		int64(stat.Resolution),
		stat.Time.UnixNano(),
		float64(stat.Sum),
		float64(stat.Mean),
		float64(stat.Min),
		float64(stat.Max),
		stat.Count,
		ttl,
	).Exec()
	//cass.log.Critical("METRICS WRITE %d: %v", ttl, stat)
	if err != nil {
		cass.log.Error("Cassandra Driver: insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandra.metric-failures", 1)

		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandra.metric-writes", 1)

	return 1, nil
}

func (cass *CassandraMetric) Write(stat repr.StatRepr) error {

	/**** dispatcher queue ***/
	if cass.write_queue == nil {
		workers := CASSANDRA_METRIC_WORKERS
		cass.write_queue = make(chan dispatch.IJob, CASSANDRA_METRIC_QUEUE_LEN)
		cass.dispatch_queue = make(chan chan dispatch.IJob, workers)
		cass.write_dispatcher = dispatch.NewDispatch(workers, cass.dispatch_queue, cass.write_queue)
		cass.write_dispatcher.SetRetries(2)
		cass.write_dispatcher.Run()
	}

	cass.write_queue <- CassandraMetricJob{Stat: stat, Cass: cass}
	return nil

	/**** single write queue to keep a connection depletion in cassandra
	if cass.write_queue == nil {
		cass.write_queue = make(chan repr.StatRepr, cass.db.Cluster().NumConns*100)
		for i := 0; i < cass.db.Cluster().NumConns; i++ {
			go cass.consumeWriter()
		}
	}

	cass.write_queue <- stat
	return nil
	***/

	/** Direct insert_one tech
	go cass.InsertOne(stat)
	return nil
	**/

	/* If cassandra batching is faster ... use this .. (cassandra batching is not like mysql batching)
	if len(cass.write_list) > cass.max_write_size {
		_, err := cass.Flush()
		if err != nil {
			return err
		}
	}

	// Flush can cause double locking
	cass.write_lock.Lock()
	defer cass.write_lock.Unlock()
	cass.write_list = append(cass.write_list, stat)
	return nil
	*/
}

/************************ READERS ****************/

// based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
// from and to needs to be > then the TTL as well
func (cass *CassandraMetric) getResolution(from int64, to int64) int {
	diff := int(math.Abs(float64(to - from)))
	n := int(time.Now().Unix())
	back_f := n - int(from)
	back_t := n - int(to)
	for _, res := range cass.resolutions {
		if diff < res[1] && back_f < res[1] && back_t < res[1] {
			return res[0]
		}
	}
	return cass.resolutions[len(cass.resolutions)-1][0]
}

// based on the resolution attempt to round start/end nicely by the resolutions
func (cass *CassandraMetric) truncateTo(num int64, mod int) int64 {
	_mods := int(math.Mod(float64(num), float64(mod)))
	if _mods < mod/2 {
		return num - int64(_mods)
	}
	return num + int64(mod-_mods)
}

func (cass *CassandraMetric) RenderOne(path string, from string, to string) (WhisperRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.renderone.get-time-ns", time.Now())

	var whis WhisperRenderItem

	start, err := ParseTime(from)
	if err != nil {
		cass.log.Error("Invalid from time `%s` :: %v", from, err)
		return whis, err
	}

	end, err := ParseTime(to)
	if err != nil {
		cass.log.Error("Invalid from time `%s` :: %v", to, err)
		return whis, err
	}
	if end < start {
		start, end = end, start
	}
	//figure out the best res
	resolution := cass.getResolution(start, end)

	start = cass.truncateTo(start, resolution)
	end = cass.truncateTo(end, resolution)

	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nano_end := end * nano
	nano_start := start * nano

	metrics, err := cass.indexer.Find(path)
	if err != nil {
		return whis, err
	}

	series := make(map[string][]DataPoint)

	first_t := int(start)
	last_t := int(end)

	for _, metric := range metrics {
		if metric.Leaf == 0 { //data only
			continue
		}
		b_len := int(end-start) / resolution //just to be safe
		if b_len <= 0 {
			continue
		}

		// check our cache
		cache_id := fmt.Sprintf("%s-%s-%s", metric.Id, from, to)
		//cass.log.Critical("Cache  Key %s", cache_id)
		gots, ok := cass.datacache.Get(cache_id)
		if ok {
			// cast the punk
			data := gots.(WhisperRenderCacher).Data
			series[metric.Id] = data
			first_t = data[0].Time
			last_t = data[len(data)-1].Time
			cass.log.Critical("Got Cache %s", cache_id)
			continue
		}

		// grab ze data. (note data is already sorted by time asc va the cassandra schema)
		cass_Q := fmt.Sprintf(
			"SELECT point.mean, point.max, point.min, point.sum, time FROM %s WHERE id={path: ?, resolution: ?} AND time <= ? and time >= ?",
			cass.db.MetricTable(),
		)
		iter := cass.conn.Query(cass_Q,
			metric.Id, resolution, nano_end, nano_start,
		).Iter()

		var t int64
		var mean, min, max, sum float64

		// which value to acctually return
		use_metric := metric.SelectValue()

		// use mins or maxes for the "upper_xxx, lower_xxx"

		m_key := metric.Id

		d_points := make([]DataPoint, b_len)

		// Since graphite does not care about the actual time stamp, but assumes
		// a "constant step" in time. Since data may not nessesarily "be there" for a given
		// interval we need to "insert nils" for steps that don't really exist
		// as basically (start - end) / resolution needs to match
		// the vector length, in effect we need to "interpolate" the vector to match sizes

		ct := 0
		// sorting order for the table is time ASC (i.e. first_t == first entry)

		for iter.Scan(&mean, &max, &min, &sum, &t) {
			on_t := int(t / nano) // back convert to seconds
			d_point := NewDataPoint(on_t, mean)
			switch use_metric {
			case "mean":
				d_point.SetValue(mean)
			case "min":
				d_point.SetValue(min)
			case "max":
				d_point.SetValue(max)
			default:
				d_point.SetValue(sum)
			}

			if len(d_points) <= ct {
				_t_pts := make([]DataPoint, len(d_points)+100)
				copy(_t_pts, d_points)
				d_points = _t_pts
			}
			d_points[ct] = d_point
			//cass.log.Critical("POINT %s time:%d data:%f", metric.Id, on_t, mean)
			ct++
			last_t = on_t
		}

		if err := iter.Close(); err != nil {
			cass.log.Error("Render: Failure closing iterator: %v", err)
		}

		if len(d_points) > 0 && d_points[0].Time > 0 {
			first_t = d_points[0].Time
		}

		// now for the interpolation bit .. basically leaving "times that have no data as nulls"
		// XXX HOPEFULLY there are usually FEWER or as much "real" data then "wanted" by the resolution
		// if there's not :boom: and you should really keep tabs on who is messing with your data in the DB
		interp_vec := make([]DataPoint, b_len)
		cur_step_time := int(start)

		if ct > 0 {
			j := 0
			for i := int(0); i < b_len; i++ {

				interp_vec[i] = DataPoint{Time: cur_step_time, Value: nil}
				cur_step_time += resolution

				for ; j < ct; j++ {
					if d_points[j].Time <= cur_step_time {
						interp_vec[i].Value = d_points[j].Value
						interp_vec[i].Time = d_points[j].Time //this is the "real" time, graphite does not care, but something might
						j++
					}
					break
				}

			}
		}
		series[m_key] = interp_vec
		cass.datacache.Set(cache_id, WhisperRenderCacher{Data: interp_vec})

		//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, first_t, last_t, len(d_points), ct)
	}

	whis.Series = series
	whis.RealStart = first_t
	whis.RealEnd = last_t
	whis.Start = int(start)
	whis.End = int(end)
	whis.Step = resolution

	return whis, nil
}

func (cass *CassandraMetric) Render(path string, from string, to string) (WhisperRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.render.get-time-ns", time.Now())

	var whis WhisperRenderItem
	whis.Series = make(map[string][]DataPoint)
	paths := strings.Split(path, ",")

	// ye old fan out technique
	render_one := func(pth string) {
		_ri, err := cass.RenderOne(pth, from, to)
		if err != nil {
			cass.render_wg.Done()
			return
		}
		cass.render_mu.Lock()
		for k, rr := range _ri.Series {
			whis.Series[k] = rr
		}
		whis.Start = _ri.Start
		whis.End = _ri.End
		whis.Step = _ri.Step

		cass.render_mu.Unlock()
		cass.render_wg.Done()
		return
	}

	for _, pth := range paths {
		if len(pth) == 0 {
			continue
		}
		cass.render_wg.Add(1)
		go render_one(pth)
	}
	cass.render_wg.Wait()
	return whis, nil

}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type CassandraMetricJob struct {
	Cass  *CassandraMetric
	Stat  repr.StatRepr
	retry int
}

func (j CassandraMetricJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j CassandraMetricJob) OnRetry() int {
	return j.retry
}

func (j CassandraMetricJob) DoWork() error {
	_, err := j.Cass.InsertOne(j.Stat)
	return err
}
