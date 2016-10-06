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
	THe ElasticSearch indexer.


	A simple mapping to uids and tags/paths

	{
		uid: [string]
		path: [string]
		tags:[ {name:[string], value: [string], is_meta:[bool]},...]
	}

*/

package indexer

import (
	"cadent/server/dispatch"
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gopkg.in/olivere/elastic.v3"
	logging "gopkg.in/op/go-logging.v1"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ES_INDEXER_QUEUE_LEN = 1024 * 1024
	ES_INDEXER_WORKERS   = 8
	ES_WRITES_PER_SECOND = 200
)

/****************** Interfaces *********************/
type ElasticIndexer struct {
	db        *dbs.ElasticSearch
	conn      *elastic.Client
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
	tagIdCache *TagCache
	indexCache *IndexReadCache

	log *logging.Logger
}

func NewElasticIndexer() *ElasticIndexer {
	my := new(ElasticIndexer)
	my.log = logging.MustGetLogger("indexer.elastic")
	my.indexCache = NewIndexCache(10000)
	my.tagIdCache = NewTagCache()
	return my
}

func (my *ElasticIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` host:9200/index_name is needed for elasticsearch config")
	}

	db, err := dbs.NewDB("elasticsearch", dsn, conf)
	if err != nil {
		return err
	}

	my.db = db.(*dbs.ElasticSearch)
	my.conn = my.db.Client

	// tweak queues and worker sizes
	my.num_workers = int(conf.Int64("write_workers", ES_INDEXER_WORKERS))
	my.queue_len = int(conf.Int64("write_queue_length", ES_INDEXER_QUEUE_LEN))

	// url parse so that the password is not in the name
	parsed, _ := url.Parse(dsn)
	host := parsed.Host + parsed.Path

	my.indexerId = "indexer:elastic:" + host
	my.cache, err = getCacherSingleton(my.indexerId)
	if err != nil {
		return err
	}

	my.writes_per_second = int(conf.Int64("writes_per_second", ES_WRITES_PER_SECOND))
	my.cache.maxKeys = int(conf.Int64("cache_index_size", CACHER_METRICS_KEYS))

	atomic.StoreUint32(&my.shutitdown, 1)
	my.shutdown = make(chan bool)

	return nil
}
func (my *ElasticIndexer) Name() string { return my.indexerId }

func (my *ElasticIndexer) Start() {
	my.startstop.Start(func() {
		my.log.Notice("starting elastic indexer: %s/%s/%s", my.db.PathTable(), my.db.TagTable())
		err := NewElasticSearchSchema(my.conn, my.db.SegmentTable(), my.db.PathTable(), my.db.TagTable()).AddIndexTables()
		if err != nil {
			panic(err)
		}
		retries := 2
		my.dispatcher = dispatch.NewDispatchQueue(my.num_workers, my.queue_len, retries)
		my.dispatcher.Start()
		my.cache.Start() //start cacher
		my.shutitdown = 0

		go my.sendToWriters() // the dispatcher
	})
}

func (my *ElasticIndexer) Stop() {
	my.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		my.log.Notice("shutting down elasticsearcg indexer: %s/%s", my.db.PathTable(), my.db.TagTable())
		my.cache.Stop()
		my.shutitdown = 1
	})
}

// this fills up the tag cache on startup
func (my *ElasticIndexer) fillTagCache() {
	//ToDo

}

// pop from the cache and send to actual writers
func (my *ElasticIndexer) sendToWriters() error {
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
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.elastic.write.send-to-writers"), 1)
				my.dispatcher.Add(&ElasticIndexerJob{ES: my, Stat: skey})
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
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.elastic.write.send-to-writers"), 1)
				my.dispatcher.Add(&ElasticIndexerJob{ES: my, Stat: skey})
				time.Sleep(dur)
			}
		}
	}
}

// keep an index of the stat keys and their fragments so we can look up
func (my *ElasticIndexer) Write(skey repr.StatName) error {
	return my.cache.Add(skey)
}

func (my *ElasticIndexer) WriteTags(inname *repr.StatName, do_main bool, do_meta bool) error {

	have_meta := !inname.MetaTags.IsEmpty()
	have_tgs := !inname.Tags.IsEmpty()
	if !have_tgs && !have_meta || (!do_main && !do_meta) {
		return nil
	}

	if have_tgs && do_main {
		for _, t := range inname.Tags {
			_, got := my.inTagCache(t[0], t[1])
			if got {
				continue
			}
			tg := new(ESTag)
			tg.Name = t[0]
			tg.Value = t[1]
			tg.IsMeta = false
			tg_sum := md5.New()
			tg_sum.Write([]byte(fmt.Sprintf("%s:%s:%v", t[0], t[1], false)))
			ret, err := my.conn.Index().
				Index(my.db.TagTable()).
				Type(my.db.TagType).
				Id(hex.EncodeToString(tg_sum.Sum(nil))).
				BodyJson(tg).
				Do()
			if err != nil {
				my.log.Error("Could not insert tag %v (%v) :: %v", t[0], t[1], err)
				continue
			}
			if len(ret.Id) > 0 {
				my.tagIdCache.Add(t[0], t[1], false, ret.Id)
			}
		}
	}
	if have_meta && do_meta {
		for _, t := range inname.Tags {
			_, got := my.inTagCache(t[0], t[1])
			if got {
				continue
			}
			tg := new(ESTag)
			tg.Name = t[0]
			tg.Value = t[1]
			tg.IsMeta = true
			tg_sum := md5.New()
			tg_sum.Write([]byte(fmt.Sprintf("%s:%s:%v", t[0], t[1], true)))
			_, err := my.conn.Index().
				Index(my.db.TagTable()).
				Type(my.db.TagType).
				Id(hex.EncodeToString(tg_sum.Sum(nil))).
				BodyJson(tg).
				Do()
			if err != nil {
				my.log.Error("Could not insert tag %v (%v) :: %v", t[0], t[1], err)
			}
		}
	}
	return nil
}

// a basic clone of the cassandra indexer
func (my *ElasticIndexer) WriteOne(inname *repr.StatName) error {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("indexer.elastic.write.path-time-ns"), time.Now())
	stats.StatsdClientSlow.Incr("indexer.elastic.noncached-writes-path", 1)

	skey := inname.Key
	unique_ID := inname.UniqueIdString()

	// we are going to assume that if the path is already in the system, we've indexed it and therefore
	// do not need to do the super loop (which is very expensive)
	got_already, err := my.conn.Get().Index(my.db.PathTable()).Id(unique_ID).Do()
	if err == nil && got_already != nil && got_already.Found {
		return my.WriteTags(inname, false, true)
	}

	pth := NewParsedPath(skey, unique_ID)

	// do the segments
	last_path := pth.Last()

	bulk := my.conn.Bulk()

	// now to upsert them all (inserts in cass are upserts)
	for idx, seg := range pth.Segments {

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
		es_obj := new(ESPath)
		es_obj.Segment = seg.Segment
		es_obj.Pos = seg.Pos
		es_obj.HasData = false
		es_obj.Length = seg.Pos + 1

		es_seg := new(ESSegment)
		es_seg.Segment = seg.Segment
		es_seg.Pos = seg.Pos

		// we set the "_id" to the md5 of the path to avoid dupes
		seg_md := md5.New()
		seg_md.Write([]byte(es_obj.Segment))
		bulk.Add(elastic.NewBulkIndexRequest().
			Index(my.db.SegmentTable()).
			Type(my.db.SegmentType).
			Id(hex.EncodeToString(seg_md.Sum(nil))).
			Doc(es_seg))

		if skey != seg.Segment && idx < pth.Len-1 {
			es_obj.Path = seg.Segment + "." + pth.Parts[idx+1]

			// we set the "_id" to the md5 of the path to avoid dupes
			pth_md := md5.New()
			pth_md.Write([]byte(es_obj.Path))
			bulk.Add(elastic.NewBulkIndexRequest().
				Index(my.db.PathTable()).
				Type(my.db.PathType).
				Id(hex.EncodeToString(pth_md.Sum(nil))).
				Doc(es_obj))

		} else {
			// full path object
			es_obj.Path = skey
			es_obj.Uid = unique_ID
			es_obj.Tags = make([]ESTag, 0)
			es_obj.HasData = true

			if !inname.Tags.IsEmpty() {
				for _, tg := range inname.Tags {
					es_obj.Tags = append(es_obj.Tags, ESTag{
						Name:   tg[0],
						Value:  tg[1],
						IsMeta: false,
					})
				}
			}
			if !inname.MetaTags.IsEmpty() {
				for _, tg := range inname.MetaTags {
					es_obj.Tags = append(es_obj.Tags, ESTag{
						Name:   tg[0],
						Value:  tg[1],
						IsMeta: true,
					})
				}
			}
			bulk.Add(elastic.NewBulkIndexRequest().
				Index(my.db.PathTable()).
				Type(my.db.PathType).
				Id(unique_ID).
				Doc(es_obj))
		}

	}

	_, err = bulk.Do()

	if err != nil {
		my.log.Error("Could not insert path %v (%v) :: %v", last_path, unique_ID, err)
		stats.StatsdClientSlow.Incr("indexer.elastic.path-failures", 1)
	} else {
		stats.StatsdClientSlow.Incr("indexer.elastic.path-writes", 1)
	}

	err = my.WriteTags(inname, true, true)
	if err != nil {
		my.log.Error("Could not write tag index", err)
		return err
	}

	return err
}

func (my *ElasticIndexer) Delete(name *repr.StatName) error {

	uid := name.UniqueIdString()
	_, err := my.conn.Delete().Index(my.db.PathTable()).Type(my.db.PathType).Id(uid).Do()
	if err != nil {
		my.log.Error("Elastic Driver: Delete Path failed, %v", err)
		return err
	}

	return nil
}

/**** READER ***/

// Expand simply pulls out any regexes into full form
func (my *ElasticIndexer) Expand(metric string) (MetricExpandItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.expand.get-time-ns", time.Now())

	m_len := len(strings.Split(metric, ".")) - 1
	base_q := my.conn.Search().Index(my.db.SegmentTable()).Type(my.db.SegmentType)

	need_reg := needRegex(metric)
	and_filter := elastic.NewBoolQuery()
	if need_reg {
		and_filter = and_filter.Must(elastic.NewTermQuery("pos", m_len))
		and_filter = and_filter.Must(elastic.NewRegexpQuery("segment", regifyKeyString(metric)))
		agg := elastic.NewTermsAggregation().Field("segment")
		base_q = base_q.Aggregation("seg_count", agg)
	} else {
		and_filter = and_filter.Must(elastic.NewTermQuery("pos", m_len))
		and_filter = and_filter.Must(elastic.NewTermQuery("segment", metric))
		base_q = base_q.From(0).Size(1)
	}
	var me MetricExpandItem

	es_items, err := base_q.Query(and_filter).Sort("segment", true).Do()

	if err != nil {
		return me, err
	}

	// if we get a grouped item, we know we're on segment not data lands
	terms, ok := es_items.Aggregations.Terms("seg_count")
	if ok && len(terms.Buckets) < len(es_items.Hits.Hits) {
		for _, h := range terms.Buckets {
			str := h.Key.(string)
			me.Results = append(me.Results, str)

		}
	} else {

		for _, h := range es_items.Hits.Hits {
			// just grab the "n+1" length ones
			var item ESSegment
			err := json.Unmarshal(*h.Source, &item)
			if err != nil {
				my.log.Error("Elastic Driver: json error, %v", err)
				continue
			}
			me.Results = append(me.Results, item.Segment)
		}
	}

	if err != nil {
		return me, err
	}

	return me, nil

}

/******************* TAG METHODS **********************************/
func (my *ElasticIndexer) inTagCache(name string, value string) (tag_id string, ismeta bool) {

	got := my.tagIdCache.Get(name, value, false)

	if got != nil {
		return got.(string), false
	}
	got = my.tagIdCache.Get(name, value, true)
	if got != nil {
		return got.(string), true
	}

	return "", false
}

func (my *ElasticIndexer) FindTagId(name string, value string, ismeta bool) (string, error) {

	// see if in the writer tag cache
	c_id, c_meta := my.inTagCache(name, value)
	if ismeta == c_meta && c_id == "" {
		return c_id, nil
	}
	and_filter := elastic.NewBoolQuery()
	and_filter = and_filter.Must(elastic.NewTermQuery("name", name))
	and_filter = and_filter.Must(elastic.NewTermQuery("value", value))
	and_filter = and_filter.Must(elastic.NewTermQuery("is_meta", ismeta))

	items, err := my.conn.Search().Index(my.db.PathTable()).Type(my.db.PathType).Query(and_filter).From(0).Size(1).Do()
	if err != nil {
		my.log.Error("Elastic Driver: Tag find error, %v", err)
		return "", err

	}
	// just the first one
	for _, h := range items.Hits.Hits {
		return h.Id, nil
	}
	return "", err
}

func (my *ElasticIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {

	base_q := my.conn.Search().Index(my.db.PathTable()).Type(my.db.PathType)
	items, err := base_q.Query(elastic.NewTermQuery("uid", unique_id)).Do()
	if err != nil {
		return tags, metatags, err
	}

	for _, h := range items.Hits.Hits {
		var tg ESPath
		err := json.Unmarshal(*h.Source, &tg)

		if err != nil {
			my.log.Error("Error Getting Tags Iterator : %v", err)
			continue
		}
		tags, metatags := tg.ToSortedTags()
		return tags, metatags, nil
	}

	return tags, metatags, err
}

func (my *ElasticIndexer) GetTagsByName(name string, page int) (tags MetricTagItems, err error) {

	var items *elastic.SearchResult
	base_q := my.conn.Search().Index(my.db.TagTable()).Type(my.db.TagType)

	if needRegex(name) {
		use_name := regifyKeyString(name)
		n_q := elastic.NewRegexpQuery("name", use_name)
		base_q = base_q.Query(n_q)
	} else {

		n_q := elastic.NewTermQuery("name", name)
		base_q = base_q.Query(n_q)
	}
	items, err = base_q.From(page * MAX_PER_PAGE).Size(MAX_PER_PAGE).Do()

	if err != nil {
		my.log.Error("Elastic Driver: Tag find error, %v", err)
		return tags, err
	}

	// just the first one
	for _, h := range items.Hits.Hits {

		var tg ESTag
		err = json.Unmarshal(*h.Source, &tg)

		if err != nil {
			my.log.Error("Error Getting Tags Iterator : %v", err)
			continue
		}
		tags = append(tags, MetricTagItem{Name: tg.Name, Value: tg.Value, Id: h.Id, IsMeta: tg.IsMeta})
		my.tagIdCache.Add(tg.Name, tg.Value, tg.IsMeta, h.Id)
	}
	return
}

func (my *ElasticIndexer) GetTagsByNameValue(name string, value string, page int) (tags MetricTagItems, err error) {
	var items *elastic.SearchResult

	base_q := my.conn.Search().Index(my.db.TagTable()).Type(my.db.TagType)
	and_filter := elastic.NewBoolQuery()

	if needRegex(name) {
		and_filter = and_filter.Must(elastic.NewRegexpQuery("name", regifyKeyString(name)))
	} else {
		and_filter = and_filter.Must(elastic.NewTermQuery("name", name))
	}
	if needRegex(value) {
		and_filter = and_filter.Must(elastic.NewRegexpQuery("value", regifyKeyString(value)))
	} else {
		and_filter = and_filter.Must(elastic.NewTermQuery("value", value))
	}

	items, err = base_q.Query(and_filter).From(page * MAX_PER_PAGE).Size(MAX_PER_PAGE).Do()
	if err != nil {
		my.log.Error("Elastic Driver: Tag find error, %v", err)
		return tags, err
	}

	for _, h := range items.Hits.Hits {

		var tg ESTag
		err = json.Unmarshal(*h.Source, &tg)

		if err != nil {
			my.log.Error("Error Getting Tags Iterator : %v", err)
			continue
		}
		tags = append(tags, MetricTagItem{Name: tg.Name, Value: tg.Value, Id: h.Id, IsMeta: tg.IsMeta})
		my.tagIdCache.Add(tg.Name, tg.Value, tg.IsMeta, h.Id)
	}

	return
}

func (my *ElasticIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	//TODO
	return
}

/********************* UID metric finders ***********************/

// List all paths w/ data
func (my *ElasticIndexer) List(has_data bool, page int) (MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.list.get-time-ns", time.Now())

	var mt MetricFindItems
	var ms MetricFindItem

	filter := elastic.NewTermQuery("has_data", true)
	items, err := my.conn.Search().Index(my.db.PathTable()).Type(my.db.PathType).Query(filter).Sort("path", true).From(page * MAX_PER_PAGE).Size(MAX_PER_PAGE).Do()

	if err != nil {
		return mt, err
	}
	for _, h := range items.Hits.Hits {

		var tg ESPath
		err := json.Unmarshal(*h.Source, &tg)

		if err != nil {
			my.log.Error("Error Getting Tags Iterator : %v", err)
			continue
		}
		spl := strings.Split(tg.Path, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = tg.Path
		ms.Path = tg.Path

		ms.Expandable = 0
		ms.Leaf = 1
		ms.AllowChildren = 0
		ms.UniqueId = tg.Uid
		ms.Tags, ms.MetaTags = tg.ToSortedTags()

		mt = append(mt, ms)
	}

	return mt, nil
}

// basic find for non-regex items
func (my *ElasticIndexer) FindBase(metric string, tags repr.SortingTags, exact bool) (MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.findbase.get-time-ns", time.Now())

	// check cache
	items := my.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.elastic.findbase.cached", 1)
		return *items, nil
	}

	base_q := my.conn.Search().Index(my.db.PathTable()).Type(my.db.PathType)
	and_filter := elastic.NewBoolQuery()
	var all_tag_filter *elastic.BoolQuery
	var mt MetricFindItems

	// if "tags" we need to find the tag Ids, and do the cross join
	for _, tag := range tags {
		t_ids, _ := my.GetTagsByNameValue(tag[0], tag[1], 0)
		if len(t_ids) > 0 {
			tag_filter := elastic.NewBoolQuery()
			for _, tg := range t_ids {
				tag_filter = tag_filter.Must(elastic.NewTermQuery("tags.name", tg.Name))
				tag_filter = tag_filter.Must(elastic.NewTermQuery("tags.value", tg.Value))
				tag_filter = tag_filter.Must(elastic.NewTermQuery("tags.is_meta", tg.IsMeta))
			}
			nest_filter := elastic.NewNestedQuery("tags", tag_filter)
			if all_tag_filter == nil {
				all_tag_filter = elastic.NewBoolQuery()
			}
			all_tag_filter = all_tag_filter.Must(nest_filter)
		} else {
			// cannot have results now can we
			return mt, nil
		}
	}

	var ms MetricFindItem
	need_reg := needRegex(metric)
	if len(metric) > 0 {
		if need_reg {
			and_filter = and_filter.Must(elastic.NewTermQuery("pos", len(strings.Split(metric, "."))-1))
			and_filter = and_filter.Must(elastic.NewRegexpQuery("segment", regifyKeyString(metric)))
			agg := elastic.NewTermsAggregation().Field("segment")
			base_q = base_q.Aggregation("seg_count", agg)
		} else {
			and_filter = and_filter.Must(elastic.NewTermQuery("pos", len(strings.Split(metric, "."))-1))
			and_filter = and_filter.Must(elastic.NewTermQuery("segment", metric))
			base_q = base_q.From(0).Size(1)
		}
	}
	if all_tag_filter != nil {
		and_filter = and_filter.Must(all_tag_filter)
	}
	es_items, err := base_q.Query(and_filter).Sort("segment", true).Do()

	if err != nil {
		return mt, err
	}

	// if we get a grouped item, we know we're on segment not data lands
	terms, ok := es_items.Aggregations.Terms("seg_count")
	if ok && len(terms.Buckets) < len(es_items.Hits.Hits) {
		for _, h := range terms.Buckets {
			str := h.Key.(string)
			spl := strings.Split(str, ".")

			ms.Text = spl[len(spl)-1]
			ms.Id = str
			ms.Path = str
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
			mt = append(mt, ms)
		}
	} else {

		for _, h := range es_items.Hits.Hits {

			var tg ESPath
			err := json.Unmarshal(*h.Source, &tg)

			if err != nil {
				my.log.Error("Error in json: %v", err)
				continue
			}
			spl := strings.Split(tg.Segment, ".")

			ms.Text = spl[len(spl)-1]
			ms.Id = tg.Segment
			ms.Path = tg.Segment
			if tg.HasData {
				ms.Expandable = 0
				ms.Leaf = 1
				ms.AllowChildren = 0
				ms.UniqueId = tg.Uid
				ms.Tags, ms.MetaTags = tg.ToSortedTags()
			} else {
				ms.Expandable = 1
				ms.Leaf = 0
				ms.AllowChildren = 1
			}
			mt = append(mt, ms)
		}
	}

	// set it
	my.indexCache.Add(metric, tags, &mt)

	return mt, nil
}

// special case for "root" == "*" finder
func (my *ElasticIndexer) FindRoot(tags repr.SortingTags) (MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.findroot.get-time-ns", time.Now())

	var mt MetricFindItems

	base_q := my.conn.Search().Index(my.db.SegmentTable()).Type(my.db.SegmentType)
	and_filter := elastic.NewBoolQuery()
	and_filter = and_filter.Must(elastic.NewTermQuery("pos", 0))
	var all_tag_filter *elastic.BoolQuery

	// if "tags" we need to find the tag Ids, and do the cross join
	for _, tag := range tags {
		t_ids, _ := my.GetTagsByNameValue(tag[0], tag[1], 0)
		if len(t_ids) > 0 {
			tag_filter := elastic.NewBoolQuery()
			for _, tg := range t_ids {
				tag_filter = tag_filter.Must(elastic.NewTermQuery("tags.name", tg.Name))
				tag_filter = tag_filter.Must(elastic.NewTermQuery("tags.value", tg.Value))
				tag_filter = tag_filter.Must(elastic.NewTermQuery("tags.is_meta", tg.IsMeta))
			}
			nest_filter := elastic.NewNestedQuery("tags", tag_filter)
			if all_tag_filter == nil {
				all_tag_filter = elastic.NewBoolQuery()
			}
			all_tag_filter = all_tag_filter.Must(nest_filter)
		} else {
			// cannot have results now can we
			return mt, nil
		}
	}

	if all_tag_filter != nil {
		and_filter = and_filter.Must(all_tag_filter)
	}
	es_items, err := base_q.Query(and_filter).Sort("segment", true).Do()

	if err != nil {
		return mt, err
	}
	for _, h := range es_items.Hits.Hits {

		var tg ESSegment
		var ms MetricFindItem

		err := json.Unmarshal(*h.Source, &tg)

		if err != nil {
			my.log.Error("Error in json: %v", err)
			continue
		}
		ms.Text = tg.Segment
		ms.Id = tg.Segment
		ms.Path = tg.Segment
		ms.Expandable = 1
		ms.Leaf = 0
		ms.AllowChildren = 1

		mt = append(mt, ms)
	}

	return mt, nil
}

// to allow for multiple targets
func (my *ElasticIndexer) Find(metric string, tags repr.SortingTags) (MetricFindItems, error) {
	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide regex abilities
	// on the server side

	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.find.get-time-ns", time.Now())

	// special case for "root" == "*"

	// check cache
	items := my.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.elastic.find.cached", 1)
		return *items, nil
	}

	if metric == "*" {
		return my.FindRoot(tags)
	}

	mt, err := my.FindBase(metric, tags, true)
	if err != nil {
		return mt, err
	}
	// set it
	my.indexCache.Add(metric, tags, &mt)

	return mt, nil
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type ElasticIndexerJob struct {
	ES    *ElasticIndexer
	Stat  repr.StatName
	retry int
}

func (j *ElasticIndexerJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j *ElasticIndexerJob) OnRetry() int {
	return j.retry
}

func (j *ElasticIndexerJob) DoWork() error {
	err := j.ES.WriteOne(&j.Stat)
	if err != nil {
		j.ES.log.Error("Insert failed for Index: %v retrying ...", j.Stat)
	}
	return err
}
