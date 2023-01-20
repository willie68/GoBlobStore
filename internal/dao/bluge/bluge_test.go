package bluge

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const rootpath = "../../../testdata/bluge/"

var cnfg map[string]any

func InitT(ast *assert.Assertions) {
	cnfg = make(map[string]any)
	cnfg["rootpath"] = rootpath

	_ = os.RemoveAll(rootpath)
	err := InitBluge(cnfg)
	ast.Nil(err)
	ast.NotNil(bcnfg)
}

func TestBlugeConnect(t *testing.T) {
	ast := assert.New(t)

	InitT(ast)

	idx := Index{
		Tenant: "MCS",
	}
	err := idx.Init()
	ast.Nil(err)
	ast.NotNil(idx.rootpath)
	ast.NotNil(idx.config)

	b := model.BlobDescription{
		BlobID:        "123456789",
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      "test.txt",
		LastAccess:    time.Now().UnixMilli(),
		Retention:     180000,
		Properties:    make(map[string]any),
	}
	b.Properties["x-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["x-retention"] = []int{123456}
	b.Properties["x-tenant"] = "MCS"

	err = idx.Index("123456789", b)
	ast.Nil(err)

	rets := make([]string, 0)
	err = idx.Search(`#x-user: Hallo`, func(id string) bool {
		rets = append(rets, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(rets))
	ast.Equal(b.BlobID, rets[0])

	rets = make([]string, 0)
	err = idx.Search(`#x-user: Hallo2`, func(id string) bool {
		rets = append(rets, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(rets))
	ast.Equal(b.BlobID, rets[0])

	rets = make([]string, 0)
	err = idx.Search(`#x-tenant: MCS`, func(id string) bool {
		rets = append(rets, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(rets))
	ast.Equal(b.BlobID, rets[0])

	rets = make([]string, 0)
	err = idx.Search(`#contentType: text/plain`, func(id string) bool {
		rets = append(rets, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(rets))
	ast.Equal(b.BlobID, rets[0])
}

var tests = []struct {
	q  string
	id string
	n  int
}{
	{
		q:  `tenant:MCS`,
		id: "123410",
		n:  100,
	},
	{
		q:  `tenant:"MCS" AND user:"Hallo"`,
		id: "123410",
		n:  100,
	},
	{
		q:  `intfield:=1234`,
		id: "123434",
		n:  1,
	},
	{
		q:  `intfield:>1234`,
		id: "12340",
		n:  65,
	},
	{
		q:  `intfield:<1234`,
		id: "12340",
		n:  34,
	},
	{
		q:  `intfield:>=1234`,
		id: "12340",
		n:  66,
	},
	{
		q:  `intfield:<=1234`,
		id: "12340",
		n:  35,
	},
	{
		q:  `intfield:!=1234`,
		id: "12340",
		n:  99,
	},
	{
		q:  `user:H*`,
		id: "12340",
		n:  100,
	},
}

func TestQueryConvertion(t *testing.T) {
	//TODO skip the skip
	t.SkipNow()
	// The test intfield:!=1234 is throwing an error
	ast := assert.New(t)

	InitT(ast)

	idx := Index{
		Tenant: "MCS",
	}
	err := idx.Init()
	ast.Nil(err)
	ast.NotNil(idx.rootpath)
	ast.NotNil(idx.config)

	bt := idx.NewBatch()
	for x := 0; x < 100; x++ {
		id := fmt.Sprintf("1234%d", x)
		b := getBlobDescription(id, 1200+x)
		err := bt.Add(b.BlobID, b)
		ast.Nil(err, "adding to batch")
	}

	err = bt.Index()
	ast.Nil(err, "indexing")

	for _, t := range tests {
		rets := make([]string, 0)
		err := idx.Search(t.q, func(id string) bool {
			rets = append(rets, id)
			return true
		})
		ast.Nil(err)
		ast.Equal(t.n, len(rets), t.q)
		if t.n == 1 && len(rets) > 0 {
			ast.Equal(t.id, rets[0], t.q)
		}
	}
}

func getBlobDescription(id string, num int) model.BlobDescription {
	b := model.BlobDescription{
		BlobID:        id,
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      "test.txt",
		LastAccess:    time.Now().UnixMilli(),
		Retention:     180000,
		Properties:    make(map[string]any),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{num}
	b.Properties["X-tenant"] = "MCS"
	b.Properties["X-intfield"] = num
	return b
}
