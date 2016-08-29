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
	THe MySQL stat write for "binary" blobs of time series

     CREATE TABLE `{table}{prefix}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) CHARACTER SET ascii NOT NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `ptype` TINYINT NOT NULL,
      `points` blob,
      `stime` BIGINT unsigned NOT NULL,
      `etime` BIGINT unsigned NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`),
      KEY `path` (`path`),
      KEY `time` (`etime`, `stime`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	Prefixes are `_{resolution}s` (i.e. "_" + (uint32 resolution) + "s")

	OPTIONS: For `Config`

	table="metrics"

	# series (and cache) encoding types
	series_encoding="gorilla"

	# the "internal carbon-like-cache" size (ram is your friend)
	# if there are more then this many metric keys in the system, newer ones will be DROPPED
	cache_metric_size=102400

	# number of points per metric to cache before we write them
	# note you'll need AT MOST cache_byte_size * cache_metric_size * 2*8 bytes of RAM
	# this is also the "chunk" size stored in Mysql
	cache_byte_size=8192

	# we write the blob after this much time even if it has not reached the byte limit
	# one hour default
	cache_longest_time=3600s

	# Rollup Type
	# this can be either `cache` or `triggered`
	# cache = means we keep each resolution in Ram and flush when approriate
	# triggered = just keep the lowest resoltuion in ram, upon a write, trigger the other resolutions to get
	# rolledup and written
	# the Cached version can take alot of RAM (NumResolutions * Metrics * BlobSize) but is better for queries
	# that over the "nowish" timescale as things are in RAM
	rollup_type="cached"

*/

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/dispatch"
	"cadent/server/repr"
	"cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
	"math"
	"strings"
	"time"
)

const (
	MYSQL_RENDER_TIMEOUT           = "5s" // 5 second time out on any render
	MYSQL_DEFAULT_ROLLUP_TYPE      = "cached"
	MYSQL_DEFAULT_METRIC_WORKERS   = 16
	MYSQL_DEFAULT_METRIC_QUEUE_LEN = 1024 * 100
	MYSQL_DEFAULT_METRIC_RETRIES   = 2
)

// common errors to avoid GC pressure
var errNotADataNode = errors.New("Mysql: Render: Not a data node")
var errTimeTooSmall = errors.New("Mysql: Render: time too narrow")

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type MysqlBlobMetricJob struct {
	My    *MySQLMetrics
	Ts    *TotalTimeSeries // where the point list live
	retry int
}

func (j MysqlBlobMetricJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j MysqlBlobMetricJob) OnRetry() int {
	return j.retry
}

func (j MysqlBlobMetricJob) DoWork() error {
	err := j.My.doInsert(j.Ts)
	return err
}

/****************** Interfaces *********************/
type MySQLMetrics struct {
	driver            string
	db                *dbs.MySQLDB
	conn              *sql.DB
	indexer           indexer.Indexer
	renderTimeout     time.Duration
	resolutions       [][]int
	currentResolution int
	static_tags       repr.SortingTags

	// this is for Render where we may have several caches, but only "one"
	// cacher get picked for the default render (things share the cache from writers
	// and the api render, but not all the caches, so we need to be able to get the
	// the cache singleton keys
	// `cache:series:seriesMaxMetrics:seriesEncoding:seriesMaxBytes:maxTimeInCache`
	cacherPrefix  string
	cacher        *Cacher
	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

	// if the rolluptype == cached, then we this just uses the internal RAM caches
	// otherwise if "trigger" we only have the lowest res cache, and trigger rollups on write
	rollupType string
	rollup     *RollupMetric
	doRollup   bool

	// dispatcher worker Queue
	num_workers      int
	queue_len        int
	dispatch_retries int
	dispatcher       *dispatch.DispatchQueue

	shutitdown bool // just a flag
	startstop  utils.StartStop

	log *logging.Logger
}

func NewMySQLMetrics() *MySQLMetrics {
	my := new(MySQLMetrics)
	my.driver = "mysql"
	my.log = logging.MustGetLogger("writers.mysql")
	return my
}

func NewMySQLTriggeredMetrics() *MySQLMetrics {
	my := new(MySQLMetrics)
	my.driver = "mysql-triggered"
	my.log = logging.MustGetLogger("writers.mysql")
	return my
}

func (my *MySQLMetrics) Config(conf map[string]interface{}) error {

	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}
	dsn := gots.(string)

	resolution := conf["resolution"]
	if resolution == nil {
		return fmt.Errorf("Resolution needed for mysql writer")
	}

	// newDB is a "cached" item to let us pass the connections around
	db_key := dsn + fmt.Sprintf("%f", resolution)
	db, err := dbs.NewDB("mysql", db_key, conf)
	if err != nil {
		return err
	}

	my.db = db.(*dbs.MySQLDB)
	my.conn = db.Connection().(*sql.DB)

	g_tag, ok := conf["tags"]
	if ok {
		my.static_tags = repr.SortingTagsFromString(g_tag.(string))
	}

	if err != nil {
		return err
	}

	// rolluptype
	my.rollupType = MYSQL_DEFAULT_ROLLUP_TYPE
	_rot, ok := conf["rollup_type"]
	if ok {
		my.rollupType = _rot.(string)
	}

	// tweak queues and worker sizes
	_workers := conf["write_workers"]
	my.num_workers = MYSQL_DEFAULT_METRIC_WORKERS
	if _workers != nil {
		my.num_workers = int(_workers.(int64))
	}

	_qs := conf["write_queue_length"]
	my.queue_len = MYSQL_DEFAULT_METRIC_QUEUE_LEN
	if _qs != nil {
		my.queue_len = int(_qs.(int64))
	}
	_rt := conf["write_queue_retries"]
	my.dispatch_retries = MYSQL_DEFAULT_METRIC_RETRIES
	if _rt != nil {
		my.dispatch_retries = int(_rt.(int64))
	}

	_cache := conf["cache"]
	if _cache == nil {
		return errMetricsCacheRequired
	}
	my.cacher = _cache.(*Cacher)
	my.cacherPrefix = my.cacher.Prefix

	if my.rollupType == "triggered" {
		my.driver = "mysql-triggered"
		my.rollup = NewRollupMetric(my, my.cacher.maxBytes)
	}

	rdur, err := time.ParseDuration(MYSQL_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	my.renderTimeout = rdur

	return nil
}

func (my *MySQLMetrics) Driver() string {
	return my.driver
}

func (my *MySQLMetrics) Start() {
	my.startstop.Start(func() {
		my.log.Notice("Starting mysql writer for %s at %d bytes per series", my.db.Tablename(), my.cacher.maxBytes)

		my.cacher.overFlowMethod = "chan"
		my.cacher.Start()

		// only register this when we start as we really want to consume
		my.cacheOverFlow = my.cacher.GetOverFlowChan()
		go my.overFlowWrite()

		// if the resolutions list is just "one" there is no triggered rollups
		if len(my.resolutions) == 1 {
			my.rollupType = "cached"
		}
		my.doRollup = my.rollupType == "triggered" && my.currentResolution == my.resolutions[0][0]
		// start the rolluper if needed
		if my.doRollup {
			// all but the lowest one
			my.rollup.blobMaxBytes = my.cacher.maxBytes
			my.rollup.SetResolutions(my.resolutions[1:])
			go my.rollup.Start()
		}

		my.dispatcher = dispatch.NewDispatchQueue(
			my.num_workers,
			my.queue_len,
			my.dispatch_retries,
		)
		my.dispatcher.Start()
	})
}

func (my *MySQLMetrics) Stop() {
	my.log.Warning("Stopping Mysql writer for (%s)", my.cacher.Name)
	my.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		if my.shutitdown {
			return // already did
		}
		my.shutitdown = true

		my.cacher.Stop()
		mets := my.cacher.Cache
		mets_l := len(mets)
		my.log.Warning("Shutting down %s and exhausting the queue (%d items) and quiting", my.cacher.Name, mets_l)

		// full tilt write out
		go_do := make(chan *TotalTimeSeries, 16)
		done := make(chan bool, 1)
		go func() {
			for {
				select {
				case s, more := <-go_do:
					if !more {
						return
					}
					stats.StatsdClient.Incr(fmt.Sprintf("writer.cache.shutdown.send-to-writers"), 1)
					my.InsertSeries(s.Name, s.Series)
					if my.doRollup {
						my.rollup.DoRollup(s)
					}
				case <-done:
					return
				}
			}
		}()

		did := 0
		for _, queueitem := range mets {
			if did%100 == 0 {
				my.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
			}
			if queueitem.Series != nil {
				go_do <- &TotalTimeSeries{Name: queueitem.Name, Series: queueitem.Series}
			}
			did++
		}
		close(done)
		close(go_do)
		my.log.Warning("shutdown purge: written %d/%d...", did, mets_l)

		if my.dispatcher != nil {
			my.dispatcher.Stop()
		}

		my.log.Warning("Shutdown finished ... quiting mysql blob writer")
	})

}
func (my *MySQLMetrics) doInsert(ts *TotalTimeSeries) error {
	stats.StatsdClientSlow.Incr("writer.mysql.consume.add", 1)
	//my.log.Debug("Cache Write byte overflow %s.", statitem.Name.Key)
	_, err := my.InsertSeries(ts.Name, ts.Series)
	if err == nil && my.doRollup {
		my.rollup.Add(ts)
	}
	return err
}

// listen to the overflow chan from the cache and attempt to write "now"
func (my *MySQLMetrics) overFlowWrite() {
	for {
		statitem, more := <-my.cacheOverFlow.Ch

		// bail
		if !more {
			return
		}
		stats.StatsdClientSlow.Incr("writer.mysql.queue.add", 1)
		my.dispatcher.Add(
			&MysqlBlobMetricJob{
				My: my,
				Ts: statitem.(*TotalTimeSeries),
			},
		)

	}
}

func (my *MySQLMetrics) SetIndexer(idx indexer.Indexer) error {
	my.indexer = idx
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (my *MySQLMetrics) SetResolutions(res [][]int) int {
	my.resolutions = res
	return len(res) // need as many writers as bins
}

func (my *MySQLMetrics) SetCurrentResolution(res int) {
	my.currentResolution = res
}

func (my *MySQLMetrics) InsertSeries(name *repr.StatName, timeseries series.TimeSeries) (int, error) {
	return my.InsertDBSeries(name, timeseries, name.Resolution)
}

func (my *MySQLMetrics) Write(stat repr.StatRepr) error {
	stat.Name.MergeMetric2Tags(my.static_tags)
	// only need to do this if the first resolution
	if my.currentResolution == my.resolutions[0][0] {
		my.indexer.Write(stat.Name)
	}
	if my.rollupType == "triggered" {
		if my.currentResolution == my.resolutions[0][0] {
			my.cacher.Add(&stat.Name, &stat)
		}
	} else {
		my.cacher.Add(&stat.Name, &stat)
	}
	return nil
}

/************** READING ********************/

// based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
// from and to needs to be > then the TTL as well
func (my *MySQLMetrics) getResolution(from int64, to int64) uint32 {
	diff := int(math.Abs(float64(to - from)))
	n := int(time.Now().Unix())
	back_f := n - int(from)
	back_t := n - int(to)
	for _, res := range my.resolutions {
		if diff < res[1] && back_f < res[1] && back_t < res[1] {
			return uint32(res[0])
		}
	}
	return uint32(my.resolutions[len(my.resolutions)-1][0])
}

func (my *MySQLMetrics) GetFromReadCache(metric string, start int64, end int64) (rawd *RawRenderItem, got bool) {
	rawd = new(RawRenderItem)

	// check read cache
	r_cache := GetReadCache()
	if r_cache == nil {
		stats.StatsdClient.Incr("reader.mysql.render.cache.miss", 1)
		return rawd, false
	}

	t_start := time.Unix(int64(start), 0)
	t_end := time.Unix(int64(end), 0)
	cached_stats, _, _ := r_cache.Get(metric, t_start, t_end)
	var d_points []RawDataPoint
	step := uint32(0)

	// the ReadCache will only have the "sum" point in the mix as that's
	// the designated cached point
	if cached_stats != nil && len(cached_stats) > 0 {
		stats.StatsdClient.Incr("reader.mysql.render.cache.hits", 1)

		f_t := uint32(0)
		for _, stat := range cached_stats {
			t := uint32(stat.Time.Unix())
			d_points = append(d_points, RawDataPoint{
				Count: 1,
				Sum:   float64(stat.Sum),
				Time:  t,
			})
			if f_t <= 0 {
				f_t = t
			}
			if step <= 0 && f_t >= 0 {
				step = t - f_t
			}
		}
		rawd.AggFunc = repr.GuessReprValueFromKey(metric)
		rawd.RealEnd = d_points[len(d_points)-1].Time
		rawd.RealStart = d_points[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.RealEnd + step
		rawd.Metric = metric
		rawd.Step = step
		rawd.Data = d_points
		return rawd, len(d_points) > 0
	} else {
		stats.StatsdClient.Incr("reader.mysql.render.cache.miss", 1)
	}

	return rawd, false
}

// grab the time series from the DBs
func (my *MySQLMetrics) GetFromDatabase(metric *indexer.MetricFindItem, resolution uint32, start int64, end int64) (rawd *RawRenderItem, err error) {
	// i.e metrics_5s
	t_name := my.db.RootMetricsTableName()
	rawd = new(RawRenderItem)

	Q := fmt.Sprintf(
		"SELECT ptype, points FROM %s_%ds WHERE uid=? AND etime >= ? AND etime <= ?",
		t_name, resolution,
	)

	// times need to be in Nanos, but comming as a epoch
	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nano_end := end * nano
	nano_start := start * nano

	vals := []interface{}{
		metric.UniqueId,
		nano_start,
		nano_end,
	}

	rows, err := my.conn.Query(Q, vals...)
	//my.log.Debug("Q: %s, %v", Q, vals)

	if err != nil {
		my.log.Error("Mysql Driver: Metric select failed, %v", err)
		return rawd, err
	}
	defer rows.Close()

	// for each "series" we get make a list of points
	u_start := uint32(start)
	u_end := uint32(end)
	rawd.Start = u_start
	rawd.End = u_end
	rawd.Id = metric.UniqueId
	rawd.Metric = metric.Path

	rawd.AggFunc = repr.GuessReprValueFromKey(metric.Id)

	for rows.Next() {
		var p_type uint8
		var p_bytes []byte
		if err := rows.Scan(&p_type, &p_bytes); err != nil {
			return rawd, err
		}
		s_name := series.NameFromId(p_type)
		s_iter, err := series.NewIter(s_name, p_bytes)
		if err != nil {
			return rawd, err
		}

		for s_iter.Next() {
			to, mi, mx, ls, su, ct := s_iter.Values()

			t := uint32(time.Unix(0, to).Unix())

			// skip if not in range
			if t > u_end || t < u_start {
				continue
			}

			rawd.Data = append(rawd.Data, RawDataPoint{
				Count: ct,
				Sum:   su,
				Max:   mx,
				Min:   mi,
				Last:  ls,
				Time:  t,
			})

			if rawd.RealEnd < t {
				rawd.RealEnd = t
			}
			if rawd.RealStart > t || rawd.RealStart == 0 {
				rawd.RealStart = t
			}
		}
		if s_iter.Error() != nil {
			return rawd, s_iter.Error()
		}
	}
	if err := rows.Err(); err != nil {
		return rawd, err
	}

	return rawd, nil

}

func (my *MySQLMetrics) GetFromWriteCache(metric *indexer.MetricFindItem, start uint32, end uint32, resolution uint32) (*RawRenderItem, error) {

	// grab data from the write inflight cache
	// need to pick the "proper" cache

	// if we are "triggered" rollups then there is only the lowest res cache,
	// but we can get that data too and resample to proper resolution
	cache_db := fmt.Sprintf("%s:%d", my.cacherPrefix, resolution)
	if my.rollupType == "triggered" {
		cache_db = fmt.Sprintf("%s:%d", my.cacherPrefix, my.resolutions[0][0])
	}

	use_cache := GetCacherByName(cache_db)
	if use_cache == nil {
		use_cache = my.cacher
	}
	inflight, err := use_cache.GetAsRawRenderItem(metric.StatName())
	if err != nil {
		return nil, err
	}
	if inflight == nil {
		return nil, nil
	}

	// resample to the res we want
	if my.rollupType == "triggered" && uint32(my.currentResolution) != resolution {
		inflight.Resample(resolution)
	}

	inflight.Metric = metric.Path
	inflight.Id = metric.UniqueId
	inflight.Step = resolution
	inflight.Start = start
	inflight.End = end
	inflight.Tags = metric.Tags
	inflight.MetaTags = metric.MetaTags
	return inflight, nil
}

func (my *MySQLMetrics) RawDataRenderOne(metric *indexer.MetricFindItem, start int64, end int64) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.mysql.renderraw.get-time-ns", time.Now())
	rawd := new(RawRenderItem)

	resolution := my.getResolution(start, end)

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	u_start := uint32(start)
	u_end := uint32(end)

	rawd.Step = resolution
	rawd.Metric = metric.Path
	rawd.Id = metric.UniqueId
	rawd.RealEnd = u_end
	rawd.RealStart = u_start
	rawd.Start = rawd.RealStart
	rawd.End = rawd.RealEnd
	rawd.AggFunc = repr.GuessReprValueFromKey(metric.Id)

	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return rawd, errNotADataNode
	}

	b_len := (u_end - u_start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, errTimeTooSmall
	}

	//cache check
	// the read cache should have "all" the points from a "start" to "end" if the read cache has been activated for
	// a while.  If not, then it's a partial list (basically the read cache just started)
	inflight, err := my.GetFromWriteCache(metric, u_start, u_end, resolution)

	// need at LEAST 2 points to get the proper step size
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// all the data we need is in the inflight
		in_range := inflight.DataInRange(u_start, u_end)
		// if all the data is in this list we don't need to go any further
		if in_range {
			// move the times to the "requested" ones and quantize the list
			inflight.RealEnd = u_end
			inflight.RealStart = u_start
			err = inflight.Quantize()
			return inflight, err
		}
	}
	if err != nil {
		my.log.Error("Mysql: Erroring getting inflight data: %v", err)
	}

	// and now for the mysql Query otherwise
	mysql_data, err := my.GetFromDatabase(metric, resolution, start, end)
	if err != nil {
		my.log.Error("Mysql: Error getting from DB: %v", err)
		return rawd, err
	}

	mysql_data.Step = resolution
	mysql_data.Start = u_start
	mysql_data.End = u_end

	//mysql_data.PrintPoints()
	if inflight == nil {
		mysql_data.Quantize()
		return mysql_data, nil
	}

	if len(mysql_data.Data) > 0 && len(inflight.Data) > 1 {
		//inflight.Quantize()
		//mysql_data.Quantize()
		inflight.Merge(mysql_data)
		/*oo_data := inflight.Data[0:]
		for idx, d := range oo_data {
			fmt.Printf("%d: %d: %f | %f | %f\n", idx, d.Time, d.Sum, mysql_data.Data[idx].Sum, inflight.Data[idx].Sum)
		}
		*/
		return inflight, nil
	}
	inflight.Quantize()
	return inflight, nil
}

// after the "raw" render we need to yank just the "point" we need from the data which
// will make the read-cache much smaller (will compress just the Mean value as the count is 1)
func (my *MySQLMetrics) RawRenderOne(metric indexer.MetricFindItem, from int64, to int64) (*RawRenderItem, error) {
	return my.RawDataRenderOne(&metric, from, to)
}

func (my *MySQLMetrics) Render(path string, from int64, to int64) (WhisperRenderItem, error) {
	raw_data, err := my.RawRender(path, from, to)

	if err != nil {
		return WhisperRenderItem{}, err
	}

	var whis WhisperRenderItem
	whis.Series = make(map[string][]DataPoint)
	for _, data := range raw_data {
		whis.End = data.End
		whis.Start = data.Start
		whis.Step = data.Step
		whis.RealEnd = data.RealEnd
		whis.RealStart = data.RealStart

		d_points := make([]DataPoint, 0)
		for _, d := range data.Data {
			v := d.AggValue(data.AggFunc)
			d_points = append(d_points, DataPoint{Time: d.Time, Value: &v})
		}
		whis.Series[data.Metric] = d_points
	}
	return whis, err
}

func (my *MySQLMetrics) RawRender(path string, from int64, to int64) ([]*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.mysql.rawrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	for _, pth := range paths {
		mets, err := my.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*RawRenderItem, len(metrics), len(metrics))

	// ye old fan out technique
	render_one := func(metric indexer.MetricFindItem, idx int) {
		defer render_wg.Done()
		timeout := time.NewTimer(my.renderTimeout)
		for {
			select {
			case <-timeout.C:
				my.log.Error("Render Timeout for %s (%d->%d)", path, from, to)
				timeout.Stop()
				return
			default:
				_ri, err := my.RawRenderOne(metric, from, to)

				if err != nil {
					my.log.Error("Read Error for %s (%d->%d) : %v", path, from, to, err)
					return
				}
				rawd[idx] = _ri
				return
			}
		}

	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(metric, idx)
	}
	render_wg.Wait()
	return rawd, nil
}

func (my *MySQLMetrics) CacheRender(path string, start int64, end int64, tags repr.SortingTags) (rawd []*RawRenderItem, err error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.cacherender.get-time-ns", time.Now())

	//figure out the best res
	resolution := my.getResolution(start, end)

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	for _, pth := range paths {
		mets, err := my.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd = make([]*RawRenderItem, len(metrics), len(metrics))

	// ye old fan out technique
	render_one := func(metric *indexer.MetricFindItem, idx int) {
		defer render_wg.Done()
		_ri, err := my.GetFromWriteCache(metric, uint32(start), uint32(end), resolution)

		if err != nil {
			my.log.Error("Read Error for %s (%d->%d) : %v", path, start, end, err)
			return
		}
		rawd[idx] = _ri
		return

	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(&metric, idx)
	}
	render_wg.Wait()
	return rawd, nil
}

func (my *MySQLMetrics) CachedSeries(path string, start int64, end int64, tags repr.SortingTags) (series *TotalTimeSeries, err error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.seriesrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	if len(paths) > 1 {
		return series, errMultiTargetsNotAllowed
	}

	metric := &repr.StatName{Key: path}
	metric.MergeMetric2Tags(tags)
	metric.MergeMetric2Tags(my.static_tags)

	resolution := my.getResolution(start, end)
	cache_db := fmt.Sprintf("%s:%v", my.cacherPrefix, resolution)
	use_cache := GetCacherByName(cache_db)
	if use_cache == nil {
		use_cache = my.cacher
	}
	name, inflight, err := use_cache.GetSeries(metric)
	if err != nil {
		return nil, err
	}
	if inflight == nil {
		// try the the path as unique ID
		gots_int := metric.StringToUniqueId(path)
		if gots_int != 0 {
			name, inflight, err = use_cache.GetSeriesById(gots_int)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, nil
		}
	}

	return &TotalTimeSeries{Name: name, Series: inflight}, nil
}

/*************** Match the DBMetrics Interface ***********************/

// given a name get the latest metric series
func (my *MySQLMetrics) GetLatestFromDB(name *repr.StatName, resolution uint32) (DBSeriesList, error) {
	t_name := my.db.RootMetricsTableName()
	Q := fmt.Sprintf(
		"SELECT id, uid, stime, etime, ptype, points FROM %s_%ds WHERE uid=? ORDER BY etime DESC LIMIT 1",
		t_name, resolution,
	)
	rows, err := my.conn.Query(Q, name.UniqueIdString())

	if err != nil {
		my.log.Error("Mysql Driver: Metric select failed, %v", err)
		return nil, err
	}
	defer rows.Close()

	rawd := make(DBSeriesList, 0)
	var p_type uint8
	var p_bytes []byte
	var uid string
	var id, start, end int64
	for rows.Next() {
		if err := rows.Scan(&id, &uid, &start, &end, &p_type, &p_bytes); err != nil {
			return nil, err
		}
		dataums := &DBSeries{
			Id:         id,
			Uid:        uid,
			Start:      start,
			End:        end,
			Ptype:      p_type,
			Pbytes:     p_bytes,
			Resolution: resolution,
		}
		rawd = append(rawd, dataums)
	}
	if err := rows.Err(); err != nil {
		return rawd, err
	}

	return rawd, nil
}

// given a name get the latest metric series
func (my *MySQLMetrics) GetRangeFromDB(name *repr.StatName, start uint32, end uint32, resolution uint32) (DBSeriesList, error) {
	t_name := my.db.RootMetricsTableName()
	Q := fmt.Sprintf(
		"SELECT id, uid, stime, etime, ptype, points FROM %s_%ds WHERE uid=? AND etime >= ? AND etime <= ?",
		t_name, resolution,
	)

	// need to convert second time to nan time
	nano := int64(time.Second)
	nano_end := int64(end) * nano
	nano_start := int64(start) * nano

	rows, err := my.conn.Query(Q, name.UniqueIdString(), nano_start, nano_end)
	//my.log.Debug("Q: %s, %s %d %d", Q, name.UniqueIdString(), nano_start, nano_end)

	if err != nil {
		my.log.Error("Mysql Driver: Metric select failed, %v", err)
		return nil, err
	}
	defer rows.Close()

	rawd := make(DBSeriesList, 0)
	var p_type uint8
	var p_bytes []byte
	var uid string
	var id, t_start, t_end int64
	for rows.Next() {
		if err := rows.Scan(&id, &uid, &t_start, &t_end, &p_type, &p_bytes); err != nil {
			return nil, err
		}
		dataums := &DBSeries{
			Id:         id,
			Uid:        uid,
			Start:      t_start,
			End:        t_end,
			Ptype:      p_type,
			Pbytes:     p_bytes,
			Resolution: resolution,
		}
		rawd = append(rawd, dataums)
	}
	if err := rows.Err(); err != nil {
		return rawd, err
	}

	return rawd, nil
}

// update the row defined in dbs w/ the new bytes from the new Timeseries
func (my *MySQLMetrics) UpdateDBSeries(dbs *DBSeries, ts series.TimeSeries) error {

	t_name := my.db.RootMetricsTableName()
	Q := fmt.Sprintf(
		"UPDATE %s_%ds SET stime=?, etime=?, ptype=?, points=? WHERE uid=? AND id=?",
		t_name, dbs.Resolution,
	)

	ptype := series.IdFromName(ts.Name())
	_, err := my.conn.Exec(Q, ts.StartTime(), ts.LastTime(), ptype, ts.Bytes(), dbs.Uid, dbs.Id)
	return err
}

// update the row defined in dbs w/ the new bytes from the new Timeseries
func (my *MySQLMetrics) InsertDBSeries(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (added int, err error) {

	if name == nil {
		return 0, errNameIsNil
	}
	if timeseries == nil {
		return 0, errSeriesIsNil
	}

	defer func() {
		if r := recover(); r != nil {
			my.log.Critical("Mysql Failure (panic) %v ::", r)
		}
	}()

	t_name := my.db.RootMetricsTableName()
	Q := fmt.Sprintf(
		"INSERT INTO %s_%ds (uid, path, ptype, points, stime, etime) VALUES ",
		t_name, resolution,
	)

	vals := []interface{}{}

	pts := timeseries.Bytes()
	s_time := timeseries.StartTime()
	e_time := timeseries.LastTime()

	Q += "(?,?,?,?,?,?)"
	vals = append(
		vals,
		name.UniqueIdString(),
		name.Key,
		series.IdFromName(timeseries.Name()),
		pts,
		s_time,
		e_time,
	)

	//format all vals at once
	_, err = my.conn.Exec(Q, vals...)
	if err != nil {
		my.log.Error("Mysql Driver: Metric insert failed, %v", err)
		return 0, err
	}

	return 1, nil
}
