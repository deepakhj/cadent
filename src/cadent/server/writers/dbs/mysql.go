/*
	THe MySQL write

	The table should have this schema to match the repr item

// this is for easy index searches on paths


CREATE TABLE `{segment_table}` (
  `segment` varchar(255) NOT NULL DEFAULT '',
  `pos` int NOT NULL,
  PRIMARY KEY (`pos`, `segment`)
);

CREATE TABLE `{path_table}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
  `segment` varchar(255) NOT NULL,
  `pos` int NOT NULL,
  `uid` varchar(50) NOT NULL,
  `path` varchar(255) NOT NULL DEFAULT '',
  `length` int NOT NULL,
  `has_data` bool DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `seg_pos` (`segment`, `pos`),
  KEY `uid` (`uid`),
  KEY `length` (`length`)
);


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
	"time"
)

/****************** Interfaces *********************/
type MySQLDB struct {
	conn           *sql.DB
	table          string
	path_table     string
	segment_table  string
	table_prefix   string
	tag_table      string
	tag_table_xref string

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

	_stable := conf["segment_table"]
	if _stable == nil {
		my.segment_table = "metric_segment"
	} else {
		my.segment_table = _stable.(string)
	}

	_tagtable := conf["tag_table"]
	if _tagtable == nil {
		my.tag_table = "metric_tag"
	} else {
		my.tag_table = _tagtable.(string)
	}

	_tagtablex := conf["tag_table_xref"]
	if _tagtablex == nil {
		my.tag_table_xref = my.tag_table + "_xref"
	} else {
		my.tag_table_xref = _tagtablex.(string)
	}

	// file prefix
	_pref := conf["prefix"]
	if _pref == nil {
		my.table_prefix = ""
	} else {
		my.table_prefix = _pref.(string)
	}

	// some reasonable defaults
	my.conn.SetMaxOpenConns(50)
	my.conn.SetConnMaxLifetime(time.Duration(5 * time.Minute))

	return nil
}

func (my *MySQLDB) RootMetricsTableName() string {
	return my.table
}

func (my *MySQLDB) Tablename() string {
	return my.table + my.table_prefix
}

func (my *MySQLDB) PathTable() string {
	return my.path_table
}

func (my *MySQLDB) TagTable() string {
	return my.tag_table
}

func (my *MySQLDB) TagTableXref() string {
	return my.tag_table_xref
}

func (my *MySQLDB) SegmentTable() string {
	return my.segment_table
}

func (my *MySQLDB) Connection() DBConn {
	return my.conn
}
