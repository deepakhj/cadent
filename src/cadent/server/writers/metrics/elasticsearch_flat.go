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
	The ElasticSearch flat stat write


	OPTIONS: For `Config`

		metric_index="metrics" #  base table name (default: metrics)
		batch_count=1000 # batch this many inserts for much faster insert performance (default 1000)
		periodic_flush="1s" # regardless of if batch_count met always flush things at this interval (default 1s)

		## create a new index based on the time of the point
		# of the form "{metrics}_{resolution}s-{YYYY-MM-DD}"
		# default is "none"
		index_per_date = "none|week|month|day"


*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"encoding/json"
	"fmt"
	"gopkg.in/olivere/elastic.v3"
	logging "gopkg.in/op/go-logging.v1"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	ELASTIC_DEFAULT_METRIC_RENDER_WORKERS = 4
	ELASTIC_RENDER_TIMEOUT                = "5s"
	ELASTIC_FLUSH_TIME_SECONDS            = 1
)

var esFlatsourceList = []string{"time", "min", "max", "last", "count", "sum"}

// ESmetric pool

var esMetricPool sync.Pool

func getESMetric() *ESMetric {
	x := esMetricPool.Get()
	if x == nil {
		return new(ESMetric)
	}
	return x.(*ESMetric)
}

func putESMetric(spl *ESMetric) {
	esMetricPool.Put(spl)
}

/****************** Interfaces *********************/
type ElasticSearchFlatMetrics struct {
	WriterBase

	db   *dbs.ElasticSearch
	conn *elastic.Client

	writeList     []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	maxWriteSize  int             // size of that buffer before a flush
	maxIdle       time.Duration   // either maxWriteSize will trigger a write or this time passing will
	writeLock     sync.Mutex
	renderTimeout time.Duration

	indexPerDate string

	log *logging.Logger

	shutdown chan bool
}

func NewElasticSearchFlatMetrics() *ElasticSearchFlatMetrics {
	es := new(ElasticSearchFlatMetrics)
	es.log = logging.MustGetLogger("writers.elasticflat")
	return es
}

func (es *ElasticSearchFlatMetrics) Config(conf *options.Options) error {

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` host:9200/index_name is needed for elasticsearch config")
	}

	dbKey := dsn + conf.String("table", "metrics")

	db, err := dbs.NewDB("elasticsearch", dbKey, conf)
	if err != nil {
		return err
	}

	es.db = db.(*dbs.ElasticSearch)
	es.conn = es.db.Client

	res, err := conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resolution needed for elasticsearch writer")
	}

	//need to hide the usr/pw from things
	p, _ := url.Parse(dsn)
	cacheKey := fmt.Sprintf("elasticflat:cache:%s/%s:%v", p.Host, conf.String("table", "metrics"), res)
	es.cacher, err = GetCacherSingleton(cacheKey, "single")

	if err != nil {
		return err
	}

	es.maxWriteSize = int(conf.Int64("batch_count", 1000))
	es.maxIdle = conf.Duration("periodic_flush", time.Duration(ELASTIC_FLUSH_TIME_SECONDS*time.Second))
	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		es.staticTags = repr.SortingTagsFromString(_tgs)
	}

	es.shutdown = make(chan bool)

	rdur, err := time.ParseDuration(ELASTIC_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	es.renderTimeout = rdur

	es.indexPerDate = conf.String("index_per_date", "none")

	return nil
}

func (es *ElasticSearchFlatMetrics) Driver() string {
	return "elasticsearch-flat"
}

func (es *ElasticSearchFlatMetrics) Stop() {
	es.startstop.Stop(func() {
		shutdown.AddToShutdown()
		es.shutdown <- true
	})
	return
}

func (es *ElasticSearchFlatMetrics) Start() {
	es.startstop.Start(func() {
		// now we make sure the metrics schemas are added
		err := NewElasticMetricsSchema(es.conn, es.db.Tablename(), es.resolutions, "flat").AddMetricsTable()
		if err != nil {
			panic(err)
		}
		es.cacher.Start()
		go es.periodFlush()
	})
}

func (es *ElasticSearchFlatMetrics) periodFlush() {
	for {
		select {
		case <-es.shutdown:
			es.log.Notice("Got shutdown, doing final flush...")
			es.flush()
			shutdown.ReleaseFromShutdown()
			return
		default:
			time.Sleep(es.maxIdle)
			es.flush()
		}
	}
}

func (es *ElasticSearchFlatMetrics) indexName(res uint32, time time.Time) string {
	return fmt.Sprintf("%s_%ds", es.db.Tablename(), res)
}

func (es *ElasticSearchFlatMetrics) flush() (int, error) {
	es.writeLock.Lock()
	defer es.writeLock.Unlock()

	l := len(es.writeList)
	if l == 0 {
		return 0, nil
	}

	bulk := es.conn.Bulk()
	idx_index := make(map[string]repr.StatRepr, len(es.writeList))
	for _, stat := range es.writeList {
		es_m := getESMetric()
		defer putESMetric(es_m)

		es_m.Min = stat.Min
		es_m.Max = stat.Max
		es_m.Sum = stat.Sum
		es_m.Last = stat.Last
		es_m.Count = stat.Count
		es_m.Path = stat.Name.Key
		es_m.Uid = stat.Name.UniqueIdString()

		// time we need to convert to millis as that's the biggest "date" ES will support
		es_m.Time = stat.ToTime()

		id := fmt.Sprintf("%s-%d", stat.Name.UniqueIdString(), stat.Time)
		idx_index[id] = stat
		for _, t := range stat.Name.Tags {
			es_m.Tags = append(es_m.Tags, indexer.ESTag{
				Name:   t.Name,
				Value:  t.Value,
				IsMeta: false,
			})
		}
		for _, t := range stat.Name.MetaTags {
			es_m.Tags = append(es_m.Tags, indexer.ESTag{
				Name:   t.Name,
				Value:  t.Value,
				IsMeta: true,
			})
		}
		bulk.Add(elastic.NewBulkIndexRequest().
			Index(es.indexName(stat.Name.Resolution, stat.ToTime())).
			Type(es.db.MetricType).
			Id(id).
			Doc(es_m))

	}
	erred := make([]repr.StatRepr, 0)
	gots, err := bulk.Do()

	if err != nil {
		es.log.Error("Could not insert metrics %v", err)
		stats.StatsdClientSlow.Incr("writer.elasticflat.bulk-failures", 1)
		return 0, nil
	} else {
		stats.StatsdClientSlow.Incr("writer.elasticflat.bulk-writes", 1)
	}

	// need to check for those that "failed" this is mostly caused by under provisioned ES clusters
	// due to lack of queue/threads
	for _, b := range gots.Items {
		idx := 0
		for _, bb := range b {
			if bb.Error != nil {
				stats.StatsdClientSlow.Incr("writer.elasticflat.insert-one-failures", 1)
				erred_id := bb.Id
				es.log.Error("Could not insert metrics %v id: (%v) ... putting it back into the queue", bb.Error, bb.Id)
				if g, ok := idx_index[erred_id]; ok {
					erred = append(erred, g)
				}
			}
			idx++
		}
	}

	es.writeList = nil
	es.writeList = erred
	return l, nil
}

func (es *ElasticSearchFlatMetrics) Write(stat repr.StatRepr) error {
	stat.Name.MergeMetric2Tags(es.staticTags)

	if len(es.writeList) > es.maxWriteSize {
		es.flush()
	}

	es.indexer.Write(*stat.Name) // to the indexer

	// Flush can cause double locking
	es.writeLock.Lock()
	es.writeList = append(es.writeList, stat)
	es.writeLock.Unlock()
	return nil
}

/**** READER ***/

func (es *ElasticSearchFlatMetrics) RawRenderOne(metric indexer.MetricFindItem, start int64, end int64, resample uint32) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.elasticflat.renderraw.get-time-ns", time.Now())

	rawd := new(RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("RawRenderOne: Not a data node")
	}

	//figure out the best res
	resolution := es.GetResolution(start, end)
	outResolution := resolution

	//obey the bigger
	if resample > resolution {
		outResolution = resample
	}

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	b_len := uint32(end-start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, fmt.Errorf("time too narrow")
	}

	// time in ES is max millis BUT only 13 chars for searches
	// so it's not really  "millis" but 1000ths
	milli := int64(1000)
	qtEnd := end * milli
	qtStart := start * milli

	firstT := uint32(start)
	lastT := uint32(end)

	// try the write inflight cache as nothing is written yet
	stat_name := metric.StatName()
	inflightRenderitem, err := es.cacher.GetAsRawRenderItem(stat_name)

	// need at LEAST 2 points to get the proper step size
	if inflightRenderitem != nil && err == nil {
		// move the times to the "requested" ones and quantize the list
		inflightRenderitem.Metric = metric.Id
		inflightRenderitem.Tags = metric.Tags
		inflightRenderitem.MetaTags = metric.MetaTags
		inflightRenderitem.Id = metric.UniqueId
		inflightRenderitem.AggFunc = stat_name.AggType()
		if inflightRenderitem.Start < uint32(start) {
			inflightRenderitem.RealEnd = uint32(end)
			inflightRenderitem.RealStart = uint32(start)
			inflightRenderitem.Start = inflightRenderitem.RealStart
			inflightRenderitem.End = inflightRenderitem.RealEnd
			return inflightRenderitem, err
		}
	}

	// sorting order for the table is time ASC (i.e. firstT == first entry)
	// on resamples (if >0 ) we simply merge points until we hit the time steps
	do_resample := resample > 0 && resample > resolution

	on_time := time.Unix(start, 0)
	use_index := es.indexName(resolution, on_time)
	baseQ := es.conn.Search().Index(use_index).Type(es.db.MetricType)

	and_filter := elastic.NewBoolQuery()
	and_filter = and_filter.Must(elastic.NewTermQuery("uid", metric.UniqueId))
	and_filter = and_filter.Must(elastic.NewRangeQuery("time").From(qtStart).To(qtEnd).Format("epoch_millis"))
	esItems, err := baseQ.Query(and_filter).Sort("time", true).Do()

	if err != nil {
		ss, _ := elastic.NewSearchSource().Query(and_filter).Sort("time", true).Source()
		data, _ := json.Marshal(ss)
		es.log.Error("Query failed: Index %s: %v (QUERY :: %s)", use_index, err, data)
		return rawd, err
	}

	m_key := metric.Id

	tStart := uint32(start)
	curPt := NullRawDataPoint(tStart)

	for _, h := range esItems.Hits.Hits {
		// just grab the "n+1" length ones
		item := getESMetric()
		defer putESMetric(item)

		err := json.Unmarshal(*h.Source, item)
		if err != nil {
			es.log.Error("Elastic Driver: json error, %v", err)
			continue
		}
		t := uint32(item.Time.Unix())
		if do_resample {
			if t >= tStart+resample {
				tStart += resample
				rawd.Data = append(rawd.Data, curPt)
				curPt = &RawDataPoint{
					Count: item.Count,
					Sum:   item.Sum,
					Max:   item.Max,
					Min:   item.Min,
					Last:  item.Last,
					Time:  t,
				}
			} else {
				curPt.Merge(&RawDataPoint{
					Count: item.Count,
					Sum:   item.Sum,
					Max:   item.Max,
					Min:   item.Min,
					Last:  item.Last,
					Time:  t,
				})
			}
		} else {
			rawd.Data = append(rawd.Data, &RawDataPoint{
				Count: item.Count,
				Sum:   item.Sum,
				Max:   item.Max,
				Min:   item.Min,
				Last:  item.Last,
				Time:  t,
			})
		}
		lastT = t

	}
	if !curPt.IsNull() {
		rawd.Data = append(rawd.Data, curPt)
	}
	if len(rawd.Data) > 0 && rawd.Data[0].Time > 0 {
		firstT = rawd.Data[0].Time
	}

	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, firstT, lastT, len(d_points), ct)

	rawd.RealEnd = uint32(lastT)
	rawd.RealStart = uint32(firstT)
	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	rawd.Step = outResolution
	rawd.Metric = m_key
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.Id = metric.UniqueId
	rawd.AggFunc = stat_name.AggType()

	// grab the "current inflight" from the cache and merge into the main array
	if inflightRenderitem != nil && len(inflightRenderitem.Data) > 1 {
		//merge with any inflight bits (inflight has higher precedence over the file)
		inflightRenderitem.MergeWithResample(rawd, outResolution)
		return inflightRenderitem, nil
	}

	return rawd, nil
}

func (es *ElasticSearchFlatMetrics) RawRender(path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.elasticflat.rawrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	for _, pth := range paths {
		mets, err := es.indexer.Find(pth, tags)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*RawRenderItem, 0, len(metrics))

	procs := ELASTIC_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan indexer.MetricFindItem, len(metrics))
	results := make(chan *RawRenderItem, len(metrics))

	renderOne := func(met indexer.MetricFindItem) *RawRenderItem {
		_ri, err := es.RawRenderOne(met, from, to, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.elasticflat.rawrender.errors", 1)
			es.log.Error("Read Error for %s (%d->%d) : %v", path, from, to, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan indexer.MetricFindItem, resultqueue chan<- *RawRenderItem) {
		rec_chan := make(chan *RawRenderItem, 1)
		for met := range taskqueue {
			go func() { rec_chan <- renderOne(met) }()
			select {
			case <-time.After(es.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.elasticflat.rawrender.timeouts", 1)
				es.log.Errorf("Render Timeout for %s (%d->%d)", path, from, to)
				resultqueue <- nil
			case res := <-rec_chan:
				resultqueue <- res
			}
		}
	}

	for i := 0; i < procs; i++ {
		go jobWorker(i, jobs, results)
	}

	for _, metric := range metrics {
		jobs <- metric
	}
	close(jobs)

	for i := 0; i < len(metrics); i++ {
		res := <-results
		if res != nil {
			rawd = append(rawd, res)
		}
	}
	close(results)
	stats.StatsdClientSlow.Incr("reader.elasticflat.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}

func (es *ElasticSearchFlatMetrics) CacheRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error) {
	return nil, fmt.Errorf("ElasticSearchFlatMetrics: CacheRender: NOT YET IMPLIMNETED")
}
func (es *ElasticSearchFlatMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*TotalTimeSeries, error) {
	return nil, fmt.Errorf("ElasticSearchFlatMetrics: CachedSeries: NOT YET IMPLIMNETED")
}
