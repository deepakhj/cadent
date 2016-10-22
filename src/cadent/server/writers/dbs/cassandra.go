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
	just a little abstraction around the DBs connections
	and various config options we need
*/

package dbs

import (
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/utils/options"
	"strings"
	"sync"
	"time"
)

const CASSANDRA_DEFAULT_CONNECTIONS int64 = 10

// the singleton as we really ONLY want one connection per DSN
var _SESSION_SINGLETON map[string]*gocql.Session
var _session_mutex sync.Mutex

func getSessionSingleton(nm string, cluster *gocql.ClusterConfig) (*gocql.Session, error) {
	_session_mutex.Lock()
	defer _session_mutex.Unlock()

	if val, ok := _SESSION_SINGLETON[nm]; ok {
		return val, nil
	}
	sess, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	_SESSION_SINGLETON[nm] = sess
	return sess, nil
}

// special onload init
func init() {
	_SESSION_SINGLETON = make(map[string]*gocql.Session)
}

const DEFAULT_KEYSPACE_QUERY = `CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 } AND DURABLE_WRITES = true`

type CassandraDB struct {
	conn             *gocql.Session
	cluster          *gocql.ClusterConfig
	keyspace         string
	metricTable      string
	pathTable        string
	segmentTable     string
	logTable         string
	version          string
	readConsistency  gocql.Consistency
	writeConsistency gocql.Consistency

	log *logging.Logger
}

func NewCassandraDB() *CassandraDB {
	cass := new(CassandraDB)
	cass.log = logging.MustGetLogger("writers.cassandra")
	return cass
}

func (cass *CassandraDB) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (server1,server2,server3) is needed for cassandra config")
	}

	port := int(conf.Int64("port", 9042))

	cass.keyspace = conf.String("keyspace", "metric")
	cass.metricTable = conf.String("metrics_table", "metric")
	cass.pathTable = conf.String("path_able", "path")
	cass.logTable = conf.String("log_table", "metriclogs")
	cass.segmentTable = conf.String("segment_table", "segment")

	wconsistency := conf.String("write_consistency", "one")
	noSniff := !conf.Bool("sniff", false)

	cass.writeConsistency = gocql.LocalOne
	if wconsistency == "local_quorum" {
		cass.writeConsistency = gocql.LocalQuorum
	} else if wconsistency == "quorum" {
		cass.writeConsistency = gocql.Quorum
	}

	r_consistency := conf.String("read_consistency", "one")
	cass.readConsistency = gocql.LocalOne
	if r_consistency == "local_quorum" {
		cass.readConsistency = gocql.LocalQuorum
	} else if r_consistency == "quorum" {
		cass.readConsistency = gocql.Quorum
	}

	timeout := conf.Duration("timeout", time.Duration(30*time.Second))

	numcons := conf.Int64("numcons", CASSANDRA_DEFAULT_CONNECTIONS)

	con_key := fmt.Sprintf("%s:%v/keyspace:%v-tables:%v+%v+%v", dsn, port, cass.keyspace, cass.metricTable, cass.pathTable, cass.segmentTable)

	servers := strings.Split(dsn, ",")
	cluster := gocql.NewCluster(servers...)
	cluster.DisableInitialHostLookup = noSniff // for dev on docker this part takes too long
	cluster.Port = port
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}
	cluster.ProtoVersion = 0x04 //so much faster then v3
	// need to test/add for keyspace first
	err = cass.injectKeySpace(cluster)
	if err != nil {
		return err
	}

	cluster.Keyspace = cass.keyspace
	cluster.Consistency = cass.writeConsistency
	cluster.Timeout = timeout
	cluster.NumConns = int(numcons)

	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())

	_compress := conf.Bool("compress", true)
	if _compress {
		cluster.Compressor = new(gocql.SnappyCompressor)
	}

	// auth
	user := conf.String("user", "")
	pass := conf.String("pass", "")

	if user != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: user,
			Password: pass,
		}
	}

	sess_key := fmt.Sprintf("%s/%s", dsn, cass.keyspace)
	cass.log.Notice("Connecting to Cassandra (can take a bit of time) ... %s (%s)", sess_key, con_key)
	cass.conn, err = getSessionSingleton(sess_key, cluster)

	if err != nil {
		return err
	}
	cass.log.Notice("Connected to Cassandra: %v (%v)", con_key, servers)
	cass.cluster = cluster
	return nil
}

// need to inject the keyspace if not there
func (cass *CassandraDB) injectKeySpace(tCluster *gocql.ClusterConfig) error {
	ses, err := tCluster.CreateSession()
	if err != nil {
		return err
	}
	defer ses.Close()
	return ses.Query(fmt.Sprintf(DEFAULT_KEYSPACE_QUERY, cass.keyspace)).Exec()
}

func (cass *CassandraDB) GetCassandraVersion() string {
	if cass.version != "" {
		return cass.version
	}
	// get the version
	iter := cass.conn.Query("SELECT release_version FROM system.local").Iter()
	var vers string
	for iter.Scan(&vers) {
	}
	iter.Close()
	cass.version = vers
	return cass.version
}

// getters
func (cass *CassandraDB) Cluster() *gocql.ClusterConfig {
	return cass.cluster
}

// getters
func (cass *CassandraDB) MetricTable() string {
	return cass.metricTable
}
func (cass *CassandraDB) LogTableBase() string {
	return cass.logTable
}
func (cass *CassandraDB) PathTable() string {
	return cass.pathTable
}
func (cass *CassandraDB) SegmentTable() string {
	return cass.segmentTable
}
func (cass *CassandraDB) Keyspace() string {
	return cass.keyspace
}
func (cass *CassandraDB) Connection() DBConn {
	return cass.conn
}
