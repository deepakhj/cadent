/*

LevelDB key/value store for local saving for indexes on the stat key space
and how it maps to files on disk.

This is much like how Prometheus does things, stores a index in levelDB
and single "custom" files for each series.

We have "2" dbs, one for a
{key} -> {file} mapping
and
{tag} -> {key} mapping.

If no tags are used (i.e. graphite like) then this will be basically empty

*/

package dbs

import (
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_filter "github.com/syndtr/goleveldb/leveldb/filter"
	leveldb_opt "github.com/syndtr/goleveldb/leveldb/opt"

	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"path/filepath"
)

const (
	DEFAULT_LEVELDB_READ_CACHE_SIZE = 8 * leveldb_opt.MiB
	DEFAULT_LEVELDB_FILE_SIZE       = 20 * leveldb_opt.MiB
	DEFAULT_LEVELDB_SEGMENT_FILE    = "segments"
	DEFAULT_LEVELDB_TAG_FILE        = "tags"
)

/****************** Interfaces *********************/
type LevelDB struct {
	segment_conn *leveldb.DB
	path_conn    *leveldb.DB
	tag_conn     *leveldb.DB
	table_path   string
	segment_file string
	tag_file     string
	path_file    string

	level_opts *leveldb_opt.Options

	log *logging.Logger
}

func NewLevelDB() *LevelDB {
	lb := new(LevelDB)
	lb.log = logging.MustGetLogger("writers.leveldb")
	return lb
}

func (lb *LevelDB) Config(conf map[string]interface{}) (err error) {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (/path/to/db/folder) is needed for leveldb config")
	}
	dsn := gots.(string)

	lb.segment_file = DEFAULT_LEVELDB_SEGMENT_FILE
	lb.tag_file = DEFAULT_LEVELDB_TAG_FILE

	lb.table_path = dsn
	lb.level_opts = new(leveldb_opt.Options)
	lb.level_opts.Filter = leveldb_filter.NewBloomFilter(10)
	lb.level_opts.BlockCacheCapacity = DEFAULT_LEVELDB_READ_CACHE_SIZE
	lb.level_opts.CompactionTableSize = DEFAULT_LEVELDB_FILE_SIZE

	_c_size := conf["read_cache_size"]
	if _c_size != nil {
		lb.level_opts.BlockCacheCapacity = _c_size.(int)
	}
	_w_size := conf["file_compact_size"]
	if _w_size != nil {
		lb.level_opts.CompactionTableSize = _w_size.(int)
	}

	lb.tag_conn, err = leveldb.OpenFile(lb.TagTableName(), lb.level_opts)
	if err != nil {
		return err
	}

	lb.segment_conn, err = leveldb.OpenFile(lb.SegmentTableName(), lb.level_opts)
	if err != nil {
		return err
	}

	return nil
}

func (lb *LevelDB) TagTableName() string {
	return filepath.Join(lb.table_path, lb.tag_file)
}

func (lb *LevelDB) SegmentTableName() string {
	return filepath.Join(lb.table_path, lb.segment_file)
}
func (lb *LevelDB) SegmentConn() *leveldb.DB {
	return lb.segment_conn
}
func (lb *LevelDB) TagConn() *leveldb.DB {
	return lb.tag_conn
}

// this is just for the interface match ..
// real users need to cast this to a real LevelDB obj
func (lb *LevelDB) Connection() DBConn {
	return lb.path_file
}

func (lb *LevelDB) Close() (err error) {
	if lb.tag_conn != nil {
		err = lb.tag_conn.Close()
	}
	if lb.segment_conn != nil {
		err = lb.segment_conn.Close()
	}
	return err
}
