package bluge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/pkg/model"
)

var cnfg map[string]interface{}

func InitT(ast *assert.Assertions) {
	cnfg = make(map[string]interface{})
	cnfg["rootpath"] = "r:\\blbstg\\"

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
	idx.Init()
	ast.NotNil(idx.rootpath)
	ast.NotNil(idx.config)

	b := model.BlobDescription{
		BlobID:        "123456789",
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		Filename:      "test.txt",
		LastAccess:    int(time.Now().UnixNano() / 1000000),
		Retention:     180000,
		Properties:    make(map[string]interface{}),
	}
	b.Properties["x-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["x-retention"] = []int{123456}
	b.Properties["x-tenant"] = "MCS"

	err := idx.Index("123456789", b)
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
	q string
	n int
}{
	{
		q: `x-tenant:"MCS"`,
		n: 1,
	},
	{
		q: `x-tenant:"MCS" AND x-user:"Hallo"`,
		n: 1,
	},
	{
		q: `x-intfield:=1234`,
		n: 1,
	},
	{
		q: `x-intfield:>1234`,
		n: 0,
	},
	{
		q: `x-intfield:<1234`,
		n: 0,
	},
	{
		q: `x-intfield:>=1234`,
		n: 1,
	},
	{
		q: `x-intfield:<=1234`,
		n: 1,
	},
	{
		q: `x-intfield:!=1234`,
		n: 0,
	},
}

func TestQueryConvertion(t *testing.T) {
	ast := assert.New(t)

	idx := Index{
		Tenant: "MCS",
	}
	idx.Init()
	ast.NotNil(idx.rootpath)
	ast.NotNil(idx.config)

	b := getBlobDescription()

	err := idx.Index(b.BlobID, b)
	ast.Nil(err)

	for _, t := range tests {
		rets := make([]string, 0)
		err = idx.Search(t.q, func(id string) bool {
			rets = append(rets, id)
			return true
		})
		ast.Nil(err)
		ast.Equal(t.n, len(rets), t.q)
		if t.n == 1 && len(rets) > 0 {
			ast.Equal(b.BlobID, rets[0])
		}
	}
}

func getBlobDescription() model.BlobDescription {
	b := model.BlobDescription{
		BlobID:        "123456789",
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		Filename:      "test.txt",
		LastAccess:    int(time.Now().UnixNano() / 1000000),
		Retention:     180000,
		Properties:    make(map[string]interface{}),
	}
	b.Properties["x-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["x-retention"] = []int{123456}
	b.Properties["x-tenant"] = "MCS"
	b.Properties["x-intfield"] = 1234
	return b
}
