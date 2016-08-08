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

// for "mysql-flat
CREATE TABLE `{metrics-table}{resolutionprefix}` (
      `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) NULL,
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

// for "mysql"  blob form
CREATE TABLE `{table}{prefix}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `point_type` varchar(20) NOT NULL DEFAULT '',
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
		path_table: base table name (default: metric_path)
		prefix: table prefix if any (_1s, _5m)
		batch_count: batch this many inserts for much faster insert performance (default 1000)
		periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package dbs

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
)

/****************** Interfaces *********************/
type MySQLDB struct {
	conn         *sql.DB
	table        string
	path_table   string
	table_prefix string

	log *logging.Logger
}

func NewMySQLDB() *MySQLDB {
	my := new(MySQLDB)
	my.log = logging.MustGetLogger("writers.mysql")
	return my
}

func (my *MySQLDB) Config(conf map[string]interface{}) error {
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

	// file prefix
	_pref := conf["prefix"]
	if _pref == nil {
		my.table_prefix = ""
	} else {
		my.table_prefix = _pref.(string)
	}

	return nil
}

func (my *MySQLDB) Tablename() string {
	return my.table + my.table_prefix
}

func (my *MySQLDB) PathTable() string {
	return my.path_table
}

func (my *MySQLDB) Connection() DBConn {
	return my.conn
}
