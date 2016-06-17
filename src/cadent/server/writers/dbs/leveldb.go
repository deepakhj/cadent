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
	DEFAULT_LEVELDB_READ_CACHE_SIZE = 8 * 1024 * 1024
	DEFAULT_LEVELDB_INDEX_FILE      = "filemap.db"
	DEFAULT_LEVELDB_TAG_FILE        = "tags.db"
)

/****************** Interfaces *********************/
type LevelDB struct {
	index_conn *leveldb.DB
	tag_conn   *leveldb.DB
	table_path string
	index_file string
	tag_file   string

	read_opts  *leveldb_opt.Options
	write_opts *leveldb_opt.Options

	log *logging.Logger
}

func NewLevelDB() *LevelDB {
	my := new(LevelDB)
	my.log = logging.MustGetLogger("writers.leveldb")
	return my
}

func (my *LevelDB) Config(conf map[string]interface{}) (err error) {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (/path/to/db/folder) is needed for leveldb config")
	}
	dsn := gots.(string)

	my.index_file = DEFAULT_LEVELDB_INDEX_FILE
	my.tag_file = DEFAULT_LEVELDB_TAG_FILE

	my.table_path = dsn
	my.read_opts = new(leveldb_opt.Options)
	my.read_opts.Filter = leveldb_filter.NewBloomFilter(10)
	my.read_opts.BlockCacheCapacity = DEFAULT_LEVELDB_READ_CACHE_SIZE

	_c_size := conf["read_cache_size"]
	if _c_size != nil {
		my.read_opts.BlockCacheCapacity = _c_size.(int)
	}

	my.index_conn, err = leveldb.OpenFile(my.IndexTableName(), my.read_opts)
	if err != nil {
		return err
	}

	my.tag_conn, err = leveldb.OpenFile(my.TagTableName(), my.read_opts)
	if err != nil {
		return err
	}

	return nil
}

func (my *LevelDB) IndexTableName() string {
	return filepath.Join(my.table_path, my.index_file)
}

func (my *LevelDB) TagTableName() string {
	return filepath.Join(my.table_path, my.tag_file)
}

func (my *LevelDB) Connection() DBConn {
	return my.index_conn
}
