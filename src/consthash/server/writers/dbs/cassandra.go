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
	"time"
)

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
		port = _port.(int)
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

	numcons := 20
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

	cass.log.Notice("Connecting to Cassandra (can take a bit of time) ... %s", dsn)

	servers := strings.Split(dsn, ",")
	cluster := gocql.NewCluster(servers...)
	cluster.Port = port
	cluster.Keyspace = cass.keyspace
	cluster.Consistency = cass.write_consistency
	cluster.Timeout = timeout
	cluster.NumConns = int(numcons)
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}
	cluster.ProtoVersion = 0x04 //

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

	/*
		if *compress {
			cluster.Compressor = new(gocql.SnappyCompressor)
		}

		if *policy == "token" {
			cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
		}
	*/
	cass.conn, err = cluster.CreateSession()

	if err != nil {
		return err
	}
	cass.log.Notice("Connected to Cassandra: %v", servers)
	cass.cluster = cluster
	// Not needed as cass does not like Big insert queries
	// go cass.PeriodicFlush()

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
