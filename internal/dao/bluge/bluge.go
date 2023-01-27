// Package bluge this package contains all things related to the bluge fulltext index engine. see: https://github.com/blugelabs/bluge
package bluge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blugelabs/bluge"
	querystr "github.com/blugelabs/query_string"
	"github.com/pkg/errors"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
	"github.com/willie68/GoBlobStore/pkg/model/query"
)

// BlugeIndex name of the index engine
const BlugeIndex = "bluge"

var _ interfaces.Index = &Index{}
var _ interfaces.IndexBatch = &IndexBatch{}

// Index a tenant based single indexer
type Index struct {
	Tenant   string
	rootpath string
	config   bluge.Config
	wsync    sync.Mutex
	qsync    sync.Mutex
}

// IndexBatch for bulk indexing
type IndexBatch struct {
	docs  []model.BlobDescription
	index *Index
}

// Config the config for the indexer
type Config struct {
	Rootpath string `yaml:"rootpath"`
}

// DefaultConfig sets the default config
var (
	bcnfg Config
)

// InitBluge initialise the main engine, mainly retriving and storing the configuration
func InitBluge(p map[string]any) error {
	jsonStr, err := json.Marshal(p)
	if err != nil {
		log.Logger.Errorf("%v", err)
		return err
	}
	err = json.Unmarshal(jsonStr, &bcnfg)
	return err
}

// CloseBluge just for the sake of completeness
func CloseBluge() {
}

// Init initialise a index service for a tenant
func (m *Index) Init() error {
	m.Tenant = strings.ToLower(m.Tenant)
	m.rootpath = filepath.Join(bcnfg.Rootpath, m.Tenant, "_idx")
	err := os.MkdirAll(m.rootpath, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "Error init bluge")
	}
	m.config = bluge.DefaultConfig(m.rootpath)
	return err
}

// Search doing a search for a tenant
func (m *Index) Search(qry string, callback func(id string) bool) error {
	bq, err := m.buildQuery(qry)
	if err != nil {
		return err
	}
	reader, err := bluge.OpenReader(m.config)
	if err != nil {
		return err
	}
	defer reader.Close()
	request := bluge.NewTopNSearch(1000, bq).
		WithStandardAggregations()
	documentMatchIterator, err := reader.Search(context.Background(), request)
	if err != nil {
		return err
	}
	count := 0
	match, err := documentMatchIterator.Next()
	for err == nil && match != nil {
		err = match.VisitStoredFields(func(field string, value []byte) bool {
			if field == "_id" {
				callback(string(value))
			}
			return true
		})
		if err != nil {
			return err
		}
		match, err = documentMatchIterator.Next()
		count++
	}
	if err != nil {
		return err
	}
	return nil
}

func (m *Index) buildQuery(qry string) (bluge.Query, error) {
	var bq bluge.Query
	var err error
	if strings.HasPrefix(qry, "#") {
		qry = strings.TrimPrefix(qry, "#")
		bq, err = querystr.ParseQueryString(qry, querystr.DefaultOptions())
		if err != nil {
			return nil, err
		}
	} else {
		// parse query string to bluge query
		q, err := m.buildAST(qry)
		if err != nil {
			return nil, err
		}

		bq, err = toBlugeQuery(*q)
		if err != nil {
			return nil, err
		}
	}
	return bq, nil
}

func (m *Index) buildAST(q string) (*query.Query, error) {
	m.qsync.Lock()
	defer m.qsync.Unlock()
	query.N.Reset()
	res, err := query.Parse("query", []byte(q))
	if err != nil {
		return nil, err
	}
	qu, ok := res.(query.Query)
	if !ok {
		return nil, errors.New("unknown result")
	}
	return &qu, nil
}

// Index the index will index a single document, be aware this will only work in a single instance installation.
// the implementation will check, if the index writer is already opened and wait til it's closed, but only
// in a single instance of the blob storage. So in a multinode enviroment this will fail.
func (m *Index) Index(_ string, b model.BlobDescription) error {
	// index some data
	doc := m.toBlugeDoc(b)

	m.wsync.Lock()
	defer m.wsync.Unlock()
	writer, err := bluge.OpenWriter(m.config)
	if err != nil {
		return err
	}
	defer writer.Close()

	err = writer.Update(doc.ID(), doc)
	if err != nil {
		return err
	}
	return nil
}

func (m *Index) toBlugeDoc(b model.BlobDescription) bluge.Document {
	doc := bluge.NewDocument(b.BlobID)
	for k, i := range b.Map() {
		key := strings.TrimPrefix(k, config.Get().HeaderMapping[api.HeaderPrefixKey])
		switch v := i.(type) {
		case int:
			doc.AddField(bluge.NewNumericField(key, float64(v)).StoreValue())
		case []int:
			for _, y := range v {
				doc.AddField(bluge.NewNumericField(key, float64(y)).StoreValue())
			}
		case int64:
			doc.AddField(bluge.NewNumericField(key, float64(v)).StoreValue())
		case []int64:
			for _, y := range v {
				doc.AddField(bluge.NewNumericField(key, float64(y)).StoreValue())
			}
		case string:
			doc.AddField(bluge.NewTextField(key, v).StoreValue())
		case []string:
			for _, y := range v {
				doc.AddField(bluge.NewTextField(key, y).StoreValue())
			}
		case time.Time:
			doc.AddField(bluge.NewDateTimeField(key, v).StoreValue())
		case []time.Time:
			for _, y := range v {
				doc.AddField(bluge.NewDateTimeField(key, y).StoreValue())
			}
		default:
		}
	}
	return *doc
}

// NewBatch creating a new batch job for indexing
func (m *Index) NewBatch() interfaces.IndexBatch {
	return &IndexBatch{index: m}
}

// Add adding a description to the batch
func (i *IndexBatch) Add(id string, b model.BlobDescription) error {
	if id != b.BlobID {
		return fmt.Errorf(`ID "%s" is not equal to BlobID "%s" `, id, b.BlobID)
	}
	i.docs = append(i.docs, b)
	return nil
}

// Index indexing the batch
func (i *IndexBatch) Index() error {
	b := bluge.NewBatch()
	for _, bd := range i.docs {
		doc := i.index.toBlugeDoc(bd)
		b.Update(doc.ID(), doc)
	}

	i.index.wsync.Lock()
	defer i.index.wsync.Unlock()
	writer, err := bluge.OpenWriter(i.index.config)
	if err != nil {
		return err
	}
	defer writer.Close()

	err = writer.Batch(b)
	if err != nil {
		return err
	}
	i.docs = make([]model.BlobDescription, 0)
	return nil
}
