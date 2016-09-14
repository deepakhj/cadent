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
      KEY `uid` (`uid`, `etime`),
      KEY `start_end` (`etime`, `stime`),
      KEY `path` (`path`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	Prefixes are `_{resolution}s` (i.e. "_" + (uint32 resolution) + "s")


*/

package metrics

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/utils"
	"database/sql"
)

const MYSQL_METRICS_BLOB_TABLE = `
CREATE TABLE IF NOT EXISTS %s  (
	id BIGINT unsigned NOT NULL AUTO_INCREMENT,
       	uid varchar(50) CHARACTER SET ascii NOT NULL,
      	path varchar(255) NOT NULL DEFAULT '',
      	ptype TINYINT NOT NULL,
      	points blob NOT NULL,
      	stime BIGINT unsigned NOT NULL,
      	etime BIGINT unsigned NOT NULL,
      	PRIMARY KEY (id),
      	KEY start_end (etime, stime),
      	KEY uid_etime (uid, etime)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED`

const MYSQL_METRICS_FLAT_TABLE = `
CREATE TABLE IF NOT EXISTS %s  (
	id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
          uid varchar(50) CHARACTER SET ascii NOT NULL,
          path varchar(255) NOT NULL DEFAULT '',
          sum float NULL,
          min float NULL,
          max float NULL,
          last float NULL,
          count float NULL,
          time datetime(6) NOT NULL,
          PRIMARY KEY (id),
          KEY uid (uid),
          KEY time (time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED`

/****************** Interfaces *********************/
type MySQLMetricsSchema struct {
	conn        *sql.DB
	tableBase   string
	resolutions [][]int
	mode        string
	log         *logging.Logger
	startstop   utils.StartStop
}

func NewMySQLMetricsSchema(conn *sql.DB, metric_table string, resolutions [][]int, mode string) *MySQLMetricsSchema {
	my := new(MySQLMetricsSchema)
	my.conn = conn
	my.tableBase = metric_table
	my.resolutions = resolutions
	my.mode = mode
	my.log = logging.MustGetLogger("writers.mysql.schema")
	return my
}

func (my *MySQLMetricsSchema) AddMetricsTable() error {
	var err error
	my.startstop.Start(func() {
		// resolutions are [resolution][ttl]
		for _, res := range my.resolutions {
			tname := fmt.Sprintf("%s_%ds", my.tableBase, res[0])
			my.log.Notice("Adding mysql metric tables `%s`", tname)
			var Q string
			if my.mode == "flat" {
				Q = fmt.Sprintf(MYSQL_METRICS_FLAT_TABLE, tname)
			} else {
				Q = fmt.Sprintf(MYSQL_METRICS_BLOB_TABLE, tname)

			}
			_, err = my.conn.Exec(Q)
			if err != nil {
				my.log.Errorf("Mysql Schema Driver: Metric insert failed, %v", err)
			}
		}
	})
	return err
}
