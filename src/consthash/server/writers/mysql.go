/*
	THe MySQL write

	The table should have this schema to match the repr item

// this is for easy index searches on paths

CREATE TABLE `{path_table}` (
  `path` varchar(255) NOT NULL DEFAULT '',
  `length` int NOT NULL,
  PRIMARY KEY (`path`),
  KEY `length` (`length`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `{table}{table_prefix}` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `stat` varchar(255) NOT NULL DEFAULT '',
  `sum` float NOT NULL,
  `mean` float NOT NULL,
  `min` float NOT NULL,
  `max` float NOT NULL,
  `count` float NOT NULL,
  `resolution` int(11) NOT NULL,
  `time` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `stat` (`stat`),
  KEY `time` (`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	OPTIONS: For `Config`

		table: base table name (default: metrics)
		path_table: base table name (default: metric_path)
		prefix: table prefix if any (_1s, _5m)
		batch_count: batch this many inserts for much faster insert performance (default 1000)
		periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package writers

import (
	"consthash/server/repr"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"sync"
	"time"
)

type MyPath struct {
	Path   string
	Length int
}

/****************** Interfaces *********************/
type MySQLWriter struct {
	conn         *sql.DB
	table        string
	path_table   string
	table_prefix string

	write_list     []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	max_write_size int             // size of that buffer before a flush
	max_idle       time.Duration   // either max_write_size will trigger a write or this time passing will
	write_lock     sync.Mutex

	log *logging.Logger
}

func NewMySQLWriter() *MySQLWriter {
	my := new(MySQLWriter)
	my.log = logging.MustGetLogger("writers.mysql")
	return my
}

func (my *MySQLWriter) Config(conf map[string]interface{}) error {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}
	dsn := gots.(string)
	var err error
	my.conn, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	_table := conf["table"]
	if _table == nil {
		my.table = "metrics"
	} else {
		my.table = _table.(string)
	}

	_ptable := conf["path_table"]
	if _ptable == nil {
		my.path_table = "metric_path"
	} else {
		my.path_table = _ptable.(string)
	}

	_wr_buffer := conf["batch_count"]
	if _wr_buffer == nil {
		my.max_write_size = 1000
	} else {
		// toml things generic ints are int64
		my.max_write_size = int(_wr_buffer.(int64))
	}

	// file prefix
	_pref := conf["prefix"]
	if _pref == nil {
		my.table_prefix = ""
	} else {
		my.table_prefix = _pref.(string)
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

func (my *MySQLWriter) Tablename() string {
	return my.table + my.table_prefix
}

func (my *MySQLWriter) PeriodFlush() {
	for {
		time.Sleep(my.max_idle)
		my.Flush()
	}
	return
}

func (my *MySQLWriter) Flush() (int, error) {
	my.write_lock.Lock()
	defer my.write_lock.Unlock()

	l := len(my.write_list)
	if l == 0 {
		return 0, nil
	}

	Q := fmt.Sprintf(
		"INSERT INTO %s (stat, sum, mean, min, max, count, resolution, time) VALUES ",
		my.Tablename(),
	)

	pthQ := fmt.Sprintf(
		"INSERT IGNORE INTO %s (path, length) VALUES ",
		my.path_table,
	)

	pvals := []interface{}{}
	vals := []interface{}{}

	for _, stat := range my.write_list {
		Q += "(?,?,?,?,?,?,?,?), "
		vals = append(
			vals, stat.Key, stat.Sum, stat.Mean, stat.Min, stat.Max, stat.Count, stat.Resolution, stat.Time,
		)

		pthQ += "(?, ?), "
		pvals = append(pvals, stat.Key, len(strings.Split(stat.Key, ".")))

	}

	//trim the last ", "
	Q = Q[0 : len(Q)-2]
	pthQ = pthQ[0 : len(pthQ)-2]

	//prepare the statement
	stmt, err := my.conn.Prepare(Q)
	if err != nil {
		my.log.Error("Mysql Driver: Metric prepare failed, %v", err)
		return 0, err
	}
	defer stmt.Close()

	//prepare the statement
	pstmt, err := my.conn.Prepare(pthQ)
	if err != nil {
		my.log.Error("Mysql Driver: Path prepare failed, %v", err)
		return 0, err
	}
	defer pstmt.Close()

	//format all vals at once
	_, err = stmt.Exec(vals...)
	if err != nil {
		my.log.Error("Mysql Driver: Metric insert failed, %v", err)
		return 0, err
	}

	//format all vals at once
	_, err = pstmt.Exec(pvals...)
	if err != nil {
		my.log.Error("Mysql Driver: Path insert failed, %v", err)
		return 0, err
	}

	my.write_list = nil
	my.write_list = []repr.StatRepr{}
	return l, nil
}

func (my *MySQLWriter) Write(stat repr.StatRepr) error {

	if len(my.write_list) > my.max_write_size {
		_, err := my.Flush()
		if err != nil {
			return err
		}
	}

	// Flush can cause double locking
	my.write_lock.Lock()
	defer my.write_lock.Unlock()
	my.write_list = append(my.write_list, stat)
	return nil
}
