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

	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

/****************** Writer *********************/
type CassandraReader struct {
	// juse the writer connections for this
	db          *dbs.CassandraDB
	conn        *gocql.Session
	resolutions [][]int

	render_wg sync.WaitGroup
	render_mu sync.Mutex

	log *logging.Logger
}

func NewCassandraReader() *CassandraReader {
	cass := new(CassandraReader)
	cass.log = logging.MustGetLogger("reader.cassandra")

	return cass
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (cass *CassandraReader) SetResolutions(res [][]int) {
	cass.resolutions = res
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

// based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
func (cass *CassandraReader) getResolution(from int64, to int64) int {
	diff := int(math.Abs(float64(to - from)))
	for _, res := range cass.resolutions {
		if diff < res[1] {
			return res[0]
		}
	}
	return cass.resolutions[len(cass.resolutions)-1][0]
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

func (cass *CassandraReader) RenderOne(path string, from string, to string) (WhisperRenderItem, error) {

	var whis WhisperRenderItem

	start, err := ParseTime(from)
	if err != nil {
		cass.log.Error("Invalid from time `%s` :: %v", from, err)
	}

	end, err := ParseTime(to)
	if err != nil {
		cass.log.Error("Invalid from time `%s` :: %v", from, err)
	}
	if end < start {
		start, end = end, start
	}
	//figure out the best res
	resolution := cass.getResolution(start, end)

	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nano_end := end * nano
	nano_start := start * nano

	metrics, err := cass.Find(path)
	if err != nil {
		return whis, err
	}

	series := make(map[string][]DataPoint)

	first_t := int(start)
	last_t := int(end)
	for _, metric := range metrics {
		if metric.Leaf == 0 { //data only
			continue
		}
		b_len := int(end-start) / resolution //just to be safe
		if b_len <= 0 {
			continue
		}

		// grab ze data.
		cass_Q := fmt.Sprintf(
			"SELECT point.mean, point.max, point.min, point.sum, time FROM %s WHERE id={path: ?, resolution: ?} AND time <= ? and time >= ?",
			cass.db.MetricTable(),
		)
		iter := cass.conn.Query(cass_Q,
			metric.Id, resolution, nano_end, nano_start,
		).Iter()

		var t int64
		var mean, min, max, sum float64

		// yes this is not the best use of "smart-y-pants ness" but it does the job
		// not sure how to detect the min/max usage .. yet
		use_mean := strings.HasPrefix(metric.Text, "mean")
		m_key := metric.Id

		d_points := make([]DataPoint, b_len)
		//step_map := make(map[int64]DataPoint)

		// sorting order for the table is time ASC (i.e. first_t == first entry)

		// Since graphite does not care about the actual time stamp, but assumes
		// a "constant step" in time.  We figure out the
		// "step" -> Point mapping which for this system may not be exactly at
		// step intervals, so since data may not nessesarily "be there" for a given
		// interval we need to "insert nils" for steps that don't really exist
		// as basically (start - end) / resolution needs to match
		// the vector length, in effect we need to "interpolate" the vector to match sizes

		ct := 0
		for iter.Scan(&mean, &max, &min, &sum, &t) {
			on_t := int(t / nano) // back convert to seconds
			d_point := NewDataPoint(on_t, mean)
			if !use_mean {
				d_point.SetValue(sum)
			}

			if len(d_points) <= ct {
				_t_pts := make([]DataPoint, len(d_points)+100)
				copy(_t_pts, d_points)
				d_points = _t_pts
			}
			d_points[ct] = d_point

			ct++
			last_t = on_t
		}

		if err := iter.Close(); err != nil {
			cass.log.Error("Render: Failure closing iterator: %v", err)
		}

		if len(d_points) > 0 {
			first_t = d_points[0].Time
		}
		// now for the interpolation bit .. just assume linear as we hope metric points are
		// luckily things are "sorted" already (or should be)
		// XXX HOPEFULLY there are usually FEWER or as much "real" data then not
		//
		interp_vec := make([]DataPoint, b_len)
		cur_step_time := int(start)

		j := 0
		for i := int(0); i < b_len; i++ {

			interp_vec[i] = DataPoint{Time: cur_step_time, Value: nil}
			cur_step_time += resolution

			for ; j < ct; j++ {
				//cass.log.Critical("Start:%d, End:%d, d_points: %d interp_vec: %d  IIdx: %d Jidx: %d ", start, end-start, d_points[j].Time-int(start), cur_step_time-int(start), i, j)
				if d_points[j].Time <= cur_step_time {
					interp_vec[i].Value = d_points[j].Value
					j++
					break
				} else {
					break
				}
			}

		}

		series[m_key] = interp_vec

		cass.log.Critical("METR: %s Start: %d END: %d LEN: %d", metric.Id, first_t, last_t, len(d_points))
	}

	whis.Series = series
	whis.Start = first_t
	whis.End = last_t
	whis.Step = resolution
	return whis, nil
}

func (cass *CassandraReader) Render(path string, from string, to string) (WhisperRenderItem, error) {

	var whis WhisperRenderItem
	whis.Series = make(map[string][]DataPoint)
	paths := strings.Split(path, ",")

	// ye old fan out technique
	render_one := func(pth string) {
		_ri, err := cass.RenderOne(pth, from, to)
		if err != nil {
			cass.render_wg.Done()
			return
		}
		cass.render_mu.Lock()
		for k, rr := range _ri.Series {
			whis.Series[k] = rr
		}
		whis.Start = _ri.Start
		whis.End = _ri.End
		whis.Step = _ri.Step

		cass.render_mu.Unlock()
		cass.render_wg.Done()
		return
	}

	for _, pth := range paths {
		if len(pth) == 0 {
			continue
		}
		cass.render_wg.Add(1)
		go render_one(pth)
	}
	cass.render_wg.Wait()
	return whis, nil

}

// basic find for non-regex items
func (cass *CassandraReader) FindNonRegex(metric string) (MetricFindItems, error) {

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone down

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

// special case for "root" == "*" finder
func (cass *CassandraReader) FindRoot() (MetricFindItems, error) {

	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		0,
	).Iter()

	var mt MetricFindItems
	var seg string

	for iter.Scan(&seg) {
		var ms MetricFindItem
		ms.Text = seg
		ms.Id = seg
		ms.Path = seg

		ms.Expandable = 1
		ms.Leaf = 0
		ms.AllowChildren = 1

		mt = append(mt, ms)
	}

	if err := iter.Close(); err != nil {
		return mt, err

	}

	return mt, nil
}

// to allow for multiple targets
func (cass *CassandraReader) Find(metric string) (MetricFindItems, error) {
	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide regex abilities
	// on the server side

	// special case for "root" == "*"

	if metric == "*" {
		return cass.FindRoot()
	}

	paths := strings.Split(metric, ".")
	m_len := len(paths)

	//if the last fragment is "*" then we realy mean just then next level, not another "." level
	// this is the graphite /find?query=consthash.zipperwork.local which will mean the same as
	// /find?query=consthash.zipperwork.local.* for us
	if paths[len(paths)-1] == "*" {
		metric = strings.Join(paths[:len(paths)-1], ".")
		paths = strings.Split(metric, ".")
		m_len = len(paths)
	}

	has_reg := regexp.MustCompile(`\*|\{|\}|\[|\]`)
	needs_regex := has_reg.Match([]byte(metric))

	//cass.log.Debug("HasReg: %v Metric: %s", needs_regex, metric)
	if !needs_regex {
		return cass.FindNonRegex(metric)
	}

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
		cass.log.Debug("REG:::::PATH %s", seg)

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
