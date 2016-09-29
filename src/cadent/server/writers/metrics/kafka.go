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
	THe Kafka Metrics "blob" metrics writer

	this only emits on the "overflow" from the cacher and a big binary blobl of data


*/

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/repr"
	"cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/schemas"
	"fmt"
	"github.com/Shopify/sarama"
	logging "gopkg.in/op/go-logging.v1"
	"math"
	"strings"
	"sync"
	"time"
)

/****************** Interfaces *********************/
type KafkaMetrics struct {
	WriterBase

	db   *dbs.KafkaDB
	conn sarama.AsyncProducer

	batches int // number of stats to "batch" per message (default 0)
	log     *logging.Logger

	resolution uint32

	enctype schemas.SendEncoding

	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

}

func NewKafkaMetrics() *KafkaMetrics {
	kf := new(KafkaMetrics)
	kf.batches = 0
	kf.log = logging.MustGetLogger("writers.kafka.metrics")

	kf.shutitdown = false
	kf.is_primary = false
	return kf
}

func (kf *KafkaMetrics) Config(conf map[string]interface{}) error {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (kafkahost1,kafkahost2) is needed for kafka config")
	}
	dsn := gots.(string)
	db, err := dbs.NewDB("kafka", dsn, conf)
	if err != nil {
		return err
	}

	resolution := conf["resolution"]
	if resolution == nil {
		return fmt.Errorf("Resolution needed for kafka blob writer")
	} else {
		kf.resolution = uint32(resolution.(float64))
	}

	kf.db = db.(*dbs.KafkaDB)
	kf.conn = db.Connection().(sarama.AsyncProducer)

	g_tag, ok := conf["tags"]
	if ok {
		kf.static_tags = repr.SortingTagsFromString(g_tag.(string))
	}

	enct, ok := conf["encoding"]
	if ok {
		kf.enctype = schemas.SendEncodingFromString(enct.(string))
	}

	_cache := conf["cache"]
	if _cache == nil {
		return errMetricsCacheRequired
	}
	kf.cacher = _cache.(*Cacher)
	kf.cacherPrefix = kf.cacher.Prefix

	// for the overflow cached items::
	// these caches can be shared for a given writer set, and the caches may provide the data for
	// multiple writers, we need to specify that ONE of the writers is the "main" one otherwise
	// the Metrics Write function will add the points over again, which is not a good thing
	// when the accumulator flushes things to the multi wrtiers
	// The Writer needs to know it's "not" the primary writer and thus will not "add" points to the
	// cache .. so the cache basically gets "one" primary writer pointed (first come first serve)
	kf.is_primary = kf.cacher.SetPrimaryWriter(kf)
	if kf.is_primary {
		kf.log.Notice("Kafka series writer is the primary writer to write back cache %s", kf.cacher.Name)
	}
	return nil
}

func (kf *KafkaMetrics) Driver() string {
	return "kafka"
}

func (kf *KafkaMetrics) Start() {
	kf.startstop.Start(func() {
		kf.log.Notice("Starting Kafka writer for %s at %d bytes per series", kf.db.DataTopic(), kf.cacher.maxBytes)
		kf.cacher.overFlowMethod = "chan" // force chan

		kf.cacher.Start()
		// only register this if we are really going to consume it
		kf.cacheOverFlow = kf.cacher.GetOverFlowChan()
		go kf.overFlowWrite()
		go kf.onError()
		go kf.onSuccess()
	})
}

func (kf *KafkaMetrics) Stop() {
	kf.log.Warning("Stopping Kafka writer for (%s)", kf.cacher.Name)
	kf.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		if kf.shutitdown {
			return // already did
		}
		kf.shutitdown = true
		kf.cacher.Stop()

		mets := kf.cacher.Cache
		mets_l := len(mets)
		kf.log.Warning("Shutting down %s and exhausting the queue (%d items) and quiting", kf.cacher.Name, mets_l)

		// full tilt write out
		procs := 16
		go_do := make(chan TotalTimeSeries, procs)
		wg := sync.WaitGroup{}

		goInsert := func() {
			for {
				select {
				case s, more := <-go_do:
					if !more {
						return
					}
					stats.StatsdClient.Incr(fmt.Sprintf("writer.cache.shutdown.send-to-writers"), 1)
					kf.PushSeries(s.Name, s.Series)
					wg.Done()
				}
			}
		}
		for i := 0; i < procs; i++ {
			go goInsert()
		}

		did := 0
		for _, queueitem := range mets {
			wg.Add(1)
			if did%100 == 0 {
				kf.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
			}
			if queueitem.Series != nil {
				go_do <- TotalTimeSeries{Name: queueitem.Name, Series: queueitem.Series.Copy()}
			}
			did++
		}
		wg.Wait()
		close(go_do)
		kf.log.Warning("shutdown purge: written %d/%d...", did, mets_l)

		kf.log.Warning("Shutdown finished ... quiting kafka writer")
		return
	})
}

func (kf *KafkaMetrics) onError() {

	for {
		err, more := <-kf.conn.Errors()
		if !more {
			return
		}
		stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes.error", 1)
		kf.log.Errorf("%s", err)
	}
}

func (kf *KafkaMetrics) onSuccess() {

	for {
		_, more := <-kf.conn.Successes()
		if !more {
			return
		}
		stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes.success", 1)
	}
}

// listen to the overflow chan from the cache and attempt to write "now"
func (kf *KafkaMetrics) overFlowWrite() {
	defer func() {
		if r := recover(); r != nil {
			kf.log.Critical("Kafka Failure (panic) %v", r)
		}
	}()

	for {
		statitem, more := <-kf.cacheOverFlow.Ch
		if !more {
			return
		}
		if statitem != nil {
			ts := statitem.(*TotalTimeSeries)
			kf.PushSeries(ts.Name, ts.Series)
		} else {
			kf.log.Errorf("%s", schemas.ErrMetricIsNil)
		}
	}
}

// based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
// from and to needs to be > then the TTL as well
func (kf *KafkaMetrics) getResolution(from int64, to int64) uint32 {
	diff := int(math.Abs(float64(to - from)))
	n := int(time.Now().Unix())
	back_f := n - int(from)
	back_t := n - int(to)
	for _, res := range kf.resolutions {
		if diff < res[1] && back_f < res[1] && back_t < res[1] {
			return uint32(res[0])
		}
	}
	return uint32(kf.resolutions[len(kf.resolutions)-1][0])
}

func (kf *KafkaMetrics) Write(stat repr.StatRepr) error {
	// merge the tags in
	stat.Name.MergeMetric2Tags(kf.static_tags)
	kf.indexer.Write(stat.Name) // to the indexer
	// not primary writer .. move along
	if !kf.is_primary {
		return nil
	}
	kf.cacher.Add(&stat.Name, &stat)
	return nil
}

func (kf *KafkaMetrics) PushSeries(name *repr.StatName, points series.TimeSeries) error {
	if name == nil {
		return errNameIsNil
	}
	if points == nil {
		return errSeriesIsNil
	}

	pts, err := points.MarshalBinary()
	if err != nil {
		return err
	}

	s_metric := new(schemas.SeriesMetric)
	s_metric.Metric = name.Key
	s_metric.Time = time.Now().UnixNano()
	s_metric.Data = pts
	s_metric.Encoding = points.Name()
	s_metric.Resolution = name.Resolution
	s_metric.Ttl = name.TTL
	s_metric.Tags = schemas.ToMetricTag(name.SortedTags())
	s_metric.MetaTags = schemas.ToMetricTag(name.SortedMetaTags())

	obj := &schemas.KMetric{
		AnyMetric: schemas.AnyMetric{
			Series: s_metric,
		},
	}
	obj.SetSendEncoding(kf.enctype)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.DataTopic(),
		Key:   sarama.StringEncoder(obj.Id()),
		Value: obj,
	}

	stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes", 1)
	return nil
}

/**** READER ***/

// we can impliment this here as we have timeseries in RAM
// it's different the other ones as there is no "index" here really to speak of
func (kf *KafkaMetrics) GetFromWriteCache(metric *repr.StatName, start uint32, end uint32, resolution uint32) (*RawRenderItem, error) {

	// grab data from the write inflight cache
	// need to pick the "proper" cache
	cache_db := fmt.Sprintf("%s:%v", kf.cacherPrefix, resolution)
	use_cache := GetCacherByName(cache_db)
	if use_cache == nil {
		use_cache = kf.cacher
	}
	inflight, err := use_cache.GetAsRawRenderItem(metric)
	if err != nil {
		return nil, err
	}
	if inflight == nil {
		return nil, nil
	}
	inflight.Metric = metric.Key
	inflight.Id = metric.UniqueIdString()
	inflight.Step = resolution
	inflight.Start = start
	inflight.End = end
	inflight.Tags = metric.Tags
	inflight.MetaTags = metric.MetaTags
	return inflight, nil
}

// needed to match interface, but we obviously cannot do this

func (kf *KafkaMetrics) RawRender(path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*RawRenderItem, error) {
	return []*RawRenderItem{}, errKafkaReaderNotImplimented
}

func (kf *KafkaMetrics) CacheRender(path string, start int64, end int64, tags repr.SortingTags) (rawd []*RawRenderItem, err error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.cacherender.get-time-ns", time.Now())

	//figure out the best res
	resolution := kf.getResolution(start, end)

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	paths := strings.Split(path, ",")
	var metrics []*repr.StatName

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	for _, pth := range paths {
		nm := &repr.StatName{
			Key: pth,
		}
		// need to merge in static tags to get UniqueIDs proper
		nm.MergeMetric2Tags(kf.static_tags)
		nm.MergeMetric2Tags(tags)
		metrics = append(metrics, nm)
	}

	rawd = make([]*RawRenderItem, len(metrics), len(metrics))

	// ye old fan out technique
	render_one := func(metric *repr.StatName, idx int) {
		defer render_wg.Done()
		_ri, err := kf.GetFromWriteCache(metric, uint32(start), uint32(end), resolution)

		if err != nil {
			kf.log.Error("Read Error for %s (%d->%d) : %v", path, start, end, err)
			return
		}
		rawd[idx] = _ri
		return

	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(metric, idx)
	}
	render_wg.Wait()
	return rawd, nil
}

func (kf *KafkaMetrics) CachedSeries(path string, start int64, end int64, tags repr.SortingTags) (*TotalTimeSeries, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.seriesrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	if len(paths) > 1 {
		return nil, errMultiTargetsNotAllowed
	}

	metric := &repr.StatName{Key: path}
	metric.MergeMetric2Tags(tags)
	metric.MergeMetric2Tags(kf.static_tags)

	resolution := kf.getResolution(start, end)
	cache_db := fmt.Sprintf("%s:%v", kf.cacherPrefix, resolution)
	use_cache := GetCacherByName(cache_db)
	if use_cache == nil {
		use_cache = kf.cacher
	}
	name, inflight, err := use_cache.GetSeries(metric)
	if err != nil {
		return nil, err
	}
	if inflight == nil {
		// try the the path as unique ID
		gots_int := metric.StringToUniqueId(path)
		if gots_int != 0 {
			name, inflight, err = use_cache.GetSeriesById(gots_int)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, nil
		}
	}

	return &TotalTimeSeries{Name: name, Series: inflight}, nil
}
