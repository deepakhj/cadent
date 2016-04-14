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

	OPTIONS: For `Config`

		table: base table name (default: metrics)
		prefix: table prefix if any (_1s, _5m)
		batch_count: batch this many inserts for much faster insert performance (default 1000)
		periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package indexer

import (
	"cadent/server/writers/dbs"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"sync"
	"time"
)

type MyPath struct {
	Path   string
	Length int
}

/****************** Interfaces *********************/
type MySQLIndexer struct {
	db          *dbs.MySQLDB
	conn        *sql.DB
	resolutions [][]int

	write_list     []string      // buffer the writes so as to do "multi" inserts per query
	max_write_size int           // size of that buffer before a flush
	max_idle       time.Duration // either max_write_size will trigger a write or this time passing will
	write_lock     sync.Mutex

	log *logging.Logger
}

func NewMySQLIndexer() *MySQLIndexer {
	my := new(MySQLIndexer)
	my.log = logging.MustGetLogger("writers.mysql")
	return my
}

func (my *MySQLIndexer) Config(conf map[string]interface{}) error {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}
	dsn := gots.(string)
	db, err := dbs.NewDB("mysql", dsn, conf)
	if err != nil {
		return err
	}

	my.db = db.(*dbs.MySQLDB)
	my.conn = db.Connection().(*sql.DB)

	_wr_buffer := conf["batch_count"]
	if _wr_buffer == nil {
		my.max_write_size = 1000
	} else {
		// toml things generic ints are int64
		my.max_write_size = int(_wr_buffer.(int64))
	}

	_pr_flush := conf["periodic_flush"]
	my.max_idle = time.Duration(time.Second)
	if _pr_flush != nil {
		dur, err := time.ParseDuration(_pr_flush.(string))
		if err == nil {
			my.max_idle = dur
		} else {
			my.log.Error("Mysql Driver: Invalid Duration `%v`", _pr_flush)
		}
	}

	go my.PeriodFlush()

	return nil
}

func (my *MySQLIndexer) Stop() {
	//noop
}

func (my *MySQLIndexer) PeriodFlush() {
	for {
		time.Sleep(my.max_idle)
		my.Flush()
	}
	return
}

func (my *MySQLIndexer) Flush() (int, error) {
	my.write_lock.Lock()
	defer my.write_lock.Unlock()

	l := len(my.write_list)
	if l == 0 {
		return 0, nil
	}

	pthQ := fmt.Sprintf(
		"INSERT IGNORE INTO %s (path, length) VALUES ",
		my.db.PathTable(),
	)

	pvals := []interface{}{}

	for _, stat := range my.write_list {
		pthQ += "(?, ?), "
		pvals = append(pvals, stat, len(strings.Split(stat, ".")))

	}
	//trim the last ", "
	pthQ = pthQ[0 : len(pthQ)-2]

	//prepare the statement
	stmt, err := my.conn.Prepare(pthQ)
	if err != nil {
		my.log.Error("Mysql Driver: Indexer Path prepare failed, %v", err)
		return 0, err
	}
	defer stmt.Close()

	//format all vals at once
	_, err = stmt.Exec(pvals...)
	if err != nil {
		my.log.Error("Mysql Driver: Path insert failed, %v", err)
		return 0, err
	}

	return l, nil
}

func (my *MySQLIndexer) Write(skey string) error {

	if len(my.write_list) > my.max_write_size {
		_, err := my.Flush()
		if err != nil {
			return err
		}
	}

	// Flush can cause double locking
	my.write_lock.Lock()
	defer my.write_lock.Unlock()
	my.write_list = append(my.write_list, skey)
	return nil
}

/**** READER ***/
// XXX TODO
func (my *MySQLIndexer) Find(metric string) (MetricFindItems, error) {
	return MetricFindItems{}, fmt.Errorf("MYSQL INDEXER NOT YET DONE")
}

func (my *MySQLIndexer) Expand(metric string) (MetricExpandItem, error) {
	return MetricExpandItem{}, fmt.Errorf("MYSQL INDEXER NOT YET DONE")
}
