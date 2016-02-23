/*
	The Cassandra reader

	The table should have this schema to match the repr item
	The same as the writer items


		keyspace: base keyspace name (default: metric)
		metric_table: base table name (default: metric)
		path_table: base table name (default: path)
		segment_table: base table name (default: segment)
		write_consistency: "one"
		read_consistency: "one"
		port: 9042
		numcons: 5  (connection pool size)
		timeout: "30s"
		user: ""
		pass: ""
		# NOPE: batch_count: batch this many inserts for much faster insert performance (default 1000)
		# NOPE: periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package readers

import (
	"consthash/server/dbs"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"regexp"
	"strings"
	"time"
)

/****************** Writer *********************/
type CassandraReader struct {
	// juse the writer connections for this
	db   *dbs.CassandraDB
	conn *gocql.Session

	log *logging.Logger
}

func NewCassandraReader() *CassandraReader {
	cass := new(CassandraReader)
	cass.log = logging.MustGetLogger("reader.cassandra")

	return cass
}

func (cass *CassandraReader) Config(conf map[string]interface{}) (err error) {

	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (server1,server2,server3) is needed for cassandra config")
	}
	dsn := gots.(string)

	db, err := dbs.NewDB("cassandra", dsn, conf)
	if err != nil {
		return err
	}
	cass.db = db.(*dbs.CassandraDB)
	cass.conn = db.Connection().(*gocql.Session)
	return nil

}

func (cass *CassandraReader) ExpandNonRegex(metric string) (MetricExpandItem, error) {
	paths := strings.Split(metric, ".")
	m_len := len(paths)
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=? AND segment=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		m_len-1, metric,
	).Iter()

	var on_pth string

	var me MetricExpandItem
	// just grab the "n+1" length ones
	for iter.Scan(&on_pth) {
		me.Results = append(me.Results, on_pth)
	}
	if err := iter.Close(); err != nil {
		return me, err
	}
	return me, nil
}

// Expand simply pulls out any regexes into full form
func (cass *CassandraReader) Expand(metric string) (MetricExpandItem, error) {

	has_reg := regexp.MustCompile(`\*|\{|\}|\[|\]`)
	needs_regex := has_reg.Match([]byte(metric))
	if !needs_regex {
		return cass.ExpandNonRegex(metric)
	}
	paths := strings.Split(metric, ".")
	m_len := len(paths)

	var me MetricExpandItem

	// convert the "graphite regex" into something golang understands (just the "."s really)
	regable := strings.Replace(metric, ".", "\\.", 0)
	the_reg, err := regexp.Compile(regable)
	if err != nil {
		return me, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		m_len-1,
	).Iter()

	var seg string
	// just grab the "n+1" length ones
	for iter.Scan(&seg) {
		//cass.log.Debug("SEG: %s", seg)

		if !the_reg.Match([]byte(seg)) {
			continue
		}
		me.Results = append(me.Results, seg)
	}
	if err := iter.Close(); err != nil {
		return me, err
	}
	return me, nil

}

func (cass *CassandraReader) Render(path string, from string, to string) (RenderItems, error) {

	var ri RenderItems

	start, err := ParseTime(from)
	if err != nil {
		cass.log.Error("Invalid from time `%s` :: %v", from, err)
	}

	end, err := ParseTime(to)
	if err != nil {
		cass.log.Error("Invalid from time `%s` :: %v", from, err)
	}
	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nans := int64(time.Second)
	end = end * nans
	start = start * nans
	if end > start {
		start, end = end, start
	}

	metrics, err := cass.Find(path)
	if err != nil {
		return ri, err
	}
	for _, metric := range metrics {
		if metric.Leaf == 0 {
			continue
		}

		var ritem RenderItem
		// grab ze data.
		cass_Q := fmt.Sprintf(
			"SELECT point.mean, point.max, point.min, point.sum, time FROM %s WHERE id={path: ?, resolution: ?} AND time <= ? and time >= ?",
			cass.db.MetricTable(),
		)
		iter := cass.conn.Query(cass_Q,
			metric.Id, 60, start, end,
		).Iter()

		//cass.log.Debug("METR: %s Start: %d END: %d", metric.Id, start, end)
		var t int64
		var mean, min, max, sum float64

		use_mean := strings.HasPrefix(metric.Text, "mean")
		ritem.Target = metric.Id

		for iter.Scan(&mean, &max, &min, &sum, &t) {
			use_t := int(t / nans)
			if use_mean {
				ritem.Datapoints = append(ritem.Datapoints, DataPoint{
					Time:  use_t,
					Value: mean,
				})
			} else {
				ritem.Datapoints = append(ritem.Datapoints, DataPoint{
					Time:  use_t,
					Value: sum,
				})
			}
		}
		if err := iter.Close(); err != nil {
			cass.log.Error("Render: Failure closing iterator: %v", err)
		}
		ri = append(ri, ritem)
	}

	return ri, nil
}

// basic find for non-regex items
func (cass *CassandraReader) FindNonRegex(metric string) (MetricFindItems, error) {

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone
	// down
	paths := strings.Split(metric, ".")
	m_len := len(paths)

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes
	cass_Q := fmt.Sprintf(
		"SELECT path,length,has_data FROM %s WHERE segment={pos: ?, segment: ?} ",
		cass.db.PathTable(),
	)
	iter := cass.conn.Query(cass_Q,
		m_len-1, metric,
	).Iter()

	var mt MetricFindItems
	var ms MetricFindItem
	var on_pth string
	var pth_len int
	var has_data bool

	// just grab the "n+1" length ones
	for iter.Scan(&on_pth, &pth_len, &has_data) {
		//cass.log.Debug("PATH %s LEN %d m_len: %d", on_pth, pth_len, m_len)
		if pth_len > m_len {
			continue
		}
		spl := strings.Split(on_pth, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = on_pth

		if has_data {
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
		} else {
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
		}

		mt = append(mt, ms)
	}

	if err := iter.Close(); err != nil {
		return mt, err
	}

	// now we also need any paths that are not leaf nodes that are

	return mt, nil
}

func (cass *CassandraReader) Find(metric string) (MetricFindItems, error) {

	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide rege abilities
	// on the servier side
	has_reg := regexp.MustCompile(`\*|\{|\}|\[|\]`)
	needs_regex := has_reg.Match([]byte(metric))

	//cass.log.Debug("HasReg: %v Metric: %s", needs_regex, metric)
	if !needs_regex {
		return cass.FindNonRegex(metric)
	}

	paths := strings.Split(metric, ".")
	m_len := len(paths)

	// convert the "graphite regex" into something golang understands (just the "."s really)
	regable := strings.Replace(metric, ".", "\\.", 0)
	the_reg, err := regexp.Compile(regable)
	if err != nil {
		return nil, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		m_len-1,
	).Iter()

	var mt MetricFindItems

	var seg string
	// just grab the "n+1" length ones
	for iter.Scan(&seg) {
		//cass.log.Debug("SEG: %s", seg)

		if !the_reg.Match([]byte(seg)) {
			continue
		}
		items, err := cass.FindNonRegex(seg)
		if err != nil {
			cass.log.Warning("could not get segments for `%s` :: %v", seg, err)
			continue
		}
		if items != nil && len(items) > 0 {
			for _, ms := range items {
				mt = append(mt, ms)
			}
			items = nil
		}
	}

	return mt, nil

}
