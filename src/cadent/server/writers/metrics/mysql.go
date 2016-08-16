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



*/

package metrics

import (
	"cadent/server/broadcast"
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
	MYSQL_DEFAULT_LONGEST_TIME = "3600s"
	MYSQL_DEFAULT_SERIES_TYPE  = "gorilla"
	MYSQL_DEFAULT_SERIES_CHUNK = 16 * 1024 // 16kb
	MYSQL_RENDER_TIMEOUT       = "5s"      // 5 second time out on any render
)

// common errors to avoid GC pressure
var errNotADataNode = errors.New("Mysql: Render: Not a data node")
var errTimeTooSmall = errors.New("Mysql: Render: time too narrow")

/****************** Interfaces *********************/
type MySQLMetrics struct {
	db            *dbs.MySQLDB
	conn          *sql.DB
	indexer       indexer.Indexer
	renderTimeout time.Duration
	resolutions   [][]int
	static_tags   repr.SortingTags

	series_encoding  string
	blob_max_bytes   int
	blob_oldest_time time.Duration

	// this is for Render where we may have several caches, but only "one"
	// cacher get picked for the default render (things share the cache from writers
	// and the api render, but not all the caches, so we need to be able to get the
	// caches from other resolutions
	// cache:mysqlblob:[mysql hosts]:[resolution]
	// the cache singleton keys
	cacherPrefix  string
	cacher        *Cacher
	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

	shutitdown        bool      // just a flag
	shutdown          chan bool // just a flag
	writes_per_second int       // allowed writes per second
	num_workers       int
	queue_len         int

	log *logging.Logger
}

func NewMySQLMetrics() *MySQLMetrics {
	my := new(MySQLMetrics)
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

	// cacher and mysql options for series
	my.series_encoding = MYSQL_DEFAULT_SERIES_TYPE
	_bt := conf["series_encoding"]
	if _bt != nil {
		my.series_encoding = _bt.(string)
	}

	//cacher
	my.cacherPrefix = fmt.Sprintf("cache:mysqlblob:%s", dsn)
	cache_key := fmt.Sprintf(my.cacherPrefix, dsn, resolution)
	my.cacher, err = getCacherSingleton(cache_key)
	if err != nil {
		return err
	}

	my.blob_max_bytes = MYSQL_DEFAULT_SERIES_CHUNK
	_bz := conf["cache_byte_size"]
	if _bz != nil {
		my.blob_max_bytes = int(_bz.(int64))
	}

	pd := MYSQL_DEFAULT_LONGEST_TIME
	_pd := conf["cache_longest_time"]
	if _pd != nil {
		pd = _pd.(string)
	}
	dur, err := time.ParseDuration(pd)
	if err != nil {
		return fmt.Errorf("Mysql cache_longest_time is not a valid duration: %v", err)
	}
	my.blob_oldest_time = dur

	g_tag, ok := conf["tags"]
	if ok {
		my.static_tags = repr.SortingTagsFromString(g_tag.(string))
	}

	if err != nil {
		return err
	}
	// only set these if it's not been started/init'ed
	// as the readers will use this object as well
	if !my.cacher.started && !my.cacher.inited {
		my.cacher.inited = true
		my.cacher.maxBytes = my.blob_max_bytes
		my.cacher.seriesType = my.series_encoding

		my.cacher.maxKeys = CACHER_METRICS_KEYS
		_cz := conf["cache_metric_size"]
		if _cz != nil {
			my.cacher.maxKeys = int(_cz.(int64))
		}

		// unlike the other writers, overflows of cache size are
		// exactly what we want to write
		my.cacher.overFlowMethod = "chan"
	}

	my.shutdown = make(chan bool)

	rdur, err := time.ParseDuration(MYSQL_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	my.renderTimeout = rdur

	return nil
}

func (my *MySQLMetrics) Start() {
	my.log.Notice("Starting mysql writer for %s at %d bytes per series", my.db.Tablename(), my.blob_max_bytes)
	my.cacher.maxBytes = my.blob_max_bytes
	my.cacher.Start()

	// only register this when we start as we really want to consume
	my.cacheOverFlow = my.cacher.GetOverFlowChan()
	go my.overFlowWrite()
}

func (my *MySQLMetrics) Stop() {
	shutdown.AddToShutdown()
	defer shutdown.ReleaseFromShutdown()
	my.log.Warning("Starting Shutdown of writer")

	if my.shutitdown {
		return // already did
	}
	my.shutitdown = true

	my.cacher.Stop()

	mets := my.cacher.Queue
	mets_l := len(mets)
	my.log.Warning("Shutting down, exhausting the queue (%d items) and quiting", mets_l)
	// full tilt write out
	did := 0
	for _, queueitem := range mets {
		if did%100 == 0 {
			my.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
		}
		name, series, _ := my.cacher.GetSeriesById(queueitem.metric)
		if series != nil {
			stats.StatsdClient.Incr(fmt.Sprintf("writer.mysql.write.send-to-writers"), 1)
			my.InsertSeries(name, series)
		}
		did++
	}
	my.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
	my.log.Warning("Shutdown finished ... quiting mysql blob writer")

}

// listen to the overflow chan from the cache and attempt to write "now"
func (my *MySQLMetrics) overFlowWrite() {
	for {
		select {
		case statitem, more := <-my.cacheOverFlow.Ch:

			// bail
			if my.shutitdown || !more {
				return
			}

			//my.log.Debug("Cache Write byte overflow %s.", statitem.Name.Key)
			my.InsertSeries(statitem.(*TotalTimeSeries).Name, statitem.(*TotalTimeSeries).Series)
		}
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

func (my *MySQLMetrics) InsertSeries(name *repr.StatName, timeseries series.TimeSeries) (int, error) {

	// i.e metrics_5s
	t_name := my.db.Tablename()

	Q := fmt.Sprintf(
		"INSERT INTO %s (uid, path, ptype, points, stime, etime) VALUES ",
		t_name,
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
		series.IdFromName(my.series_encoding),
		pts,
		s_time,
		e_time,
	)

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

	return 1, nil
}

func (my *MySQLMetrics) Write(stat repr.StatRepr) error {
	stat.Name.MergeMetric2Tags(my.static_tags)
	my.indexer.Write(stat.Name)
	my.cacher.Add(&stat.Name, &stat)
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
			to, mi, mx, fi, ls, su, ct := s_iter.Values()

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
				First: fi,
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

func (my *MySQLMetrics) RawDataRenderOne(metric *indexer.MetricFindItem, from string, to string) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.mysql.renderraw.get-time-ns", time.Now())
	rawd := new(RawRenderItem)

	start, err := ParseTime(from)
	if err != nil {
		my.log.Error("Invalid from time `%s` :: %v", from, err)
		return rawd, err
	}

	end, err := ParseTime(to)
	if err != nil {
		my.log.Error("Invalid from time `%s` :: %v", to, err)
		return rawd, err
	}
	if end < start {
		start, end = end, start
	}

	//figure out the best res
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
	stat_name := metric.StatName()

	/* no need for this
	cached, got_cache := my.GetFromReadCache(stat_name.Key, start, end)

	// we assume the "cache" is hot data (by design) so if the num points is "lacking"
	// we know we need to get to the data store (or at least the inflight) for the rest
	// add the "step" here as typically we have the "forward" step of data
	if got_cache && cached.StartInRange(u_start+cached.Step) {
		stats.StatsdClient.Incr("reader.mysql.render.read.cache.hit", 1)
		my.log.Debug("Read Cache Hit: %s [%d, %d)", metric.Id, start, end)
		// need to set the start end to the ones requested before quantization to create the proper
		// spanning range
		cached.Start = u_start
		cached.End = u_end
		cached.Quantize()
		return cached, nil
	}
	*/

	// grab data from the write inflight cache
	// need to pick the "proper" cache
	cache_db := fmt.Sprintf("%s:%v", my.cacherPrefix, resolution)
	use_cache := getCacherByName(cache_db)
	if use_cache == nil {
		use_cache = my.cacher
	}
	inflight, err := use_cache.GetAsRawRenderItem(stat_name)

	// need at LEAST 2 points to get the proper step size
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// all the data we need is in the inflight
		in_range := inflight.DataInRange(u_start, u_end)
		inflight.Metric = metric.Path
		inflight.Id = metric.UniqueId
		inflight.Step = resolution
		inflight.Start = u_start
		inflight.End = u_end

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
func (my *MySQLMetrics) RawRenderOne(metric indexer.MetricFindItem, from string, to string) (*RawRenderItem, error) {
	return my.RawDataRenderOne(&metric, from, to)
}

func (my *MySQLMetrics) Render(path string, from string, to string) (WhisperRenderItem, error) {
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

func (my *MySQLMetrics) RawRender(path string, from string, to string) ([]*RawRenderItem, error) {
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
				my.log.Error("Render Timeout for %s (%s->%s)", path, from, to)
				timeout.Stop()
				return
			default:
				_ri, err := my.RawRenderOne(metric, from, to)

				if err != nil {
					my.log.Error("Read Error for %s (%s->%s) : %v", path, from, to, err)
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
