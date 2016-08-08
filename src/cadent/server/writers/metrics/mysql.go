/*
	THe MySQL stat write for "binary" blobs of time series

     CREATE TABLE `{table}{prefix}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `point_type` TINYINT NOT NULL,
      `points` blob,
      `stime` BIGINT unsigned NOT NULL,
      `etime` BIGINT unsigned NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`),
      KEY `path` (`path`),
      KEY `time` (`stime`, `etime`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	OPTIONS: For `Config`

	table: base table name (default: metrics)
	prefix: table prefix if any (_1s, _5m)

	# series (and cache) encoding types
	series_encoding: gorilla

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
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"cadent/server/writers/series"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	MYSQL_DEFAULT_LONGEST_TIME = "3600s"
	MYSQL_DEFAULT_SERIES_TYPE  = "gorilla"
	MYSQL_DEFAULT_SERIES_CHUNK = 16 * 1024 * 1024 // 16kb
)

/****************** Interfaces *********************/
type MySQLMetrics struct {
	db          *dbs.MySQLDB
	conn        *sql.DB
	indexer     indexer.Indexer
	resolutions [][]int

	series_encoding  string
	blob_max_bytes   int
	blob_oldest_time time.Duration

	write_list     []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	max_write_size int             // size of that buffer before a flush
	max_idle       time.Duration   // either max_write_size will trigger a write or this time passing will
	write_lock     sync.Mutex

	cacher            *Cacher
	cacheOverFlowChan chan *TotalTimeSeries // on byte overflow of cacher force a write
	overFlowShutdown  chan bool

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
		return fmt.Errorf("Resulotuion needed for mysql writer")
	}

	// newDB is a "cached" item to let us pass the connections around
	db_key := dsn + fmt.Sprintf("%f", resolution)
	db, err := dbs.NewDB("mysql", db_key, conf)
	if err != nil {
		return err
	}

	my.db = db.(*dbs.MySQLDB)
	my.conn = db.Connection().(*sql.DB)

	_wr_buffer := conf["batch_count"]
	if _wr_buffer == nil {
		my.max_write_size = 1000
	} else {
		// toml things generic ints are int64
		my.max_write_size = int(_wr_buffer.(int64))
	}

	_pr_flush := conf["periodic_flush"]
	my.max_idle = time.Duration(time.Second)
	if _pr_flush != nil {
		dur, err := time.ParseDuration(_pr_flush.(string))
		if err == nil {
			my.max_idle = dur
		} else {
			my.log.Error("Mysql Driver: Invalid Duration `%v`", _pr_flush)
		}
	}
	// cacher and mysql options for series
	my.series_encoding = MYSQL_DEFAULT_SERIES_TYPE
	_bt := conf["series_encoding"]
	if _bt != nil {
		my.series_encoding = _bt.(string)
	}

	//cacher
	my.cacher = NewCacher()
	if err != nil {
		return err
	}
	my.cacher.seriesType = my.series_encoding

	my.blob_max_bytes = MYSQL_DEFAULT_SERIES_CHUNK
	_bz := conf["cache_byte_size"]
	if _bz != nil {
		my.blob_max_bytes = int(_bz.(int64))
	}
	my.cacher.maxBytes = my.blob_max_bytes

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

	my.cacher.maxKeys = CACHER_METRICS_KEYS
	_cz := conf["cache_metric_size"]
	if _cz != nil {
		my.cacher.maxKeys = _cz.(int)
	}

	// unlike the other writers, overflows of cache size are
	// exactly what we want to write
	my.cacher.overFlowMethod = "chan"
	my.cacheOverFlowChan = make(chan *TotalTimeSeries, 128) // a little buffer
	my.overFlowShutdown = make(chan bool)
	my.cacher.SetOverflowChan(my.cacheOverFlowChan)

	my.shutdown = make(chan bool)

	return nil
}

func (my *MySQLMetrics) Start() {
	my.log.Notice("Starting mysql writer for %s at %d bytes per series", my.db.Tablename(), my.blob_max_bytes)
	my.cacher.Start()
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

	if my.overFlowShutdown != nil {
		my.overFlowShutdown <- true
	}

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
		case <-my.overFlowShutdown:
			return
		case statitem := <-my.cacheOverFlowChan:

			// bail
			if my.shutitdown {
				return
			}

			//my.log.Debug("Cache Write byte overflow %s.", statitem.Name.Key)
			my.InsertSeries(statitem.Name, statitem.Series)
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
		"INSERT INTO %s (uid, path, point_type, points, stime, etime) VALUES ",
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
	my.indexer.Write(stat.Name)
	my.cacher.Add(&stat.Name, &stat)
	return nil
}

/**** READER ***/
// XXX TODO
func (my *MySQLMetrics) Render(path string, from string, to string) (WhisperRenderItem, error) {
	return WhisperRenderItem{}, fmt.Errorf("MYSQL READER NOT YET DONE")
}
func (my *MySQLMetrics) RawRender(path string, from string, to string) ([]*RawRenderItem, error) {
	return []*RawRenderItem{}, fmt.Errorf("MYSQL READER NOT YET DONE")
}
