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
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"cadent/server/writers/indexer"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	logging "gopkg.in/op/go-logging.v1"
	"time"
)

const (
	KAFKA_DEFAULT_SERIES_TYPE  = "gorilla"
	KAFKA_DEFAULT_SERIES_CHUNK = 16 * 1024 // 16kb
)

/** kafka put object **/
type KafkaMetric struct {
	Type       string      `json:"type"`
	Time       int64       `json:"time"`
	Metric     string      `json:"metric"`
	Encoding   string      `json:"encoding"`
	Data       []byte      `json:"data"`
	Resolution uint32      `json:"resolution"`
	Id         repr.StatId `json:"id"`
	Uid        string      `json:"uid"`
	TTL        uint32      `json:"ttl"`
	Tags       [][]string  `json:"tags,omitempty"` // key1=value1,key2=value2...
	MetaTags   [][]string  `json:"meta_tags,omitempty"`

	encoded []byte
	err     error
}

func (kp *KafkaMetric) ensureEncoded() {
	if kp.encoded == nil && kp.err == nil {
		kp.encoded, kp.err = json.Marshal(kp)
	}
}

func (kp *KafkaMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KafkaMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

/****************** Interfaces *********************/
type KafkaMetrics struct {
	db          *dbs.KafkaDB
	conn        sarama.AsyncProducer
	indexer     indexer.Indexer
	resolutions [][]int
	static_tags repr.SortingTags

	batches int // number of stats to "batch" per message (default 0)
	log     *logging.Logger

	shutitdown bool
	shutdown   chan bool

	resolution uint32

	cacher        *Cacher
	cacheOverFlow *broadcast.Listener
}

func NewKafkaMetrics() *KafkaMetrics {
	kf := new(KafkaMetrics)
	kf.batches = 0
	kf.log = logging.MustGetLogger("writers.kafka.metrics")

	kf.shutitdown = false
	kf.shutdown = make(chan bool)
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

	cache_key := fmt.Sprintf("kafka:cache:%s:%v", dsn, kf.resolution)
	kf.cacher, err = getCacherSingleton(cache_key)
	if err != nil {
		return err
	}

	// only set these if it's not been started/init'ed
	// as the readers will use this object as well
	if !kf.cacher.started && !kf.cacher.inited {
		kf.cacher.inited = true

		_cz := conf["cache_metric_size"]
		if _cz != nil {
			kf.cacher.maxKeys = _cz.(int)
		} else {
			kf.cacher.maxKeys = CACHER_METRICS_KEYS
		}

		// cacher and mysql options for series
		_bt := conf["series_encoding"]
		if _bt != nil {
			kf.cacher.seriesType = _bt.(string)
		} else {
			kf.cacher.seriesType = KAFKA_DEFAULT_SERIES_TYPE
		}

		_bz := conf["cache_byte_size"]
		if _bz != nil {
			kf.cacher.maxBytes = int(_bz.(int64))
		} else {
			kf.cacher.maxBytes = KAFKA_DEFAULT_SERIES_CHUNK
		}

		// we write the overflows
		kf.cacher.overFlowMethod = "chan"
		kf.cacheOverFlow = kf.cacher.GetOverFlowChan()
	}

	return nil
}

func (kf *KafkaMetrics) Start() {
	kf.log.Notice("Starting Kafka writer for %s at %d bytes per series", kf.db.DataTopic(), kf.cacher.maxBytes)
	kf.cacher.Start()
	go kf.overFlowWrite()
}

func (kf *KafkaMetrics) Stop() {
	shutdown.AddToShutdown()
	defer shutdown.ReleaseFromShutdown()

	if kf.shutitdown {
		return // already did
	}
	kf.shutitdown = true
	kf.shutdown <- true
	kf.cacher.Stop()

	mets := kf.cacher.Queue
	mets_l := len(mets)
	kf.log.Warning("Shutting down, exhausting the queue (%d items) and quiting", mets_l)
	// full tilt write out
	did := 0
	for _, queueitem := range mets {
		if did%100 == 0 {
			kf.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
		}
		name, points, _ := kf.cacher.GetSeriesById(queueitem.metric)
		if points != nil {
			stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandra.write.send-to-writers"), 1)
			kf.PushSeries(name, points)
		}
		did++
	}
	err := kf.conn.Close()
	if err != nil {
		kf.log.Error("shutdown of kafka connection failed: %v", err)
	}
	kf.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
	kf.log.Warning("Shutdown finished ... quiting kafka writer")
	return
}

// listen to the overflow chan from the cache and attempt to write "now"
func (kf *KafkaMetrics) overFlowWrite() {
	for {
		select {
		case statitem, more := <-kf.cacheOverFlow.Ch:

			// bail
			if kf.shutitdown || !more {
				return
			}
			kf.PushSeries(statitem.(*TotalTimeSeries).Name, statitem.(*TotalTimeSeries).Series)
		}
	}
}

func (kf *KafkaMetrics) SetIndexer(idx indexer.Indexer) error {
	kf.indexer = idx
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (kf *KafkaMetrics) SetResolutions(res [][]int) int {
	kf.resolutions = res
	return len(res) // need as many writers as bins
}

func (kf *KafkaMetrics) Write(stat repr.StatRepr) error {
	// merge the tags in
	stat.Name.MergeMetric2Tags(kf.static_tags)
	kf.indexer.Write(stat.Name) // to the indexer
	kf.cacher.Add(&stat.Name, &stat)
	return nil
}

func (kf *KafkaMetrics) PushSeries(name *repr.StatName, points series.TimeSeries) error {

	obj := &KafkaMetric{
		Type:       "metricblob",
		Metric:     name.Key,
		Time:       time.Now().UnixNano(),
		Data:       points.Bytes(),
		Encoding:   points.Name(),
		Resolution: name.Resolution,
		TTL:        name.TTL,
		Id:         name.UniqueId(),
		Uid:        name.UniqueIdString(),
		Tags:       name.SortedTags().Tags(),
		MetaTags:   name.SortedMetaTags().Tags(),
	}

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.DataTopic(),
		Key:   sarama.StringEncoder(obj.Id), // hash on unique id
		Value: obj,
	}

	stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes", 1)
	return nil
}

/**** READER ***/
// needed to match interface, but we obviously cannot do this
func (kf *KafkaMetrics) Render(path string, from string, to string) (WhisperRenderItem, error) {
	return WhisperRenderItem{}, fmt.Errorf("KAKFA DRIVER CANNOT DO RENDER")
}
func (kf *KafkaMetrics) RawRender(path string, from string, to string) ([]*RawRenderItem, error) {
	return []*RawRenderItem{}, fmt.Errorf("KAKFA DRIVER CANNOT DO RENDER")
}
