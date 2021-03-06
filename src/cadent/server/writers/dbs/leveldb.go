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

LevelDB key/value store for local saving for indexes on the stat key space
and how it maps to files on disk.

We have "2" dbs, one for a
{segment} -> {path|uid} mapping
and
{tag} -> {uid} mapping.

If no tags are used (i.e. graphite like) then this will be basically empty

*/

package dbs

import (
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_filter "github.com/syndtr/goleveldb/leveldb/filter"
	leveldb_opt "github.com/syndtr/goleveldb/leveldb/opt"

	"cadent/server/utils/options"
	"fmt"
	leveldb_storage "github.com/syndtr/goleveldb/leveldb/storage"
	logging "gopkg.in/op/go-logging.v1"
	"path/filepath"
)

const (
	DEFAULT_LEVELDB_READ_CACHE_SIZE = 8 * leveldb_opt.MiB
	DEFAULT_LEVELDB_FILE_SIZE       = 20 * leveldb_opt.MiB
	DEFAULT_LEVELDB_SEGMENT_FILE    = "segments"
	DEFAULT_LEVELDB_PATH_FILE       = "paths"
	DEFAULT_LEVELDB_UID_FILE        = "uids"
	DEFAULT_LEVELDB_TAG_FILE        = "tags"
	DEFAULT_LEVELDB_TAGXREF_FILE    = "tags_xref"
)

/****************** Interfaces *********************/
type LevelDB struct {
	segment_conn *leveldb.DB
	path_conn    *leveldb.DB
	tag_conn     *leveldb.DB
	tagxref_conn *leveldb.DB
	uid_conn     *leveldb.DB
	table_path   string
	segment_file string
	tag_file     string
	path_file    string
	uid_file     string
	tagxref_file string

	level_opts *leveldb_opt.Options

	log *logging.Logger
}

func NewLevelDB() *LevelDB {
	lb := new(LevelDB)
	lb.log = logging.MustGetLogger("writers.leveldb")
	return lb
}

func (lb *LevelDB) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (/path/to/db/folder) is needed for leveldb config")
	}

	lb.segment_file = DEFAULT_LEVELDB_SEGMENT_FILE
	lb.tag_file = DEFAULT_LEVELDB_TAG_FILE
	lb.path_file = DEFAULT_LEVELDB_PATH_FILE
	lb.uid_file = DEFAULT_LEVELDB_UID_FILE
	lb.tagxref_file = DEFAULT_LEVELDB_TAGXREF_FILE

	lb.table_path = dsn
	lb.level_opts = new(leveldb_opt.Options)
	lb.level_opts.Filter = leveldb_filter.NewBloomFilter(10)

	lb.level_opts.BlockCacheCapacity = int(conf.Int64("read_cache_size", DEFAULT_LEVELDB_READ_CACHE_SIZE))
	lb.level_opts.CompactionTableSize = int(conf.Int64("file_compact_size", DEFAULT_LEVELDB_FILE_SIZE))

	lb.tag_conn, err = leveldb.OpenFile(lb.TagTableName(), lb.level_opts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("Tab DB is corrupt. Recovering.")
			lb.segment_conn, err = leveldb.RecoverFile(lb.TagTableName(), lb.level_opts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lb.segment_conn, err = leveldb.OpenFile(lb.SegmentTableName(), lb.level_opts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("Segment DB is corrupt. Recovering.")
			lb.segment_conn, err = leveldb.RecoverFile(lb.SegmentTableName(), lb.level_opts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lb.path_conn, err = leveldb.OpenFile(lb.PathTableName(), lb.level_opts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("Path DB is corrupt. Recovering.")
			lb.segment_conn, err = leveldb.RecoverFile(lb.PathTableName(), lb.level_opts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lb.uid_conn, err = leveldb.OpenFile(lb.UidTableName(), lb.level_opts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("UID DB is corrupt. Recovering.")
			lb.uid_conn, err = leveldb.RecoverFile(lb.UidTableName(), lb.level_opts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	lb.tagxref_conn, err = leveldb.OpenFile(lb.TagXrefTableName(), lb.level_opts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("TagXref DB is corrupt. Recovering.")
			lb.uid_conn, err = leveldb.RecoverFile(lb.TagXrefTableName(), lb.level_opts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (lb *LevelDB) TagTableName() string {
	return filepath.Join(lb.table_path, lb.tag_file)
}

func (lb *LevelDB) TagXrefTableName() string {
	return filepath.Join(lb.table_path, lb.tagxref_file)
}

func (lb *LevelDB) SegmentTableName() string {
	return filepath.Join(lb.table_path, lb.segment_file)
}

func (lb *LevelDB) PathTableName() string {
	return filepath.Join(lb.table_path, lb.path_file)
}

func (lb *LevelDB) UidTableName() string {
	return filepath.Join(lb.table_path, lb.uid_file)
}

func (lb *LevelDB) SegmentConn() *leveldb.DB {
	return lb.segment_conn
}

func (lb *LevelDB) PathConn() *leveldb.DB {
	return lb.path_conn
}

func (lb *LevelDB) UidConn() *leveldb.DB {
	return lb.uid_conn
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
	if lb.uid_conn != nil {
		err = lb.uid_conn.Close()
	}
	if lb.path_conn != nil {
		err = lb.path_conn.Close()
	}
	if lb.tagxref_conn != nil {
		err = lb.tagxref_conn.Close()
	}
	return err
}
