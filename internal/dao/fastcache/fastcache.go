package fastcache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	BINARY_EXT = ".bin"
	// this is the max size of a blob that will be stored into memory, if possible
	mffrs = 100 * 1024
)

type FastCache struct {
	RootPath   string // this is the root path for the file system storage
	MaxCount   int64
	MaxRamSize int64
	size       int64
	count      int64
	entries    []LRUEntry
}

type LRUEntry struct {
	lastAccess  time.Time
	description model.BlobDescription
	data        []byte
}

var _ interfaces.BlobStorageDao = &FastCache{}
var fastcache FastCache

// initialise this dao
func (f *FastCache) Init() error {
	f.count = 0
	f.size = 0

	err := f.removeContents(f.RootPath)
	if err != nil {
		return err
	}

	f.entries = make([]LRUEntry, 0)
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
	size, dat, err := f.writeBinFile(b.BlobID, r)
	if err != nil {
		return "", err
	}
	atomic.AddInt64(&f.size, size)
	f.entries = append(f.entries, LRUEntry{
		lastAccess:  time.Now(),
		description: *b,
		data:        dat,
	})
	f.handleContrains()
	return b.BlobID, nil
}

func (f *FastCache) handleContrains() error {
	if len(f.entries) > int(f.MaxCount) {
		// remove oldest entry
	}
	return nil
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
	if size < mffrs {
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
	return false, nil
}

// getting the description of the file
func (f *FastCache) GetBlobDescription(id string) (*model.BlobDescription, error) {
	return nil, nil
}

// retrieving the binary data from the storage system
func (f *FastCache) RetrieveBlob(id string, w io.Writer) error {
	return nil
}

// removing a blob from the storage system
func (f *FastCache) DeleteBlob(id string) error {
	return nil
}

//Retentionrelated methods
// for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (f *FastCache) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return nil
}
func (f *FastCache) AddRetention(r *model.RetentionEntry) error {
	return nil
}
func (f *FastCache) GetRetention(id string) (model.RetentionEntry, error) {
	return model.RetentionEntry{}, nil
}
func (f *FastCache) DeleteRetention(id string) error {
	return nil
}
func (f *FastCache) ResetRetention(id string) error {
	return nil
}

// closing the storage
func (f *FastCache) Close() error {
	return nil
}
