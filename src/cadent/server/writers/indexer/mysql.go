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
	THe MySQL indexer.

	Both the Cassandra and MySQL indexers share the same basic table space and methods


CREATE TABLE `{segment_table}` (
  `segment` varchar(255) NOT NULL DEFAULT '',
  `pos` int NOT NULL,
  PRIMARY KEY (`pos`, `segment`)
);

CREATE TABLE `{path_table}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
  `segment` varchar(255) NOT NUL,
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
	"cadent/server/dispatch"
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MYSQL_INDEXER_QUEUE_LEN = 1024 * 1024
	MYSQL_INDEXER_WORKERS   = 8
	MYSQL_WRITES_PER_SECOND = 200
)

/****************** Interfaces *********************/
type MySQLIndexer struct {
	db        *dbs.MySQLDB
	conn      *sql.DB
	indexerId string

	write_lock sync.Mutex

	num_workers int
	queue_len   int
	_accept     bool //shtdown notice

	dispatcher *dispatch.DispatchQueue

	cache             *Cacher // simple cache to rate limit and buffer writes
	writes_per_second int     // rate limit writer

	shutitdown uint32
	shutdown   chan bool
	startstop  utils.StartStop

	// tag ID cache so we don't do a bunch of unessesary inserts
	tagMu      sync.RWMutex
	tagIdCache map[string]int64

	log *logging.Logger
}

func NewMySQLIndexer() *MySQLIndexer {
	my := new(MySQLIndexer)
	my.log = logging.MustGetLogger("indexer.mysql")
	my.tagIdCache = make(map[string]int64, 0)
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

	// tweak queues and worker sizes
	_workers := conf["write_workers"]
	my.num_workers = MYSQL_INDEXER_WORKERS
	if _workers != nil {
		my.num_workers = int(_workers.(int64))
	}

	_qs := conf["write_queue_length"]
	my.queue_len = MYSQL_INDEXER_QUEUE_LEN
	if _qs != nil {
		my.queue_len = int(_qs.(int64))
	}

	my.indexerId = "indexer:mysql:" + dsn
	my.cache, err = getCacherSingleton(my.indexerId)
	if err != nil {
		return err
	}

	_ms := conf["cache_index_size"]
	if _ms != nil {
		my.cache.maxKeys = int(_ms.(int64))
	}

	my.writes_per_second = MYSQL_WRITES_PER_SECOND
	_ws := conf["writes_per_second"]
	if _ws != nil {
		my.writes_per_second = int(_ws.(int64))
	}

	atomic.StoreUint32(&my.shutitdown, 1)
	my.shutdown = make(chan bool)

	return nil
}
func (my *MySQLIndexer) Name() string { return my.indexerId }

func (my *MySQLIndexer) Start() {
	my.startstop.Start(func() {
		my.log.Notice("starting mysql indexer: %s", my.Name())

		workers := my.num_workers
		retries := 2
		my.dispatcher = dispatch.NewDispatchQueue(workers, my.queue_len, retries)
		my.dispatcher.Start()

		my.cache.Start() //start cacher

		go my.sendToWriters() // the dispatcher
	})
}

func (my *MySQLIndexer) Stop() {
	my.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		my.log.Notice("shutting down mysql indexer: %s", my.Name())
		my.cache.Stop()
		if my.dispatcher != nil {
			my.dispatcher.Stop()
		}
	})
}

// pop from the cache and send to actual writers
func (my *MySQLIndexer) sendToWriters() error {
	// this may not be the "greatest" ratelimiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backedup
	// it will block on the write_queue stage

	if my.writes_per_second <= 0 {
		my.log.Notice("Starting indexer writer: No rate limiting enabled")
		for {
			if my.shutitdown == 1 {
				return nil
			}
			skey := my.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.cassandra.write.send-to-writers"), 1)
				my.dispatcher.Add(&MysqlIndexerJob{Msql: my, Stat: skey})
			}
		}
	} else {
		sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(my.writes_per_second))
		my.log.Notice("Starting indexer writer: limiter every %f nanoseconds (%d writes per second)", sleep_t, my.writes_per_second)
		dur := time.Duration(int(sleep_t))
		for {
			if my.shutitdown == 1 {
				return nil
			}
			skey := my.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.cassandra.write.send-to-writers"), 1)
				my.dispatcher.Add(&MysqlIndexerJob{Msql: my, Stat: skey})
				time.Sleep(dur)
			}
		}
	}
}

// keep an index of the stat keys and their fragments so we can look up
func (my *MySQLIndexer) Write(skey repr.StatName) error {
	return my.cache.Add(skey)
}

func (my *MySQLIndexer) WriteTags(inname *repr.StatName, do_main bool, do_meta bool) error {

	have_meta := !inname.MetaTags.IsEmpty()
	have_tgs := !inname.Tags.IsEmpty()
	if !have_tgs && !have_meta || (!do_main && !do_meta) {
		return nil
	}

	tx, err := my.conn.Begin()
	if err != nil {
		my.log.Error("Failure in Index Write getting transations: %v", tx)
		return err
	}

	unique_ID := inname.UniqueIdString()

	// Tag Time
	tagQ := fmt.Sprintf(
		"INSERT INTO %s (name, value, is_meta) VALUES (?, ?, ?)",
		my.db.TagTable(),
	)
	if have_tgs {
		my.tagMu.Lock()
		defer my.tagMu.Unlock()
	}

	var tag_ids []int64
	if have_tgs && do_main {

		for _, tag := range inname.Tags.Tags() {
			tag_key := fmt.Sprintf("%s:%s:%v", tag[0], tag[1], true)
			gots, ok := my.tagIdCache[tag_key]
			if ok {
				tag_ids = append(tag_ids, gots)
			} else {
				res, err := tx.Exec(tagQ, tag[0], tag[1], false)
				if err != nil {
					if strings.Contains(fmt.Sprintf("%s", err), "Duplicate entry") {
						id, err := my.FindTagId(tag[0], tag[1], false)
						if err == nil {
							my.tagIdCache[tag_key] = id
							continue
						}
					}

					my.log.Error("Could not write tag %v: %v", tag, err)
					continue
				}
				id, err := res.LastInsertId()
				if err != nil {
					my.log.Error("Could not get insert tag ID %v: %v", tag, err)
					continue
				}
				tag_ids = append(tag_ids, id)
				my.tagIdCache[tag_key] = id
			}
		}
	}

	if have_meta && do_meta {
		for _, tag := range inname.MetaTags.Tags() {
			tag_key := fmt.Sprintf("%s:%s:%v", tag[0], tag[1], true)
			gots, ok := my.tagIdCache[tag_key]

			if ok {
				tag_ids = append(tag_ids, gots)
			} else {
				res, err := tx.Exec(tagQ, tag[0], tag[1], true)
				if err != nil {
					if strings.Contains(fmt.Sprintf("%s", err), "Duplicate entry") {
						id, err := my.FindTagId(tag[0], tag[1], true)
						if err == nil {
							my.tagIdCache[tag_key] = id
							continue
						}
					}
					my.log.Error("Could not write tag %v: %v", tag, err)
					continue
				}
				id, err := res.LastInsertId()
				if err != nil {
					my.log.Error("Could not get insert tag ID %v: %v", tag, err)
					continue
				}
				tag_ids = append(tag_ids, id)
				my.tagIdCache[tag_key] = id
			}
		}
	}

	if len(tag_ids) > 0 {
		tagQxr := fmt.Sprintf(
			"INSERT IGNORE INTO %s (tag_id, uid) VALUES ",
			my.db.TagTableXref(),
		)
		var val_q []string
		var q_bits []interface{}
		for _, tg := range tag_ids {
			val_q = append(val_q, "(?,?)")
			q_bits = append(q_bits, []interface{}{tg, unique_ID}...)
		}
		_, err := tx.Exec(tagQxr+strings.Join(val_q, ","), q_bits...)
		if err != nil {
			my.log.Error("Could not get insert tag UID-ID: %v", err)
		}
	}
	err = tx.Commit()
	if err != nil {
		my.log.Error("Could Tag commit failed: %v", err)
		return err
	}
	// and we are done
	return nil
}

// a basic clone of the cassandra indexer
func (my *MySQLIndexer) WriteOne(inname *repr.StatName) error {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("indexer.mysql.write.path-time-ns"), time.Now())
	stats.StatsdClientSlow.Incr("indexer.mysql.noncached-writes-path", 1)

	skey := inname.Key
	s_parts := strings.Split(skey, ".")
	p_len := len(s_parts)

	// we are going to assume that if the path is already in the system, we've indexed it and therefore
	// do not need to do the super loop (which is very expensive)
	SelQ := fmt.Sprintf(
		"SELECT path, length, has_data FROM %s WHERE pos=? AND segment=?",
		my.db.PathTable(),
	)

	var _pth string
	var _len int
	var _dd bool

	rows, gerr := my.conn.Query(SelQ, p_len-1, skey)

	// already indexed
	if gerr == nil {
		for rows.Next() {
			if err := rows.Scan(&_pth, &_len, &_dd); err != nil {
				if _pth == skey && _dd && _len == p_len-1 {
					return my.WriteTags(inname, false, true)
				}
			}
		}
	}

	// just use the Cassandra Base objects
	cur_part := ""
	segments := []CassSegment{}
	paths := []CassPath{}
	unique_ID := inname.UniqueIdString()

	for idx, part := range s_parts {
		if len(cur_part) > 1 {
			cur_part += "."
		}
		cur_part += part
		on_segment := CassSegment{
			Segment: cur_part,
			Pos:     idx,
		}
		segments = append(segments, on_segment)

		on_path := CassPath{
			Id:      unique_ID,
			Segment: on_segment,
			Path:    skey,
			Length:  p_len - 1, // starts at 0
		}

		paths = append(paths, on_path)
	}

	// begin the big transation
	tx, err := my.conn.Begin()
	if err != nil {
		my.log.Error("Failure in Index Write getting transations: %v", tx)
		return err
	}

	last_path := paths[len(paths)-1]
	// now to upsert them all (inserts in cass are upserts)
	for idx, seg := range segments {
		Q := fmt.Sprintf(
			"INSERT IGNORE INTO %s (pos, segment) VALUES  (?, ?) ",
			my.db.SegmentTable(),
		)
		_, err := tx.Exec(Q, seg.Pos, seg.Segment)

		if err != nil {
			my.log.Error("Could not insert segment %v", seg)
			stats.StatsdClientSlow.Incr("indexer.mysql.segment-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.mysql.segment-writes", 1)
		}

		// now for each "partial path" add in the fact that it's not a "data" node
		// for each "segment" add in the path to do a segment to path(s) lookup
		// the skey one obviously has data
		// if key is consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
		/* insert

		consthash -> consthash.zipperwork
		consthash.zipperwork -> consthash.zipperwork.local
		consthash.zipperwork.local -> consthash.zipperwork.local.writer
		consthash.zipperwork.local.writer -> consthash.zipperwork.local.writer.cassandra
		consthash.zipperwork.local.writer.cassandra -> consthash.zipperwork.local.writer.cassandra.write
		consthash.zipperwork.local.writer.cassandra.write -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns

		as data-less nodes

		*/
		if skey != seg.Segment && idx < len(paths)-2 {
			Q = fmt.Sprintf(
				"INSERT INTO %s (segment, pos, path, uid, length, has_data) VALUES  (?, ?, ?, ?, ?, ?)",
				my.db.PathTable(),
			)
			_, err = tx.Exec(Q,
				seg.Segment, seg.Pos, seg.Segment+"."+s_parts[idx+1], unique_ID, seg.Pos+1, false,
			)
			//cass.log.Critical("NODATA:: Seg INS: %s PATH: %s Len: %d", seg.Segment, seg.Segment+"."+s_parts[idx+1], seg.Pos)
		}

		// for each "segment" add in the path to do a segment to path(s) lookup
		/*
			for key consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			insert
			consthash -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local.writer -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local.writer.cassandra -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local.writer.cassandra.write -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local.writer.cassandra.write.metric-time-ns -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99

		*/
		Q = fmt.Sprintf(
			"INSERT INTO %s (segment, pos, path, uid, length, has_data) VALUES  (?, ?, ?, ?, ?, ?)",
			my.db.PathTable(),
		)

		/*
			segment frozen<segment_pos>,
			path text,
			length int
		*/
		_, err = tx.Exec(Q,
			seg.Segment, seg.Pos, skey, unique_ID, p_len-1, true,
		)
		//cass.log.Critical("DATA:: Seg INS: %s PATH: %s Len: %d", seg.Segment, skey, p_len-1)

		if err != nil {
			my.log.Error("Could not insert path %v (%v) :: %v", last_path, unique_ID, err)
			stats.StatsdClientSlow.Incr("indexer.mysql.path-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.mysql.path-writes", 1)
		}
	}
	if err != nil {
		my.log.Error("Could not write index", err)
	}

	err = my.WriteTags(inname, true, true)
	if err != nil {
		my.log.Error("Could not write tag index", err)
		return err
	}
	err = tx.Commit()

	return err
}

func (my *MySQLIndexer) Delete(name *repr.StatName) error {
	pthQ := fmt.Sprintf(
		"DELETE FROM %s WHERE uid=? AND path=? AND length=?",
		my.db.PathTable(),
	)

	tagQ := fmt.Sprintf(
		"DELETE FROM %s WHERE uid=?",
		my.db.TagTableXref(),
	)

	uid := name.UniqueIdString()
	pvals := []interface{}{uid, name.Key, len(strings.Split(name.Key, "."))}

	tx, err := my.conn.Begin()

	//format all vals at once
	_, err = tx.Exec(pthQ, pvals...)
	if err != nil {
		my.log.Error("Mysql Driver: Delete Path failed, %v", err)
		return err
	}

	//format all vals at once
	_, err = tx.Exec(tagQ, uid)
	if err != nil {
		my.log.Error("Mysql Driver: Delete Tags failed, %v", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

/**** READER ***/

func (my *MySQLIndexer) ExpandNonRegex(metric string) (MetricExpandItem, error) {
	paths := strings.Split(metric, ".")
	m_len := len(paths)
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=? AND segment=?",
		my.db.SegmentTable(),
	)

	var on_pth string
	var me MetricExpandItem

	rows, err := my.conn.Query(cass_Q, m_len-1, metric)
	if err != nil {
		return me, err
	}

	for rows.Next() {
		// just grab the "n+1" length ones
		err = rows.Scan(&on_pth)
		if err != nil {
			return me, err
		}
		me.Results = append(me.Results, on_pth)
	}
	if err := rows.Close(); err != nil {
		return me, err
	}
	return me, nil
}

// Expand simply pulls out any regexes into full form
func (my *MySQLIndexer) Expand(metric string) (MetricExpandItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.expand.get-time-ns", time.Now())

	needs_regex := needRegex(metric)
	//cass.log.Debug("REGSS: %v, %s", needs_regex, metric)

	if !needs_regex {
		return my.ExpandNonRegex(metric)
	}
	paths := strings.Split(metric, ".")
	m_len := len(paths)

	var me MetricExpandItem

	the_reg, err := regifyKey(metric)

	if err != nil {
		return me, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		my.db.SegmentTable(),
	)

	rows, err := my.conn.Query(cass_Q, m_len-1)
	if err != nil {
		return me, err
	}
	var seg string
	for rows.Next() {
		// just grab the "n+1" length ones
		err = rows.Scan(&seg)
		if err != nil {
			return me, err
		}
		//cass.log.Debug("SEG: %s", seg)

		if !the_reg.Match([]byte(seg)) {
			continue
		}
		me.Results = append(me.Results, seg)
	}
	if err := rows.Close(); err != nil {
		return me, err
	}
	return me, nil

}

func (my *MySQLIndexer) FindTagId(name string, value string, ismeta bool) (int64, error) {
	sel_tagQ := fmt.Sprintf(
		"SELECT id FROM %s WHERE name=? AND value=? AND is_meta=?",
		my.db.TagTable(),
	)

	rows, err := my.conn.Query(sel_tagQ, name, value, ismeta)
	if err != nil {
		my.log.Error("Error Getting Tags: %s : %v", sel_tagQ, err)
		return 0, err
	}
	var id int64
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			my.log.Error("Error Getting Tags Iterator: %s : %v", sel_tagQ, err)
			return 0, err
		}
		return id, nil
	}
	return 0, nil
}

func (my *MySQLIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags) {

	tag_table := my.db.TagTable()
	tag_xr_table := my.db.TagTableXref()

	tgsQ := fmt.Sprintf(
		"SELECT name,value,is_meta FROM %s WHERE id IN ( SELECT tag_id FROM %s WHERE uid = ? )",
		tag_table, tag_xr_table,
	)

	var name string
	var value string
	var is_meta bool
	rows, err := my.conn.Query(tgsQ, unique_id)
	if err != nil {
		my.log.Error("Error Getting Tags: %s : %v", tgsQ, err)
		return
	}
	for rows.Next() {
		err = rows.Scan(&name, &value, &is_meta)
		if err != nil {
			my.log.Error("Error Getting Tags Iterator: %s : %v", tgsQ, err)
		}
		if is_meta {
			metatags = metatags.Set(name, value)
		} else {
			tags = tags.Set(name, value)
		}
	}
	return tags, metatags
}

// basic find for non-regex items
func (my *MySQLIndexer) FindNonRegex(metric string) (MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.findnoregex.get-time-ns", time.Now())

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone down

	paths := strings.Split(metric, ".")
	m_len := len(paths)

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes
	pathQ := fmt.Sprintf(
		"SELECT uid,path,length,has_data FROM %s WHERE pos=? AND segment=?",
		my.db.PathTable(),
	)

	var mt MetricFindItems
	var ms MetricFindItem
	var on_pth string
	var pth_len int
	var id string
	var has_data bool

	rows, err := my.conn.Query(pathQ, m_len-1, metric)
	if err != nil {
		return mt, err
	}
	for rows.Next() {
		err = rows.Scan(&id, &on_pth, &pth_len, &has_data)
		if err != nil {
			return mt, err
		}
		if pth_len > m_len {
			continue
		}
		//cass.log.Critical("NON REG:::::PATH %s LEN %d m_len: %d", on_pth, pth_len, m_len)
		spl := strings.Split(on_pth, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = on_pth
		ms.Path = on_pth

		if has_data {
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
			ms.UniqueId = id

			// grab ze tags
			ms.Tags, ms.MetaTags = my.GetTagsByUid(id)

		} else {
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
		}

		mt = append(mt, ms)
	}

	if err := rows.Close(); err != nil {
		return mt, err
	}

	return mt, nil
}

// special case for "root" == "*" finder
func (my *MySQLIndexer) FindRoot() (MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.findroot.get-time-ns", time.Now())

	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		my.db.SegmentTable(),
	)

	var mt MetricFindItems
	var seg string

	rows, err := my.conn.Query(cass_Q, 0)
	if err != nil {
		return mt, err
	}
	for rows.Next() {

		err = rows.Scan(&seg)
		if err != nil {
			return mt, err
		}
		var ms MetricFindItem
		ms.Text = seg
		ms.Id = seg
		ms.Path = seg
		ms.UniqueId = ""
		ms.Expandable = 1
		ms.Leaf = 0
		ms.AllowChildren = 1

		mt = append(mt, ms)

	}

	if err := rows.Close(); err != nil {
		return mt, err

	}

	return mt, nil
}

// to allow for multiple targets
func (my *MySQLIndexer) Find(metric string) (MetricFindItems, error) {
	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide regex abilities
	// on the server side

	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.find.get-time-ns", time.Now())

	// special case for "root" == "*"

	if metric == "*" {
		return my.FindRoot()
	}

	paths := strings.Split(metric, ".")
	m_len := len(paths)

	//if the last fragment is "*" then we really mean just then next level, not another "." level
	// this is the graphite /find?query=consthash.zipperwork.local which will mean the same as
	// /find?query=consthash.zipperwork.local.* for us
	if paths[len(paths)-1] == "*" {
		metric = strings.Join(paths[:len(paths)-1], ".")
		paths = strings.Split(metric, ".")
		m_len = len(paths)
	}

	needs_regex := needRegex(metric)

	//cass.log.Debug("HasReg: %v Metric: %s", needs_regex, metric)
	if !needs_regex {
		return my.FindNonRegex(metric)
	}

	// convert the "graphite regex" into something golang understands (just the "."s really)
	// need to replace things like "moo*" -> "moo.*" but careful not to do "..*"
	// the "graphite" globs of {moo,goo} we can do with (moo|goo) so convert { -> (, , -> |, } -> )
	the_reg, err := regifyKey(metric)

	if err != nil {
		return nil, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		my.db.SegmentTable(),
	)

	var mt MetricFindItems
	rows, err := my.conn.Query(cass_Q, m_len-1)
	if err != nil {
		return mt, err
	}

	var seg string
	for rows.Next() {

		err = rows.Scan(&seg)
		if err != nil {
			return mt, err
		}

		if !the_reg.MatchString(seg) {
			continue
		}
		items, err := my.FindNonRegex(seg)
		if err != nil {
			my.log.Warning("could not get segments for `%s` :: %v", seg, err)
			continue
		}
		if items != nil && len(items) > 0 {
			for _, ms := range items {
				mt = append(mt, ms)
			}
			items = nil
		}

	}

	if err := rows.Close(); err != nil {
		return mt, err
	}

	return mt, nil
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type MysqlIndexerJob struct {
	Msql  *MySQLIndexer
	Stat  repr.StatName
	retry int
}

func (j *MysqlIndexerJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j *MysqlIndexerJob) OnRetry() int {
	return j.retry
}

func (j *MysqlIndexerJob) DoWork() error {
	err := j.Msql.WriteOne(&j.Stat)
	if err != nil {
		j.Msql.log.Error("Insert failed for Index: %v retrying ...", j.Stat)
	}
	return err
}
