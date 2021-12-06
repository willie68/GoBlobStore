package fastcache

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	clog "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	BINARY_EXT = ".bin"
	// this is the max size of a blob that will be stored into memory, if possible
	Defaultmffrs = 100 * 1024
)

var (
	errEmptyIndex     error = errors.New("empty id not allowed")
	errNotImplemented error = errors.New("method not implemented in fastcache")
)

type FastCache struct {
	RootPath          string // this is the root path for the file system storage
	MaxCount          int64
	MaxRamSize        int64
	MaxFileSizeForRAM int64
	size              int64
	count             int64
	entries           []LRUEntry
	dmu               sync.Mutex
}

type LRUEntry struct {
	lastAccess  time.Time
	description model.BlobDescription
	data        []byte
}

var _ interfaces.BlobStorageDao = &FastCache{}

// initialise this dao
func (f *FastCache) Init() error {
	err := os.MkdirAll(f.RootPath, os.ModePerm)
	if err != nil {
		return err
	}

	f.count = 0
	f.size = 0

	err = f.removeContents(f.RootPath)
	if err != nil {
		return err
	}

	f.entries = make([]LRUEntry, 0)

	if f.MaxFileSizeForRAM == 0 {
		f.MaxFileSizeForRAM = Defaultmffrs
	}
	return nil
}

func (f *FastCache) removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// get the tenant id
func (f *FastCache) GetTenant() string {
	return "n.n."
}

// getting a list of blob from the storage
func (f *FastCache) GetBlobs(callback func(id string) bool) error {
	for _, d := range f.entries {
		next := callback(d.description.BlobID)
		if !next {
			break
		}
	}
	return nil
}

// CRUD operation on the blob files
// storing a blob to the storage system
func (f *FastCache) StoreBlob(b *model.BlobDescription, r io.Reader) (string, error) {
	ok, err := f.HasBlob(b.BlobID)
	if err != nil {
		clog.Logger.Errorf("cache: error checking file: %v", err)
		return "", err
	}
	if ok {
		clog.Logger.Errorf("cache: file exists")
		return b.BlobID, os.ErrExist
	}
	size, dat, err := f.writeBinFile(b.BlobID, r)
	if err != nil {
		clog.Logger.Errorf("cache: writing file: %v", err)
		return "", err
	}
	atomic.AddInt64(&f.size, size)
	f.entries = append(f.entries, LRUEntry{
		lastAccess:  time.Now(),
		description: *b,
		data:        dat,
	})
	err = f.handleContrains()
	if err != nil {
		clog.Logger.Errorf("cache: handle constrains: %v", err)
	}
	return b.BlobID, nil
}

func (f *FastCache) updateAccess(id string) {
	for _, e := range f.entries {
		if e.description.BlobID == id {
			e.lastAccess = time.Now()
			break
		}
	}
}

func (f *FastCache) handleContrains() error {
	f.dmu.Lock()
	defer f.dmu.Unlock()
	if len(f.entries) > int(f.MaxCount) {
		oldest := 0
		for x, e := range f.entries {
			if e.lastAccess.Before(f.entries[oldest].lastAccess) {
				oldest = x
			}
		}
		// remove oldest entry from cache
		f.deleteBlobWithIndex(oldest)
	}
	var ramsize int64 = 0
	oldest := 0
	for x, e := range f.entries {
		if e.data != nil {
			if e.lastAccess.Before(f.entries[oldest].lastAccess) {
				oldest = x
			}
			ramsize += int64(len(e.data))
		}
	}
	for ramsize > f.MaxRamSize {
		ramsize -= int64(len(f.entries[oldest].data))
		f.entries[oldest].data = nil
		oldest = f.getOldestWithData()
	}
	return nil
}

func (f *FastCache) getOldestWithData() int {
	oldest := 0
	for x, e := range f.entries {
		if e.data != nil {
			if e.lastAccess.Before(f.entries[oldest].lastAccess) {
				oldest = x
			}
		}
	}
	return oldest
}

func (f *FastCache) writeBinFile(id string, r io.Reader) (int64, []byte, error) {
	binFile, err := f.buildFilename(id, BINARY_EXT)
	if err != nil {
		return 0, nil, err
	}

	w, err := os.Create(binFile)

	if err != nil {
		return 0, nil, err
	}
	size, err := w.ReadFrom(r)
	if err != nil {
		w.Close()
		os.Remove(binFile)
		return 0, nil, err
	}
	w.Close()
	if size < f.MaxFileSizeForRAM {
		dat, err := os.ReadFile(binFile)
		if err != nil {
			dat = nil
		}
		return size, dat, nil
	}
	return size, nil, nil
}

func (f *FastCache) buildFilename(id string, ext string) (string, error) {
	fp := f.RootPath
	fp = filepath.Join(fp, id[:2])
	fp = filepath.Join(fp, id[2:4])
	err := os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		return "", err
	}
	return filepath.Join(fp, fmt.Sprintf("%s%s", id, ext)), nil
}

// checking, if a blob is present
func (f *FastCache) HasBlob(id string) (bool, error) {
	if id == "" {
		return false, errEmptyIndex
	}
	for _, e := range f.entries {
		if e.description.BlobID == id {
			return true, nil
		}
	}
	return false, nil
}

// getting the description of the file
func (f *FastCache) GetBlobDescription(id string) (*model.BlobDescription, error) {
	if id == "" {
		return nil, errEmptyIndex
	}
	for _, e := range f.entries {
		if e.description.BlobID == id {
			go f.updateAccess(id)
			return &e.description, nil
		}
	}
	return nil, os.ErrNotExist
}

// retrieving the binary data from the storage system
func (f *FastCache) RetrieveBlob(id string, w io.Writer) error {
	if id == "" {
		return errEmptyIndex
	}
	for _, e := range f.entries {
		if e.description.BlobID == id {
			go f.updateAccess(id)
			// checking memory cache
			if e.data != nil {
				_, err := w.Write(e.data)
				if err != nil {
					return err
				}
				return nil
			} else {
				err := f.getBlob(id, w)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
	return os.ErrNotExist
}

func (f *FastCache) getBlob(id string, w io.Writer) error {
	binFile, err := f.buildFilename(id, BINARY_EXT)
	if err != nil {
		return err
	}
	if _, err := os.Stat(binFile); os.IsNotExist(err) {
		return os.ErrNotExist
	}
	r, err := os.Open(binFile)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	return nil
}

// removing a blob from the storage system
func (f *FastCache) DeleteBlob(id string) error {
	if id == "" {
		return errEmptyIndex
	}
	f.dmu.Lock()
	defer f.dmu.Unlock()
	index := -1
	for x, e := range f.entries {
		if e.description.BlobID == id {
			index = x
			break
		}
	}
	if index >= 0 {
		f.deleteBlobWithIndex(index)
		return nil
	}
	return os.ErrNotExist
}

func (f *FastCache) deleteBlobWithIndex(x int) {
	if x >= 0 {
		bd := f.entries[x]
		f.deleteBlobFile(bd.description.BlobID)
		ret := make([]LRUEntry, 0)
		ret = append(ret, f.entries[:x]...)
		f.entries = append(ret, f.entries[x+1:]...)
	}
}

func (f *FastCache) deleteBlobFile(id string) error {
	binFile, err := f.buildFilename(id, BINARY_EXT)
	if err != nil {
		return err
	}
	if _, err := os.Stat(binFile); os.IsNotExist(err) {
		return os.ErrNotExist
	}
	err = os.Remove(binFile)
	if err != nil {
		return err
	}
	return nil
}

//Retentionrelated methods
// for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (f *FastCache) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return errNotImplemented
}
func (f *FastCache) AddRetention(r *model.RetentionEntry) error {
	return errNotImplemented
}
func (f *FastCache) GetRetention(id string) (model.RetentionEntry, error) {
	return model.RetentionEntry{}, errNotImplemented
}
func (f *FastCache) DeleteRetention(id string) error {
	return errNotImplemented
}
func (f *FastCache) ResetRetention(id string) error {
	return errNotImplemented
}

// closing the storage
func (f *FastCache) Close() error {
	return nil
}
