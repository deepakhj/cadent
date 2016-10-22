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
	THe MySQL stat write for "indexer" blobs of time series

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


    CREATE TABLE `{tag_table}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
      `name` varchar(255) NOT NULL,
      `value` varchar(255) NOT NULL,
      `is_meta` tinyint(1) NOT NULL DEFAULT 0,
      PRIMARY KEY (`id`),
      KEY `name` (`name`),
      UNIQUE KEY `uid` (`value`, `name`, `is_meta`)
    );

    CREATE TABLE `{tag_table}_xref` (
      `tag_id` BIGINT unsigned,
      `uid` varchar(50) NOT NULL,
      PRIMARY KEY (`tag_id`, `uid`)
    );



*/

package indexer

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/utils"
	"database/sql"
)

var MYSQL_INDEXER_TABLES = []string{`
CREATE TABLE IF NOT EXISTS %s (
segment varchar(255) NOT NULL DEFAULT '',
pos int NOT NULL,
PRIMARY KEY (pos, segment)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
`,
	`
CREATE TABLE IF NOT EXISTS %s (
id BIGINT unsigned NOT NULL AUTO_INCREMENT,
segment varchar(255) NOT NULL,
pos int NOT NULL,
uid varchar(50) NOT NULL,
path varchar(255) NOT NULL DEFAULT '',
length int NOT NULL,
has_data bool NOT NULL DEFAULT 0,
PRIMARY KEY (id),
UNIQUE KEY segment_path (segment,pos,path,has_data),
KEY uid (uid),
KEY length (length)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;`,

	`
CREATE TABLE IF NOT EXISTS %s (
id BIGINT unsigned NOT NULL AUTO_INCREMENT,
name varchar(255) NOT NULL,
value varchar(255) NOT NULL,
is_meta tinyint(1) NOT NULL DEFAULT 0,
UNIQUE KEY name_value (value, name, is_meta),
KEY name (name),
PRIMARY KEY id (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
`,
	`
CREATE TABLE IF NOT EXISTS %s_xref (
tag_id BIGINT unsigned NOT NULL,
uid varchar(50) NOT NULL,
PRIMARY KEY (tag_id, uid),
KEY uid (uid)
) ENGINE=InnoDB DEFAULT CHARSET=ascii ROW_FORMAT=COMPRESSED;`,
}

type mySQLIndexSchema struct {
	conn         *sql.DB
	segmentTable string
	pathTable    string
	tagTable     string
	log          *logging.Logger
	startstop    utils.StartStop
}

func NewMySQLIndexSchema(conn *sql.DB, segment string, path string, tag string) *mySQLIndexSchema {
	my := new(mySQLIndexSchema)
	my.conn = conn
	my.segmentTable = segment
	my.pathTable = path
	my.tagTable = tag
	my.log = logging.MustGetLogger("writers.mysql.index.schema")
	return my
}

func (my *mySQLIndexSchema) AddIndexTables() error {
	var err error
	my.startstop.Start(func() {
		my.log.Notice("Adding mysql index tables `%s` `%s` `%s`", my.segmentTable, my.pathTable, my.tagTable)
		tables := []string{my.segmentTable, my.pathTable, my.tagTable, my.tagTable}
		for idx, q := range MYSQL_INDEXER_TABLES {
			Q := fmt.Sprintf(q, tables[idx])
			_, err = my.conn.Exec(Q)
			if err != nil {
				my.log.Errorf("Mysql Schema Driver: Indexer insert failed, %v", err)
				break
			}
		}

	})
	return err
}
