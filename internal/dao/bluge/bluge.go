package bluge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blugelabs/bluge"
	querystr "github.com/blugelabs/query_string"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
	"github.com/willie68/GoBlobStore/pkg/model/query"
)

const BLUGE_INDEX = "bluge"

var _ interfaces.Index = &Index{}
var _ interfaces.IndexBatch = &IndexBatch{}

type Index struct {
	Tenant   string
	rootpath string
	config   bluge.Config
	wsync    sync.Mutex
	qsync    sync.Mutex
}

type IndexBatch struct {
	docs  []model.BlobDescription
	index *Index
}

type Config struct {
	Rootpath string `yaml:"rootpath"`
}

// DefaultConfig sets the default config
var (
	bcnfg Config
)

func InitBluge(p map[string]interface{}) error {
	jsonStr, err := json.Marshal(p)
	if err != nil {
		log.Logger.Errorf("%v", err)
		return err
	}
	json.Unmarshal(jsonStr, &bcnfg)
	return nil
}

func CloseBluge() {
}

func (m *Index) Init() error {
	m.Tenant = strings.ToLower(m.Tenant)
	m.rootpath = filepath.Join(bcnfg.Rootpath, m.Tenant, "_idx")
	os.MkdirAll(m.rootpath, os.ModePerm)
	m.config = bluge.DefaultConfig(m.rootpath)
	return nil
}

func (m *Index) Search(query string, callback func(id string) bool) error {
	var bq bluge.Query
	var err error
	if strings.HasPrefix(query, "#") {
		query = strings.TrimPrefix(query, "#")
		bq, err = querystr.ParseQueryString(query, querystr.DefaultOptions())
		if err != nil {
			return err
		}
	} else {
		// parse query string to Mongo query
		q, err := m.buildAST(query)
		if err != nil {
			return err
		}

		bq, err = toBlugeQuery(*q)
		if err != nil {
			return err
		}
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

func (m *Index) Index(id string, b model.BlobDescription) error {
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

func (m *Index) NewBatch() interfaces.IndexBatch {
	return &IndexBatch{index: m}
}

func (i *IndexBatch) Add(id string, b model.BlobDescription) error {
	if id != b.BlobID {
		return fmt.Errorf(`ID "%s" is not equal to BlobID "%s" `, id, b.BlobID)
	}
	i.docs = append(i.docs, b)
	return nil
}

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
