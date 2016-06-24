// Copyright (C) 2015 Myfitnesspal

// testing out the leveldb world indexer
//
package main

import (
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_opt "github.com/syndtr/goleveldb/leveldb/opt"
	leveldb_util "github.com/syndtr/goleveldb/leveldb/util"
	leveldb_filter "github.com/syndtr/goleveldb/leveldb/filter"
	"flag"
	"path/filepath"
	"log"
	"math/rand"

	"strings"
	"fmt"
	"regexp"
	"bytes"
)


var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var randWords = []string{"test", "house", "here", "badline", "cow", "now"}


func RandItem(strs []string) string {
	return strs[rand.Intn(len(strs))]
}


func sprinter(ct int) string {
	r_ws := []string{RandItem(randWords)}

	for i := 0; i < ct-1; i++ {
		r_ws = append(r_ws, RandItem(randWords))
	}
	return strings.Join(r_ws, ".")
}

// random char gen
func RandChars(length int) string {

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)

}

func getKey(length int, wordlen int) string{
	out_arr := make([]string, 0)
	for i:=int(0); i<length; i++{
		out_arr = append(out_arr, RandChars(wordlen))
	}
	return strings.Join(out_arr, ".")
}


type LevelDBSegment struct {
	Pos     int
	Length  int
	Segment string
	NextSegment string
	Path string
}

func ParseKey(stat_key string) (segments []LevelDBSegment){
	s_parts := strings.Split(stat_key, ".")
	p_len := len(s_parts)

	cur_part := ""
	next_part := ""

	segments = make([]LevelDBSegment, p_len)

	if p_len > 0{
		next_part = s_parts[0]
	}

	// for each "segment" add in the path to do a segment to path(s) lookup
	/*
		for key consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99

		Segment -> NextSegment

		consthash -> consthash.zipperwork
		consthash.zipperwork -> consthash.zipperwork.local
		consthash.zipperwork.local -> consthash.zipperwork.local.writer
		consthash.zipperwork.local.writer -> consthash.zipperwork.local.writer.cassandra
		consthash.zipperwork.local.writer.cassandra -> consthash.zipperwork.local.writer.cassandra.write
		consthash.zipperwork.local.writer.cassandra.write -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns
		consthash.zipperwork.local.writer.cassandra.write.metric-time-ns -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99

		Segment -> Path
		consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99 -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
	*/
	for idx, part := range s_parts {
		if len(cur_part) > 1 {
			cur_part += "."
		}
		cur_part += part
		if idx < p_len && idx > 0{
			next_part += "."
			next_part += part
		}

		segments[idx] = LevelDBSegment{
			Segment: cur_part,
			NextSegment: next_part,
			Path:    stat_key,
			Length: p_len - 1,
			Pos:  idx, // starts at 0
		}
	}

	return segments
}

func (ls *LevelDBSegment) SegmentKey(segment string , len int) ([]byte) {
	return []byte(fmt.Sprintf("SEG:%d:%s", len, segment))
}

func (ls *LevelDBSegment) SegmentData() ([]byte, []byte) {
	return []byte(ls.SegmentKey(ls.Segment, ls.Length)), []byte(ls.Path)
}

func (ls *LevelDBSegment) PathKey(path string) ([]byte) {
	return []byte(fmt.Sprintf("PATH:%s", path))
}
func (ls *LevelDBSegment) ReverseSegmentData() ([]byte, []byte) {
	return []byte(ls.PathKey(ls.Path)), []byte(fmt.Sprintf("%d:%s", ls.Pos, ls.Segment))
}

func (ls *LevelDBSegment) PosSegmentKey(path string, pos int) ([]byte) {
	return []byte(fmt.Sprintf("POS:%d:%s", pos, path))
}

// POS:{pos}:{segment} -> {segment+1}
func (ls *LevelDBSegment) PosSegmentData() ([]byte, []byte) {
	has_data := "0"
	if ls.NextSegment == ls.Path{
		has_data = "1"
	}
	return []byte(ls.PosSegmentKey(ls.Segment, ls.Pos)),
	[]byte(fmt.Sprintf("%s:%s", ls.NextSegment, has_data))

}

// if the full path and segment are the same ..
func (ls *LevelDBSegment) HasData() bool {
	return ls.Segment == ls.Path
}

func (ls *LevelDBSegment) InsertAll(segdb *leveldb.DB) (err error){

	batch := new(leveldb.Batch)
	// only added fo the "data" full
	if ls.HasData() {
		k, v :=  ls.SegmentData()
		batch.Put(k, v)
		k1, v1 := ls.ReverseSegmentData()
		batch.Put(k1,v1)
	}
	k,v := ls.PosSegmentData()
	batch.Put(k,v)
	err = segdb.Write(batch, nil)
	return
}

func (ls *LevelDBSegment) DeletePath(segdb *leveldb.DB) (err error) {
	// first see if the path is there

	v_byte := []byte(ls.Path)
	pos_byte := []byte(ls.Path +":1")
	path_key := ls.PathKey(ls.Path)

	val, err := segdb.Get(path_key, nil)
	if err != nil {
		return err
	}
	if len(val) == 0 {
		return fmt.Errorf("Path is not present")
	}

	// return val is {length}:{segment}
	log.Printf("DEL: GotPath: Path: %s :: data: %s", path_key, val)

	// grab all the segments
	segs := ParseKey(string(ls.Path))
	l_segs := len(segs)
	errs := make([]error, 0)
	// remove all things that point to this path
	for idx, seg := range segs{
		// only the "last" segment has a value, the sub lines are "POS:..."
		if l_segs == idx + 1 {
			// remove SEG:len:...
			seg_key := seg.SegmentKey(seg.Segment, seg.Length)
			log.Printf("To DEL: Segment: %s", seg_key)
			v, err := segdb.Get(seg_key, nil)
			log.Printf("To DEL: Segment: Error %v", err)
			if err != nil {
				errs = append(errs, err)
			} else if bytes.EqualFold(v, v_byte) {
				//EqualFold as these are strings at their core
				log.Printf("Deleting Segment: %s", v_byte)
				segdb.Delete(seg_key, nil)
			}
		}

		// remove the POS:len:... ones as well
		pos_key := seg.PosSegmentKey(seg.Segment, seg.Pos)
		v, err := segdb.Get(pos_key, nil)
		log.Printf("To DEL: Pos: %s: Error %v", pos_key, err)
		if err != nil{
			errs = append(errs, err)
		}else if bytes.EqualFold(v, pos_byte){
			log.Printf("Deleting Pos: %s", pos_byte)
			segdb.Delete(pos_key, nil)
		}
	}
	if len(errs) == 0{
		// remove the path
		log.Printf("Deleting Path: %s", path_key)
		segdb.Delete(path_key, nil)
	}else{
		return fmt.Errorf("Multiple errors trying to remove %s : %v", ls.Path, errs)
	}

	return nil
}


// convert the "graphite regex" into something golang understands (just the "."s really)
// need to replace things like "moo*" -> "moo.*" but careful not to do "..*"
// the "graphite" globs of {moo,goo} we can do with (moo|goo) so convert { -> (, , -> |, } -> )
func regifyKey(metric string) (*regexp.Regexp, error) {
	regable := strings.Replace(metric, "..", ".", -1)
	regable = strings.Replace(regable, "{", "(", -1)
	regable = strings.Replace(regable, "}", ")", -1)
	regable = strings.Replace(regable, ",", "|", -1)
	regable = strings.Replace(regable, ".", "\\.", -1)
	regable = strings.Replace(regable, "*", ".*", -1)
	return regexp.Compile(regable)
}

func Find(metric string, db *leveldb.DB)(paths []string, err error){

	segs := strings.Split(metric, ".")
	p_len := len(segs)

	// find the longest chunk w/o a reg and that will be the level db prefix filter
	needs_regex := strings.IndexAny(metric, "*?[{") >= 0

	long_chunk := ""
	use_key := metric
	use_key_len := p_len - 1
	var reged *regexp.Regexp
	if needs_regex {
		for _, pth := range segs {
			if strings.IndexAny(pth, "*?[{") >= 0{
				use_key = long_chunk
				break
			}
			if len(long_chunk) > 0 {
				long_chunk += "."
			}
			long_chunk += pth
		}
		reged, err = regifyKey(metric)
		if err != nil{
			return []string{}, err
		}

	}
	prefix := fmt.Sprintf("POS:%d:%s", use_key_len, use_key)
	log.Printf("Find: USE KEY: %s", prefix)
	paths = make([]string, 0)
	iter := db.NewIterator(leveldb_util.BytesPrefix([]byte(prefix)), nil)
	for iter.Next() {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		key := iter.Key()
		value := iter.Value()
		if reged != nil && reged.Match(value){
			paths = append(paths, string(value))
			log.Printf("Find Raw :: %s: %s", key, value)
		}else{
			log.Printf("Find NON REG Raw :: %s: %s", key, value)

		}


	}
	iter.Release()
	err = iter.Error()
	return
}


func main() {
	dbpth := flag.String("path", "/tmp", "db path")
	n_items := flag.Int("words", 10, "stat key words")
	find_str := flag.String("find", "now.test.here.cow.badline", "find this path")
	del_str := flag.String("del", "", "delete this path")

	flag.Parse()

	dbfile := filepath.Join(*dbpth, "test")
	o := &leveldb_opt.Options{
		Filter: leveldb_filter.NewBloomFilter(10),
		CompactionTableSize: leveldb_opt.MiB * 20,
	}
	db, err := leveldb.OpenFile(dbfile, o)
	defer db.Close()

	if err != nil {
		panic( err)
	}

	//strs := make(map[string]int)

	for i:=0; i<*n_items; i++{
		key := sprinter(5) //getKey(rand.Intn(10), rand.Intn(30))
		segs := ParseKey(key)
		for _, seg := range segs{
			seg.InsertAll(db)
		}

	}


	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		key := iter.Key()
		value := iter.Value()
		log.Printf("Dump: %s: %s", key, value)
	}
	iter.Release()
	err = iter.Error()
	strs, n_err := Find(*find_str, db)
	log.Printf("FIND:: %v : %v", strs, n_err)

	if len(*del_str) > 0{

		segs := ParseKey(*del_str)
		err := segs[0].DeletePath(db)
		log.Printf("DELETE:: %v : %v", del_str, err)
		strs, n_err := Find(*del_str, db)
		log.Printf("Should not be found:: %v : %v", strs, n_err)

	}
}
