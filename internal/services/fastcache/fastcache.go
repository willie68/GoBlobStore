// Package fastcache implementing a fast cache in memory and fast file storage
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
	"github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	// BinaryExt extension for binary files
	BinaryExt = ".bin"
	// Defaultmffrs this is the max size of a blob that will be stored into memory, if possible
	Defaultmffrs = 100 * 1024
	// DefaultTnt the default tenant. Because this storage isn't tenant specific
	DefaultTnt = "n.n."
)

var (
	errEmptyIndex     = errors.New("empty id not allowed")
	errNotImplemented = errors.New("method not implemented in fastcache")

	logger = logging.New().WithName("fastcache")
)

// FastCache a fast cache implementation using a mix of memory and fast ssd storage
type FastCache struct {
	RootPath          string // this is the root path for the file system storage
	MaxCount          int64
	MaxRAMSize        int64
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

// Checking interface compatibility
var _ interfaces.BlobStorage = &FastCache{}

// Init initialize this service
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
		MaxRAMSize: f.MaxRAMSize,
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

// removeContents delete all files in directory
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

// GetTenant get the tenant id, niy
func (f *FastCache) GetTenant() string {
	return DefaultTnt
}

// GetBlobs getting a list of blob from the storage
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

// StoreBlob storing a blob to the storage system
func (f *FastCache) StoreBlob(b *model.BlobDescription, r io.Reader) (string, error) {
	ok, err := f.HasBlob(b.BlobID)
	if err != nil {
		logger.Errorf("cache: error checking file: %v", err)
		return "", err
	}
	if ok {
		logger.Errorf("cache: file exists")
		return b.BlobID, os.ErrExist
	}
	size, dat, err := f.writeBinFile(b.BlobID, r)
	if err != nil {
		logger.Errorf("cache: writing file: %v", err)
		return "", err
	}
	atomic.AddInt64(&f.size, size)
	f.entries.Add(LRUEntry{
		LastAccess:  time.Now(),
		Description: *b,
		Data:        dat,
	})
	for {
		id := f.entries.HandleContrains()
		if id == "" {
			break
		}
		err = f.DeleteBlob(id)
		if err != nil {
			logger.Errorf("cache: can't delete blob %s: %v", id, err)
		}
	}
	f.updateBloom(b.BlobID)
	return b.BlobID, nil
}

// UpdateBlobDescription updating the blob description
func (f *FastCache) UpdateBlobDescription(id string, b *model.BlobDescription) error {
	ok, err := f.HasBlob(b.BlobID)
	if err != nil {
		logger.Errorf("cache: error checking file: %v", err)
		return err
	}
	if !ok {
		logger.Debugf("cache: file not exists: %s", id)
		return nil
	}
	if f.inBloom(id) {
		l, ok := f.entries.Get(id)
		if ok {
			l.Description = *b
			f.entries.Update(l)
		}
	}
	return nil
}

func (f *FastCache) writeBinFile(id string, r io.Reader) (int64, []byte, error) {
	binFile, err := f.buildFilename(id, BinaryExt)
	if err != nil {
		return 0, nil, err
	}

	w, err := os.Create(binFile)

	if err != nil {
		return 0, nil, err
	}
	size, err := w.ReadFrom(r)
	if err != nil {
		_ = w.Close()
		_ = os.Remove(binFile)
		return 0, nil, err
	}
	err = w.Close()
	if err != nil {
		return 0, nil, err
	}
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
	err := os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		return "", err
	}
	return filepath.Join(fp, fmt.Sprintf("%s%s", id, ext)), nil
}

// HasBlob checking, if a blob is present
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

// GetBlobDescription getting the description of the file
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

// RetrieveBlob retrieving the binary data from the storage system
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
			}
			err := f.getBlob(id, w)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return os.ErrNotExist
}

func (f *FastCache) getBlob(id string, w io.Writer) error {
	binFile, err := f.buildFilename(id, BinaryExt)
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

// DeleteBlob removing a blob from the storage system
func (f *FastCache) DeleteBlob(id string) error {
	if id == "" {
		return errEmptyIndex
	}
	if f.inBloom(id) {
		if f.entries.Has(id) {
			lid := f.entries.Delete(id)
			if lid != "" {
				err := f.deleteBlobFile(lid)
				if err != nil {
					logger.Errorf("error deleting file: %v", err)
					return err
				}
				f.bfDirty = true
			}
			return nil
		}
	}
	return os.ErrNotExist
}

// CheckBlob checking a single blob from the storage system
func (f *FastCache) CheckBlob(id string) (*model.CheckInfo, error) {
	return utils.CheckBlob(id, f)
}

func (f *FastCache) deleteBlobFile(id string) error {
	binFile, err := f.buildFilename(id, BinaryExt)
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

// SearchBlobs querying a single blob, niy
func (f *FastCache) SearchBlobs(_ string, _ func(id string) bool) error {
	return errNotImplemented
}

// Retention related methods

// GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returning a false
func (f *FastCache) GetAllRetentions(_ func(r model.RetentionEntry) bool) error {
	return errNotImplemented
}

// AddRetention adding a retention entry to the storage
func (f *FastCache) AddRetention(_ *model.RetentionEntry) error {
	return errNotImplemented
}

// GetRetention getting a single retention entry
func (f *FastCache) GetRetention(_ string) (model.RetentionEntry, error) {
	return model.RetentionEntry{}, errNotImplemented
}

// DeleteRetention deletes the retention entry from the storage
func (f *FastCache) DeleteRetention(_ string) error {
	return errNotImplemented
}

// ResetRetention resets the retention for a blob
func (f *FastCache) ResetRetention(_ string) error {
	return errNotImplemented
}

// GetLastError returning the last error (niy)
func (f *FastCache) GetLastError() error {
	return nil
}

// Close closing the storage
func (f *FastCache) Close() error {
	f.quit <- true
	return nil
}
