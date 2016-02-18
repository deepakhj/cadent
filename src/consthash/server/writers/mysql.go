/*
	THe MySQL write

	The table should have this schema to match the repr item


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
	logging "github.com/op/go-logging"

	"sync"
	"time"
)

/****************** Interfaces *********************/
type MySQLWriter struct {
	conn         *sql.DB
	table        string
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

	vals := []interface{}{}

	for _, stat := range my.write_list {
		Q += "(?,?,?,?,?,?,?,?), "
		vals = append(
			vals, stat.Key, stat.Sum, stat.Mean, stat.Min, stat.Max, stat.Count, stat.Resolution, stat.Time,
		)
	}

	//trim the last ", "
	Q = Q[0 : len(Q)-2]

	//prepare the statement
	stmt, err := my.conn.Prepare(Q)
	defer stmt.Close()

	if err != nil {
		my.log.Error("Mysql Driver: prepare failed, %v", err)
		return 0, err
	}
	//format all vals at once
	_, err = stmt.Exec(vals...)
	if err != nil {
		my.log.Error("Mysql Driver: insert failed, %v", err)
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
