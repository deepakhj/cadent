/*
	FIFO cacher elements

	will keep the last REPR_CACHE_ITEMS metrics with REPR_CACHE_POINTS data points in RAM
	in an LRU fashion (LRU on the REPR_CACHE_ITEMS access). So keep in minds that REPR_CACHE_ITEMS * REPR_CACHE_POINTS
	may eat your ram up pretty fast
*/

package repr

import (
	"container/list"
	"sync"
)

const REPR_CACHE_POINTS = 1000
const REPR_CACHE_ITEMS = 100000

/** FIFO cacher elements **/

type cacherItem struct {
	keyele *list.Element
	reprs  *ReprList
}

type ReprCache struct {
	MaxSize  int
	itemList *list.List
	cache    map[StatId]*cacherItem

	mu sync.Mutex
}

func NewReprCache(size int) *ReprCache {
	if size <= 0 {
		size = REPR_CACHE_ITEMS
	}
	return &ReprCache{
		MaxSize:  size,
		cache:    make(map[StatId]*cacherItem),
		itemList: list.New(),
	}
}

func (s *ReprCache) Len() int {
	return s.itemList.Len()
}

func (s *ReprCache) Delete(key StatId) *ReprList {
	s.mu.Lock()
	defer s.mu.Unlock()
	element := s.cache[key]
	if element == nil {
		return nil
	}
	s.itemList.Remove(element.keyele)
	element.keyele = nil
	delete(s.cache, key)
	return element.reprs
}

func (s *ReprCache) Pop() *ReprList {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := s.itemList.Front()
	kk := k.Value.(StatId)

	if k != nil {
		element := s.cache[kk]
		delete(s.cache, kk)
		s.itemList.Remove(element.keyele)
		element.keyele = nil
		return element.reprs
	}
	return nil
}

func (s *ReprCache) Add(stat StatRepr) *ReprList {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := stat.Name.UniqueId()
	gots := s.cache[k]
	if gots == nil {
		old := s.checkSize()
		r_list := new(cacherItem)
		r_list.reprs = new(ReprList)
		r_list.reprs.Add(stat)

		kk := s.itemList.PushBack(k)
		r_list.keyele = kk

		s.cache[k] = r_list
		return old
	}
	gots.reprs.Add(stat)
	s.cache[k] = gots
	return nil
}

func (s *ReprCache) Get(key StatId) *ReprList {
	s.mu.Lock()
	defer s.mu.Unlock()
	element := s.cache[key]
	if element == nil {
		return nil
	}
	return element.reprs
}

func (s *ReprCache) checkSize() *ReprList {
	//locking outside this function please
	if s.itemList.Len() >= s.MaxSize {
		k := s.itemList.Front()
		key := k.Value.(StatId)
		element := s.cache[key]
		s.itemList.Remove(element.keyele)
		element.keyele = nil
		delete(s.cache, key)
		return element.reprs
	}
	return nil
}

var STAT_REPR_CACHE *ReprCache

// fire up the singleton
func init() {
	STAT_REPR_CACHE = NewReprCache(-1)
}
