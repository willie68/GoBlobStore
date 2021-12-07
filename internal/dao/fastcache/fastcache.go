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

	"github.com/akgarhwal/bloomfilter/bloomfilter"
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
	entries           LRUList
	bf                bloomfilter.BloomFilter
	bfDirty           bool
	bfm               sync.Mutex
	background        *time.Ticker
	quit              chan bool
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

	f.entries = LRUList{
		MaxCount:   int(f.MaxCount),
		MaxRamSize: f.MaxRamSize,
	}

	f.entries.Init()

	if f.MaxFileSizeForRAM == 0 {
		f.MaxFileSizeForRAM = Defaultmffrs
	}

	// initialise the bloomfilter
	f.bf = *bloomfilter.NewBloomFilter(uint64(f.MaxCount), 0.1)
	f.bfDirty = false
	f.background = time.NewTicker(60 * time.Second)
	f.quit = make(chan bool)
	go func() {
		for {
			select {
			case <-f.background.C:
				f.rebuildBloomFilter()
			case <-f.quit:
				f.background.Stop()
				return
			}
		}
	}()

	return nil
}

//removeContents delete all files in directory
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
	ids := f.entries.GetFullIDList()
	for _, id := range ids {
		next := callback(id)
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
	f.entries.Add(LRUEntry{
		LastAccess:  time.Now(),
		Description: *b,
		Data:        dat,
	})
	id := f.entries.HandleContrains()
	if id != "" {
		f.DeleteBlob(id)
	}
	f.updateBloom(b.BlobID)
	return b.BlobID, nil
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
	if f.inBloom(id) {
		if f.entries.Has(id) {
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
	if f.inBloom(id) {
		l, ok := f.entries.Get(id)
		if ok {
			return &l.Description, nil
		}
	}
	return nil, os.ErrNotExist
}

// retrieving the binary data from the storage system
func (f *FastCache) RetrieveBlob(id string, w io.Writer) error {
	if id == "" {
		return errEmptyIndex
	}
	if f.inBloom(id) {
		l, ok := f.entries.Get(id)
		if ok {
			// checking memory cache
			if l.Data != nil {
				_, err := w.Write(l.Data)
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
	if f.inBloom(id) {
		if f.entries.Has(id) {
			lid := f.entries.Delete(id)
			if lid != "" {
				f.deleteBlobFile(lid)
				f.bfDirty = true
			}
			return nil
		}
	}
	return os.ErrNotExist
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

func (f *FastCache) updateBloom(id string) {
	f.bfm.Lock()
	defer f.bfm.Unlock()
	f.bf.Insert([]byte(id))
}

func (f *FastCache) inBloom(id string) bool {
	f.bfm.Lock()
	defer f.bfm.Unlock()
	return f.bf.Lookup([]byte(id))
}

func (f *FastCache) rebuildBloomFilter() {
	if f.bfDirty {
		ids := f.entries.GetFullIDList()
		tb := bloomfilter.NewBloomFilter(uint64(f.MaxCount), 0.1)
		for _, id := range ids {
			tb.Insert([]byte(id))
		}
		// thats maybe not atomic!
		f.bfm.Lock()
		defer f.bfm.Unlock()
		f.bf = *tb
		f.bfDirty = false
	}
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
	f.quit <- true
	return nil
}
