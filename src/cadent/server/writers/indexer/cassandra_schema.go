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
	Cassandra Index schema injector
*/

package indexer

import (
	"bytes"
	"cadent/server/utils"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"text/template"
)

const CASSANDRA_PATH_TEMPLATE = `
CREATE TYPE IF NOT EXISTS  {{.Keyspace}}.metric_path (
            path text,
            resolution int
        );
==SPLIT==
        CREATE TYPE IF NOT EXISTS  {{.Keyspace}}.metric_id_res (
            id varchar,
            res int
        );
==SPLIT==
        CREATE TYPE IF NOT EXISTS  {{.Keyspace}}.segment_pos (
            pos int,
            segment text
        );
==SPLIT==
 CREATE TABLE  IF NOT EXISTS  {{.Keyspace}}.{{.PathTable}} (
        segment frozen<segment_pos>,
        length int,
        path varchar,
        id varchar,
        has_data boolean,
        PRIMARY KEY (segment, length, path, id)
    ) WITH
         bloom_filter_fp_chance = 0.01
        AND compaction = {'class': 'org.apache.cassandra.db.compaction.LeveledCompactionStrategy'}
        AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};
==SPLIT==
    CREATE INDEX ON metric.path (id);
==SPLIT==
    CREATE TABLE IF NOT EXISTS  {{.Keyspace}}.{{.SegmentTable}} (
        pos int,
        segment text,
        PRIMARY KEY (pos, segment)
    ) WITH COMPACT STORAGE
        AND CLUSTERING ORDER BY (segment ASC)
        AND bloom_filter_fp_chance = 0.01
        AND compaction = {'class': 'org.apache.cassandra.db.compaction.LeveledCompactionStrategy'}
        AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};`

/****************** Interfaces *********************/
type cassandraIndexerSchema struct {
	conn         *gocql.Session
	Keyspace     string
	PathTable    string
	SegmentTable string
	log          *logging.Logger
	startstop    utils.StartStop
}

func NewCassandraIndexerSchema(conn *gocql.Session, keyspace string, ptable string, stable string) *cassandraIndexerSchema {
	cass := new(cassandraIndexerSchema)
	cass.conn = conn
	cass.Keyspace = keyspace
	cass.PathTable = ptable
	cass.SegmentTable = stable
	cass.log = logging.MustGetLogger("writers.cassandara.indexer.schema")
	return cass
}

func (cass *cassandraIndexerSchema) AddIndexerTables() (err error) {
	cass.startstop.Start(func() {

		buf := bytes.NewBuffer(nil)
		cass.log.Notice("Cassandra Index Schema: verifing schema")
		tpl := template.Must(template.New("cassindexer").Parse(CASSANDRA_PATH_TEMPLATE))
		err := tpl.Execute(buf, cass)

		if err != nil {
			err = fmt.Errorf("Cassandra Schema Driver: Index failed, %v", err)
			return
		}

		Q := string(buf.Bytes())
		for _, q := range strings.Split(Q, "==SPLIT==") {
			terr := cass.conn.Query(q).Exec()
			if terr != nil {
				err = fmt.Errorf("Cassandra Schema Driver: Index failed, %v", err)
			}
		}
	})
	return err
}
