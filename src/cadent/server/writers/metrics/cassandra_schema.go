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
	Cassandra Metric schema injector
*/

package metrics

import (
	logging "gopkg.in/op/go-logging.v1"

	"bytes"
	"fmt"
	"github.com/gocql/gocql"
	"strings"
	"text/template"
)

const CASSANDRA_METRICS_FLAT_TEMPLATE = `

CREATE TYPE IF NOT EXISTS {{.Keyspace}}.metric_point (
	max double,
	min double,
	sum double,
	last double,
	count int
);

==SPLIT==

CREATE TYPE IF NOT EXISTS  {{.Keyspace}}.metric_id_res (
            id ascii,
            res int
        );
==SPLIT==

        CREATE TABLE {{.Keyspace}}.{{.Metrictable}} (
            mid frozen<metric_id_res>,
            time bigint,
            point frozen<metric_point>,
            PRIMARY KEY (mid, time)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (time ASC)
            AND {{ if .CassVersionTwo }}
            compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '1',
                'base_time_seconds': '50'
            }
            {{ end }}
            {{ if .CassVersionThree }}
            compaction = {
                'class': 'TimeWindowCompactionStrategy',
                'compaction_window_unit': 'DAYS',
                'compaction_window_size': '1'
            }
            {{ end }}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};`

const CASSANDRA_METRICS_FLAT_PER_RES_TEMPLATE = `

CREATE TYPE IF NOT EXISTS {{.Keyspace}}.metric_point (
	max double,
	min double,
	sum double,
	last double,
	count int
);
==SPLIT==
        CREATE TABLE {{.Keyspace}}.{{.Metrictable}}_{{.Resolution}}s (
            id ascii,
            time bigint,
            point frozen<metric_point>,
            PRIMARY KEY (mid, time)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (time ASC)
            AND {{ if .CassVersionTwo }}
            compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '1',
                'base_time_seconds': '50'
            }
            {{ end }}
            {{ if .CassVersionThree }}
            compaction = {
                'class': 'TimeWindowCompactionStrategy',
                'compaction_window_unit': 'DAYS',
                'compaction_window_size': '1'
            }
            {{ end }}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};`

const CASSANDRA_METRICS_BLOB_TEMPLATE = `
CREATE TYPE IF NOT EXISTS  {{.Keyspace}}.metric_id_res (
            id ascii,
            res int
        );
==SPLIT==
CREATE TABLE IF NOT EXISTS {{.Keyspace}}.{{.MetricsTable}} (
            mid frozen<metric_id_res>,
            etime bigint,
            stime bigint,
            ptype int,
            points blob,
            PRIMARY KEY (mid, etime, stime, ptype)
        ) WITH COMPACT STORAGE AND CLUSTERING ORDER BY (etime ASC)
            AND {{ if .CassVersionTwo }}
            compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '1',
                'base_time_seconds': '50'
            }
            {{ end }}
            {{ if .CassVersionThree }}
            compaction = {
                'class': 'TimeWindowCompactionStrategy',
                'compaction_window_unit': 'DAYS',
                'compaction_window_size': '1'
            }
            {{ end }}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};`

const CASSANDRA_METRICS_BLOB_PER_RES_TEMPLATE = `
CREATE TABLE IF NOT EXISTS {{.Keyspace}}.{{.MetricsTable}}_{{.Resolution}}s (
            id ascii,
            etime bigint,
            stime bigint,
            ptype int,
            points blob,
            PRIMARY KEY (id, etime, stime, ptype)
        ) WITH COMPACT STORAGE AND CLUSTERING ORDER BY (etime ASC)
            AND {{ if .CassVersionTwo }}
            compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '1',
                'base_time_seconds': '50'
            }
            {{ end }}
            {{ if .CassVersionThree }}
            compaction = {
                'class': 'TimeWindowCompactionStrategy',
                'compaction_window_unit': 'DAYS',
                'compaction_window_size': '1'
            }
            {{ end }}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};`

const CASSANDRA_METRICS_LOG_TEMPLATE = `
CREATE TABLE IF NOT EXISTS {{.Keyspace}}.{{.LogTable}}_{{.WriteIndex}}_{{.Resolution}}s (
            seq bigint,
            ts bigint,
            pts blob,
            PRIMARY KEY (seq, ts)
        ) WITH COMPACT STORAGE AND CLUSTERING ORDER BY (ts ASC)
            AND {{ if .CassVersionTwo }}
            compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '1',
                'base_time_seconds': '50'
            }
            {{ end }}
            {{ if .CassVersionThree }}
            compaction = {
                'class': 'TimeWindowCompactionStrategy',
                'compaction_window_unit': 'DAYS',
                'compaction_window_size': '1'
            }
            {{ end }}
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};`

/****************** Interfaces *********************/
type cassandraMetricsSchema struct {
	conn             *gocql.Session
	Keyspace         string
	MetricsTable     string
	Resolutions      [][]int
	Resolution       int
	LogTable         string
	WriteIndex       int
	Mode             string
	Perres           bool
	CassVersionTwo   bool
	CassVersionThree bool
	log              *logging.Logger
}

func NewCassandraMetricsSchema(
	conn *gocql.Session,
	keyscpace string,
	mtable string,
	resolutions [][]int,
	mode string,
	perres bool,
	version string,
) *cassandraMetricsSchema {
	cass := new(cassandraMetricsSchema)
	cass.conn = conn
	cass.Keyspace = keyscpace
	cass.MetricsTable = mtable
	cass.Resolutions = resolutions
	cass.Mode = mode
	cass.Perres = perres
	cass.CassVersionTwo = true
	cass.CassVersionThree = false

	if len(version) > 0 && strings.Split(version, ".")[0] == "3" {
		cass.CassVersionTwo = false
		cass.CassVersionThree = true
	}
	cass.log = logging.MustGetLogger("writers.cassandara.metric.schema")
	return cass
}

func (cass *cassandraMetricsSchema) AddMetricsTable() error {
	var err error

	if len(cass.Resolutions) == 0 {
		err = fmt.Errorf("Need resolutions")
		return err
	}

	baseTpl := CASSANDRA_METRICS_BLOB_TEMPLATE
	if cass.Perres {
		baseTpl = CASSANDRA_METRICS_BLOB_PER_RES_TEMPLATE
	}

	if cass.Mode == "flat" {
		baseTpl = CASSANDRA_METRICS_FLAT_TEMPLATE
		if cass.Perres {
			baseTpl = CASSANDRA_METRICS_FLAT_PER_RES_TEMPLATE

		}
	}

	if cass.Perres {
		for _, r := range cass.Resolutions {
			cass.Resolution = r[0]

			buf := bytes.NewBuffer(nil)
			cass.log.Notice("Cassandra Schema Driver: verifing schema")
			tpl := template.Must(template.New("cassmetric").Parse(baseTpl))
			err = tpl.Execute(buf, cass)
			if err != nil {
				cass.log.Errorf("%s", err)
				err = fmt.Errorf("Cassandra Schema Driver: Metric failed, %v", err)
				return err
			}
			Q := string(buf.Bytes())

			for _, q := range strings.Split(Q, "==SPLIT==") {

				err = cass.conn.Query(q).Exec()
				if err != nil {
					cass.log.Errorf("%s", err)
					err = fmt.Errorf("Cassandra Schema Driver: Metric failed, %v", err)
					return err
				}
				cass.log.Notice("Added table for resolution %s_%ds", cass.MetricsTable, r[0])
			}

		}

	} else {

		buf := bytes.NewBuffer(nil)
		cass.log.Notice("Cassandra Schema Driver: verifing schema")
		tpl := template.Must(template.New("cassmetric").Parse(baseTpl))
		err = tpl.Execute(buf, cass)

		if err != nil {
			cass.log.Errorf("%s", err)
			err = fmt.Errorf("Cassandra Schema Driver: Metric failed, %v", err)
			return err
		}
		cass.log.Notice("Added table for all resolutions %s", cass.MetricsTable)

		Q := string(buf.Bytes())
		for _, q := range strings.Split(Q, "==SPLIT==") {

			err = cass.conn.Query(q).Exec()
			if err != nil {
				cass.log.Errorf("%s", err)
				err = fmt.Errorf("Cassandra Schema Driver: Metric failed, %v", err)
				return err
			}
		}
	}
	return err
}

func (cass *cassandraMetricsSchema) AddMetricsLogTable() error {
	var err error

	buf := bytes.NewBuffer(nil)
	cass.log.Notice("Cassandra Schema Driver: verifing log schema")
	tpl := template.Must(template.New("cassmetriclog").Parse(CASSANDRA_METRICS_LOG_TEMPLATE))
	err = tpl.Execute(buf, cass)

	if err != nil {
		cass.log.Errorf("%s", err)
		err = fmt.Errorf("Cassandra Schema Driver: Metric Log failed, %v", err)
		return err
	}
	cass.log.Notice("Adding log table %s_%d_%ds", cass.LogTable, cass.WriteIndex, cass.Resolution)

	Q := string(buf.Bytes())
	for _, q := range strings.Split(Q, "==SPLIT==") {

		err = cass.conn.Query(q).Exec()
		if err != nil {
			cass.log.Errorf("%s", err)
			err = fmt.Errorf("Cassandra Schema Driver: Metric Log failed, %v", err)
			return err
		}
	}

	return err
}
