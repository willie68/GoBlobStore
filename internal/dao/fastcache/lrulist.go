package fastcache

import (
	"sort"
	"sync"
	"time"

	"github.com/willie68/GoBlobStore/pkg/model"
)

type LRUEntry struct {
	LastAccess  time.Time             `json:"lastAccess"`
	Description model.BlobDescription `json:"description"`
	Data        []byte                `json:"data"`
}

type LRUList struct {
	MaxCount   int
	MaxRamSize int64
	entries    []LRUEntry
	dmu        sync.Mutex
	ramsize    int64
}

func (l *LRUList) Init() {
	l.entries = make([]LRUEntry, 0)
}

func (l *LRUList) Size() int {
	return len(l.entries)
}

func (l *LRUList) Add(e LRUEntry) bool {
	l.dmu.Lock()
	defer l.dmu.Unlock()
	l.entries = l.insertSorted(l.entries, e)
	if e.Data != nil {
		l.ramsize += int64(len(e.Data))
	}
	return true
}

func (l *LRUList) UpdateAccess(id string) {
	l.dmu.Lock()
	defer l.dmu.Unlock()
	i := sort.Search(len(l.entries), func(i int) bool { return l.entries[i].Description.BlobID >= id })
	if i < len(l.entries) && l.entries[i].Description.BlobID == id {
		l.entries[i].LastAccess = time.Now()
	}
}

func (l *LRUList) GetFullIDList() []string {
	l.dmu.Lock()
	defer l.dmu.Unlock()
	ids := make([]string, len(l.entries))
	for x, e := range l.entries {
		ids[x] = e.Description.BlobID
	}
	return ids
}

func (l *LRUList) HandleContrains() string {
	var id string
	l.dmu.Lock()
	defer l.dmu.Unlock()
	if len(l.entries) > int(l.MaxCount) {
		// remove oldest entry from cache
		oldest := l.getOldest()
		id = l.entries[oldest].Description.BlobID
	}
	if l.MaxRamSize > 0 {
		for l.ramsize > l.MaxRamSize {
			oldest := l.getOldestWithData()
			if oldest == -1 {
				l.ramsize = 0
				break
			}
			l.ramsize -= int64(len(l.entries[oldest].Data))
			l.entries[oldest].Data = nil
		}
	}
	return id
}

func (l *LRUList) Has(id string) bool {
	l.dmu.Lock()
	defer l.dmu.Unlock()
	i := sort.Search(len(l.entries), func(i int) bool { return l.entries[i].Description.BlobID >= id })
	if i < len(l.entries) && l.entries[i].Description.BlobID == id {
		return true
	}
	return false
}

func (l *LRUList) Get(id string) (LRUEntry, bool) {
	l.dmu.Lock()
	defer l.dmu.Unlock()
	i := sort.Search(len(l.entries), func(i int) bool { return l.entries[i].Description.BlobID >= id })
	if i < len(l.entries) && l.entries[i].Description.BlobID == id {
		l.entries[i].LastAccess = time.Now()
		return l.entries[i], true
	}
	return LRUEntry{}, false
}

func (l *LRUList) Delete(id string) string {
	l.dmu.Lock()
	defer l.dmu.Unlock()
	i := sort.Search(len(l.entries), func(i int) bool { return l.entries[i].Description.BlobID >= id })
	if i < len(l.entries) && l.entries[i].Description.BlobID == id {
		if l.entries[i].Data != nil {
			l.ramsize -= int64(len(l.entries[i].Data))
		}
		ret := make([]LRUEntry, 0)
		ret = append(ret, l.entries[:i]...)
		l.entries = append(ret, l.entries[i+1:]...)
		return id
	}
	return ""
}

func (l *LRUList) getOldest() int {
	oldest := 0
	for x, e := range l.entries {
		if e.LastAccess.Before(l.entries[oldest].LastAccess) {
			oldest = x
		}
	}
	return oldest
}

func (l *LRUList) getOldestWithData() int {
	oldest := -1
	for x, e := range l.entries {
		if e.Data != nil {
			if oldest == -1 || e.LastAccess.Before(l.entries[oldest].LastAccess) {
				oldest = x
			}
		}
	}
	return oldest
}

func (l *LRUList) insertSorted(data []LRUEntry, v LRUEntry) []LRUEntry {
	i := sort.Search(len(data), func(i int) bool { return data[i].Description.BlobID >= v.Description.BlobID })
	return l.insertEntryAt(data, i, v)
}

func (l *LRUList) insertEntryAt(data []LRUEntry, i int, v LRUEntry) []LRUEntry {
	if i == len(data) {
		// Insert at end is the easy case.
		return append(data, v)
	}

	// Make space for the inserted element by shifting
	// values at the insertion index up one index. The call
	// to append does not allocate memory when cap(data) is
	// greater ​than len(data).
	data = append(data[:i+1], data[i:]...)

	// Insert the new element.
	data[i] = v

	// Return the updated slice.
	return data
}
