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

type CassandraDB struct {
	conn              *gocql.Session
	cluster           *gocql.ClusterConfig
	keyspace          string
	metric_table      string
	path_table        string
	segment_table     string
	read_consistency  gocql.Consistency
	write_consistency gocql.Consistency

	log *logging.Logger
}

func NewCassandraDB() *CassandraDB {
	cass := new(CassandraDB)
	cass.log = logging.MustGetLogger("writers.cassandra")
	return cass
}

func (cass *CassandraDB) Config(conf map[string]interface{}) (err error) {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (server1,server2,server3) is needed for cassandra config")
	}
	dsn := gots.(string)

	port := 9042
	_port := conf["port"]
	if _port != nil {
		port = int(_port.(int64))
	}

	cass.keyspace = "metric"
	_keyspace := conf["keyspace"]
	if _keyspace != nil {
		cass.keyspace = _keyspace.(string)
	}

	cass.metric_table = "metric"
	_table := conf["metrics_table"]
	if _table != nil {
		cass.metric_table = _table.(string)
	}

	cass.path_table = "path"
	_ptable := conf["path_table"]
	if _ptable != nil {
		cass.path_table = _ptable.(string)
	}
	cass.segment_table = "segment"
	_stable := conf["segment_table"]
	if _stable != nil {
		cass.segment_table = _stable.(string)
	}

	w_consistency := "one"
	cass.write_consistency = gocql.LocalOne
	_wconsistency := conf["write_consistency"]
	if _wconsistency != nil {
		w_consistency = _wconsistency.(string)
		if w_consistency == "local_quorum" {
			cass.write_consistency = gocql.LocalQuorum
		} else if w_consistency == "quorum" {
			cass.write_consistency = gocql.Quorum
		}
	}

	r_consistency := "one"
	cass.read_consistency = gocql.LocalOne
	_rconsistency := conf["read_consistency"]
	if _rconsistency != nil {
		r_consistency = _rconsistency.(string)
		if r_consistency == "local_quorum" {
			cass.read_consistency = gocql.LocalQuorum
		} else if r_consistency == "quorum" {
			cass.read_consistency = gocql.Quorum
		}
	}

	timeout := time.Duration(30 * time.Second)
	_timeout := conf["timeout"]
	if _rconsistency != nil {
		ok, err := time.ParseDuration(_timeout.(string))
		if err != nil {
			return err
		}
		timeout = ok
	}

	numcons := CASSANDRA_DEFAULT_CONNECTIONS
	_numcons := conf["numcons"]
	if _numcons != nil {
		numcons = _numcons.(int64)
	}

	// cassandra does not like batch of large proportions .. so this is rather "not relevant"
	// and we need to insert things One At A Time
	/*
		_wr_buffer := conf["batch_count"]
		if _wr_buffer == nil {
			cass.max_write_size = 100
		} else {
			// toml things generic ints are int64
			cass.max_write_size = int(_wr_buffer.(int64))
			//cassandra has a batch limit (on input queries) so we force 100
			if cass.max_write_size > 100 {
				cass.log.Warning("Batch size too large, need to reduce to 100")
				cass.max_write_size = 100
			}
		}
	*/

	con_key := fmt.Sprintf("%s:%v/%v/%v|%v|%v", dsn, port, cass.keyspace, cass.metric_table, cass.path_table, cass.segment_table)

	servers := strings.Split(dsn, ",")
	cluster := gocql.NewCluster(servers...)
	cluster.Port = port
	cluster.Keyspace = cass.keyspace
	cluster.Consistency = cass.write_consistency
	cluster.Timeout = timeout
	cluster.NumConns = int(numcons)
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}
	cluster.ProtoVersion = 0x04 //so much faster then v3

	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())

	compress := true
	_compress := conf["compress"]
	if _compress != nil {
		compress = _compress.(bool)
	}
	if compress {
		cluster.Compressor = new(gocql.SnappyCompressor)
	}

	// auth
	user := ""
	_user := conf["user"]
	if _user != nil {
		user = _user.(string)
	}
	pass := ""
	_pass := conf["pass"]
	if _pass != nil {
		pass = _pass.(string)
	}

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

// getters
func (cass *CassandraDB) Cluster() *gocql.ClusterConfig {
	return cass.cluster
}

// getters
func (cass *CassandraDB) MetricTable() string {
	return cass.metric_table
}
func (cass *CassandraDB) PathTable() string {
	return cass.path_table
}
func (cass *CassandraDB) SegmentTable() string {
	return cass.segment_table
}
func (cass *CassandraDB) Keyspace() string {
	return cass.keyspace
}
func (cass *CassandraDB) Connection() DBConn {
	return cass.conn
}
