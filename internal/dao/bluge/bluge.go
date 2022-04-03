package bluge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blugelabs/bluge"
	querystr "github.com/blugelabs/query_string"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const BLUGE_INDEX = "bluge"

var _ interfaces.Index = &Index{}

type Index struct {
	Tenant   string
	rootpath string
	config   bluge.Config
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

func (m *Index) Index(id string, b model.BlobDescription) error {
	writer, err := bluge.OpenWriter(m.config)
	if err != nil {
		return err
	}
	defer writer.Close()
	// index some data
	doc := bluge.NewDocument(b.BlobID).
		AddField(bluge.NewTextField("StoreID", b.StoreID).StoreValue()).
		AddField(bluge.NewNumericField("ContentLength", float64(b.ContentLength)).StoreValue()).
		AddField(bluge.NewTextField("ContentType", b.ContentType).StoreValue()).
		AddField(bluge.NewDateTimeField("CreationDate", time.Unix(int64(b.CreationDate), 0))).
		AddField(bluge.NewTextField("Filename", b.Filename).StoreValue()).
		AddField(bluge.NewTextField("TenantID", b.TenantID).StoreValue()).
		AddField(bluge.NewTextField("BlobID", b.BlobID).StoreValue()).
		AddField(bluge.NewDateTimeField("LastAccess", time.Unix(int64(b.LastAccess), 0))).
		AddField(bluge.NewNumericField("Retention", float64(b.Retention)).StoreValue()).
		AddField(bluge.NewTextField("BlobURL", b.BlobURL).StoreValue()).
		AddField(bluge.NewTextField("Hash", b.Hash).StoreValue())
	for k, i := range b.Properties {
		switch v := i.(type) {
		case int:
			doc.AddField(bluge.NewNumericField(k, float64(v)).StoreValue())
		case []int:
			for _, y := range v {
				doc.AddField(bluge.NewNumericField(k, float64(y)).StoreValue())
			}
		case int64:
			doc.AddField(bluge.NewNumericField(k, float64(v)).StoreValue())
		case []int64:
			for _, y := range v {
				doc.AddField(bluge.NewNumericField(k, float64(y)).StoreValue())
			}
		case string:
			doc.AddField(bluge.NewTextField(k, v).StoreValue())
		case []string:
			for _, y := range v {
				doc.AddField(bluge.NewTextField(k, y).StoreValue())
			}
		default:
		}
	}

	err = writer.Update(doc.ID(), doc)
	if err != nil {
		return err
	}
	return nil
}
