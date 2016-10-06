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
	THe MySQL flat stat write

     CREATE TABLE `{metrics-table}{resolutionprefix}` (
      `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) CHARACTER SET ascii NOT NULL ,
      `path` varchar(255) NOT NULL DEFAULT '',
      `sum` float NOT NULL,
      `min` float NOT NULL,
      `max` float NOT NULL,
      `first` float NOT NULL,
      `last` float NOT NULL,
      `count` float NOT NULL,
      `time` datetime(6) NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`),
      KEY `path` (`path`),
      KEY `time` (`time`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	OPTIONS: For `Config`

		table: base table name (default: metrics)
		batch_count: batch this many inserts for much faster insert performance (default 1000)
		periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"
)

/****************** Interfaces *********************/
type MySQLFlatMetrics struct {
	WriterBase

	db   *dbs.MySQLDB
	conn *sql.DB

	write_list     []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	max_write_size int             // size of that buffer before a flush
	max_idle       time.Duration   // either max_write_size will trigger a write or this time passing will
	write_lock     sync.Mutex
	renderTimeout  time.Duration

	log *logging.Logger

	shutdown chan bool
}

func NewMySQLFlatMetrics() *MySQLFlatMetrics {
	my := new(MySQLFlatMetrics)
	my.log = logging.MustGetLogger("writers.mysqlflat")
	return my
}

func (my *MySQLFlatMetrics) Config(conf *options.Options) error {

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}

	db_key := dsn + conf.String("table", "metrics")
	db, err := dbs.NewDB("mysql", db_key, conf)
	if err != nil {
		return err
	}

	my.db = db.(*dbs.MySQLDB)
	my.conn = db.Connection().(*sql.DB)

	res, err := conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resulotuion needed for cassandra writer")
	}

	//need to hide the usr/pw from things
	p, _ := url.Parse("mysql://" + dsn)
	cache_key := fmt.Sprintf("mysqlflat:cache:%s/%s:%v", p.Host, conf.String("table", "metrics"), res)
	my.cacher, err = GetCacherSingleton(cache_key)

	if err != nil {
		return err
	}

	my.max_write_size = int(conf.Int64("batch_count", 1000))
	my.max_idle = conf.Duration("periodic_flush", time.Duration(time.Second))
	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		my.static_tags = repr.SortingTagsFromString(_tgs)
	}

	my.shutitdown = false
	my.shutdown = make(chan bool)

	rdur, err := time.ParseDuration(MYSQL_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	my.renderTimeout = rdur

	return nil
}

func (my *MySQLFlatMetrics) Driver() string {
	return "mysql-flat"
}

func (my *MySQLFlatMetrics) Stop() {
	my.startstop.Stop(func() {
		if my.shutitdown {
			return
		}
		my.shutitdown = true
		shutdown.AddToShutdown()
		my.shutdown <- true
	})
	return
}

func (my *MySQLFlatMetrics) Start() {
	my.startstop.Start(func() {
		// now we make sure the metrics schemas are added
		err := NewMySQLMetricsSchema(my.conn, my.db.RootMetricsTableName(), my.resolutions, "flat").AddMetricsTable()
		if err != nil {
			panic(err)
		}
		my.cacher.Start()
		go my.PeriodFlush()
	})
}

func (my *MySQLFlatMetrics) SetIndexer(idx indexer.Indexer) error {
	my.indexer = idx
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (my *MySQLFlatMetrics) SetResolutions(res [][]int) int {
	my.resolutions = res
	return len(res) // need as many writers as bins
}

func (my *MySQLFlatMetrics) SetCurrentResolution(res int) {
	my.currentResolution = res
}

func (my *MySQLFlatMetrics) PeriodFlush() {
	for {
		select {
		case <-my.shutdown:
			my.Flush()
			shutdown.ReleaseFromShutdown()
			return
		default:
			time.Sleep(my.max_idle)
			my.Flush()
		}
	}
}

func (my *MySQLFlatMetrics) Flush() (int, error) {
	my.write_lock.Lock()
	defer my.write_lock.Unlock()

	l := len(my.write_list)
	if l == 0 {
		return 0, nil
	}

	Q := fmt.Sprintf(
		"INSERT IGNORE INTO %s_%ds (uid, path, sum, min, max, last, count, time) VALUES ",
		my.db.RootMetricsTableName(), my.currentResolution,
	)

	vals := []interface{}{}

	for _, stat := range my.write_list {
		Q += "(?,?,?,?,?,?,?,?), "
		vals = append(
			vals, stat.Name.UniqueIdString(), stat.Name.Key, stat.Sum, stat.Min, stat.Max, stat.Last, stat.Count, stat.Time.UnixNano(),
		)
	}

	//trim the last ", "
	Q = Q[0 : len(Q)-2]

	//prepare the statement
	stmt, err := my.conn.Prepare(Q)
	if err != nil {
		my.log.Error("Mysql Driver: Metric prepare failed, %v", err)
		return 0, err
	}
	defer stmt.Close()

	//format all vals at once
	_, err = stmt.Exec(vals...)
	if err != nil {
		my.log.Error("Mysql Driver: Metric insert failed, %v", err)
		return 0, err
	}

	my.write_list = nil
	my.write_list = []repr.StatRepr{}
	return l, nil
}

func (my *MySQLFlatMetrics) Write(stat repr.StatRepr) error {
	stat.Name.MergeMetric2Tags(my.static_tags)

	if len(my.write_list) > my.max_write_size {
		_, err := my.Flush()
		if err != nil {
			return err
		}
	}

	my.indexer.Write(stat.Name) // to the indexer

	// Flush can cause double locking
	my.write_lock.Lock()
	my.write_list = append(my.write_list, stat)
	my.write_lock.Unlock()
	return nil
}

/**** READER ***/

// based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
// from and to needs to be > then the TTL as well

func (my *MySQLFlatMetrics) getResolution(from int64, to int64) uint32 {
	diff := int(math.Abs(float64(to - from)))
	n := int(time.Now().Unix())
	back_f := n - int(from)
	back_t := n - int(to)
	for _, res := range my.resolutions {
		if diff <= res[1] && back_f <= res[1] && back_t <= res[1] {
			return uint32(res[0])
		}
	}
	return uint32(my.resolutions[len(my.resolutions)-1][0])
}

func (my *MySQLFlatMetrics) RawRenderOne(metric indexer.MetricFindItem, start int64, end int64, resample uint32) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.mysqlflat.renderraw.get-time-ns", time.Now())

	rawd := new(RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("Mysql: RawRenderOne: Not a data node")
	}

	//figure out the best res
	resolution := my.getResolution(start, end)
	out_resolution := resolution

	//obey the bigger
	if resample > resolution {
		out_resolution = resample
	}

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	b_len := uint32(end-start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, fmt.Errorf("Mysql: RawRenderOne: time too narrow")
	}

	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nano_end := end * nano
	nano_start := start * nano

	first_t := uint32(start)
	last_t := uint32(end)

	// try the write inflight cache as nothing is written yet
	stat_name := metric.StatName()
	inflight_renderitem, err := my.cacher.GetAsRawRenderItem(stat_name)

	// need at LEAST 2 points to get the proper step size
	if inflight_renderitem != nil && err == nil {
		// move the times to the "requested" ones and quantize the list
		inflight_renderitem.Metric = metric.Id
		inflight_renderitem.Tags = metric.Tags
		inflight_renderitem.MetaTags = metric.MetaTags
		inflight_renderitem.Id = metric.UniqueId
		inflight_renderitem.AggFunc = stat_name.AggType()
		if inflight_renderitem.Start < uint32(start) {
			inflight_renderitem.RealEnd = uint32(end)
			inflight_renderitem.RealStart = uint32(start)
			inflight_renderitem.Start = inflight_renderitem.RealStart
			inflight_renderitem.End = inflight_renderitem.RealEnd
			return inflight_renderitem, err
		}
	}

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	Q := fmt.Sprintf("SELECT max, min, sum, last, count, time FROM %s_%ds WHERE uid=? AND time BETWEEN ? AND ? ORDER BY time ASC",
		my.db.RootMetricsTableName(), resolution,
	)

	iter, err := my.conn.Query(Q, metric.UniqueId, nano_start, nano_end)
	my.log.Critical("%s %v %v %v", Q, metric.UniqueId, nano_start, nano_end)
	if err != nil {
		return rawd, err
	}

	var t, count int64
	var min, max, sum, last float64

	m_key := metric.Id

	t_start := uint32(start)
	cur_pt := NullRawDataPoint(t_start)
	// sorting order for the table is time ASC (i.e. first_t == first entry)
	// on resamples (if >0 ) we simply merge points until we hit the time steps
	do_resample := resample > 0 && resample > resolution

	for iter.Next() {
		err = iter.Scan(&max, &min, &sum, &last, &count, &t)
		if err != nil {
			my.log.Errorf("error reading: %v", err)
			continue
		}
		t := uint32(time.Unix(0, t).Unix())
		if do_resample {
			if t >= t_start+resample {
				t_start += resample
				rawd.Data = append(rawd.Data, cur_pt)
				cur_pt = RawDataPoint{
					Count: count,
					Sum:   sum,
					Max:   max,
					Min:   min,
					Last:  last,
					Time:  t,
				}
			} else {
				cur_pt.Merge(&RawDataPoint{
					Count: count,
					Sum:   sum,
					Max:   max,
					Min:   min,
					Last:  last,
					Time:  t,
				})
			}
		} else {
			rawd.Data = append(rawd.Data, RawDataPoint{
				Count: count,
				Sum:   sum,
				Max:   max,
				Min:   min,
				Last:  last,
				Time:  t,
			})
		}
		last_t = t
	}

	if err := iter.Close(); err != nil {
		my.log.Error("RawRender: Failure closing iterator: %v", err)
	}

	if len(rawd.Data) > 0 && rawd.Data[0].Time > 0 {
		first_t = rawd.Data[0].Time
	}

	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, first_t, last_t, len(d_points), ct)

	rawd.RealEnd = uint32(last_t)
	rawd.RealStart = uint32(first_t)
	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	rawd.Step = out_resolution
	rawd.Metric = m_key
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.Id = metric.UniqueId
	rawd.AggFunc = stat_name.AggType()

	// grab the "current inflight" from the cache and merge into the main array
	if inflight_renderitem != nil && len(inflight_renderitem.Data) > 1 {
		//merge with any inflight bits (inflight has higher precedence over the file)
		inflight_renderitem.MergeWithResample(rawd, out_resolution)
		return inflight_renderitem, nil
	}

	return rawd, nil
}

func (my *MySQLFlatMetrics) RawRender(path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandraflat.rawrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	for _, pth := range paths {
		mets, err := my.indexer.Find(pth, tags)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*RawRenderItem, 0, len(metrics))

	procs := MYSQL_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan indexer.MetricFindItem, len(metrics))
	results := make(chan *RawRenderItem, len(metrics))

	render_one := func(met indexer.MetricFindItem) *RawRenderItem {
		_ri, err := my.RawRenderOne(met, from, to, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.mysqlflat.rawrender.errors", 1)
			my.log.Error("Read Error for %s (%d->%d) : %v", path, from, to, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	job_worker := func(jober int, taskqueue <-chan indexer.MetricFindItem, resultqueue chan<- *RawRenderItem) {
		rec_chan := make(chan *RawRenderItem, 1)
		for met := range taskqueue {
			go func() { rec_chan <- render_one(met) }()
			select {
			case <-time.After(my.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.mysqlflat.rawrender.timeouts", 1)
				my.log.Errorf("Render Timeout for %s (%d->%d)", path, from, to)
				resultqueue <- nil
			case res := <-rec_chan:
				resultqueue <- res
			}
		}
	}

	for i := 0; i < procs; i++ {
		go job_worker(i, jobs, results)
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
	return rawd, nil
}

func (my *MySQLFlatMetrics) CacheRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error) {
	return nil, fmt.Errorf("MySQLFlatMetrics: CacheRender: NOT YET IMPLIMNETED")
}
func (my *MySQLFlatMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*TotalTimeSeries, error) {
	return nil, fmt.Errorf("MySQLFlatMetrics: CachedSeries: NOT YET IMPLIMNETED")
}
