package mongo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/pkg/model"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var cnfg map[string]interface{}

func InitT(ast *assert.Assertions) {
	cnfg = make(map[string]interface{})
	cnfg["hosts"] = []string{"127.0.0.1:27017"}
	cnfg["username"] = "blobstore"
	cnfg["password"] = "blobstore"
	cnfg["authdatabase"] = "blobstore"
	cnfg["database"] = "blobstore"

	err := InitMongoDB(cnfg)
	ast.Nil(err)
	ast.NotNil(client)
	ast.NotNil(database)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = client.Ping(ctx, readpref.Primary())
	ast.Nil(err)
}

func TestMongoConnect(t *testing.T) {

	ast := assert.New(t)

	InitT(ast)

	idx := Index{
		Tenant: "MCS",
	}
	idx.Init()
	ast.NotNil(idx.col)

	b := model.BlobDescription{
		BlobID:        "12345678",
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
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = "MCS"

	err := idx.Index("12345678", b)
	ast.Nil(err)
	rets := make([]string, 0)
	err = idx.Search("X-tenant: \"MCS\"", func(id string) bool {
		rets = append(rets, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(rets))
}
