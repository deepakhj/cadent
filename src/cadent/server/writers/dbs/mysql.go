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
	"cadent/server/utils/options"
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

func (my *MySQLDB) Config(conf options.Options) error {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}

	my.conn, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	my.table = conf.String("table", "metrics")
	my.path_table = conf.String("path_table", "metric_path")
	my.segment_table = conf.String("segment_table", "metric_segment")
	my.tag_table = conf.String("tag_table", "metric_tag")
	my.tag_table_xref = conf.String("tag_table_xref", my.tag_table+"_xref")
	my.table_prefix = conf.String("prefix", "")

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
