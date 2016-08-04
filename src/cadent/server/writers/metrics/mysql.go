/*
	THe MySQL stat write

     CREATE TABLE `{table}_{keeperprefix}` (
      `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
      `uid` int NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `sum` float NOT NULL,
      `min` float NOT NULL,
      `max` float NOT NULL,
      `first` float NOT NULL,
      `last` float NOT NULL,
      `count` float NOT NULL,
      `resolution` int(11) NOT NULL,
      `time` datetime(6) NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`),
      KEY `path` (`path`),
      KEY `time` (`time`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	OPTIONS: For `Config`

		table: base table name (default: metrics)
		prefix: table prefix if any (_1s, _5m)
		batch_count: batch this many inserts for much faster insert performance (default 1000)
		periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

/****************** Interfaces *********************/
type MySQLMetrics struct {
	db          *dbs.MySQLDB
	conn        *sql.DB
	indexer     indexer.Indexer
	resolutions [][]int

	write_list     []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	max_write_size int             // size of that buffer before a flush
	max_idle       time.Duration   // either max_write_size will trigger a write or this time passing will
	write_lock     sync.Mutex

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
	db, err := dbs.NewDB("mysql", dsn, conf)
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

	go my.PeriodFlush()

	return nil
}

// TODO
func (my *MySQLMetrics) Stop() {
	return
}

// TODO
func (my *MySQLMetrics) Start() {
	return
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

func (my *MySQLMetrics) PeriodFlush() {
	for {
		time.Sleep(my.max_idle)
		my.Flush()
	}
}

func (my *MySQLMetrics) Flush() (int, error) {
	my.write_lock.Lock()
	defer my.write_lock.Unlock()

	l := len(my.write_list)
	if l == 0 {
		return 0, nil
	}

	Q := fmt.Sprintf(
		"INSERT INTO %s (id, uid, path, sum, min, max, last, count, resolution, time) VALUES ",
		my.db.Tablename(),
	)

	vals := []interface{}{}

	for _, stat := range my.write_list {
		Q += "(?,?,?,?,?,?,?,?,?,?), "
		vals = append(
			vals, stat.UniqueId(), stat.Name.Key, stat.Sum, stat.Min, stat.Max, stat.Last, stat.Count, stat.Name.Resolution, stat.Time,
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

func (my *MySQLMetrics) Write(stat repr.StatRepr) error {

	if len(my.write_list) > my.max_write_size {
		_, err := my.Flush()
		if err != nil {
			return err
		}
	}

	my.indexer.Write(stat.Name) // to the indexer

	// Flush can cause double locking
	my.write_lock.Lock()
	defer my.write_lock.Unlock()
	my.write_list = append(my.write_list, stat)
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
