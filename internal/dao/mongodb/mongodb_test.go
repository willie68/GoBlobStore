package mongodb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/pkg/model"
	"github.com/willie68/GoBlobStore/pkg/model/query"
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
		BlobID:        "123456789",
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      "test.txt",
		LastAccess:    time.Now().UnixMilli(),
		Retention:     180000,
		Properties:    make(map[string]interface{}),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = "MCS"

	err := idx.Index("123456789", b)
	ast.Nil(err)

	rets := make([]string, 0)
	err = idx.Search(`#{"$and": [{"tenant": "MCS"}, {"user": "Hallo"} ]}`, func(id string) bool {
		rets = append(rets, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(rets))
}

func TestQueryConvertion(t *testing.T) {
	ast := assert.New(t)

	str := `#{"$or": [{"$and": [{"field1": "Willie"}, {"field2": {"$gt": 100}}, {"field3": {"$not": {"$eq": "murks"}}}]}, {"$and": [{"field1": "Max"}, {"field2": {"$lte": 100}}, {"field3": {"$ne": "murks"}}]}]}`
	//#{"$or": [ {"$and": [ {"field1":"Willie"},{field2:>100}) OR (field1:"Max" AND field2:<=100))`

	q := query.Query{
		Condition: query.Node{
			Operator: query.OROP,
			Conditions: []interface{}{
				query.Node{
					Operator: query.ANDOP,
					Conditions: []interface{}{
						query.Condition{
							Field:    "field1",
							Operator: query.NO,
							Value:    "Willie",
						},
						query.Condition{
							Field:    "field2",
							Operator: query.GT,
							Value:    100,
						},
						query.Condition{
							Field:    "field3",
							Operator: query.EQ,
							Invert:   true,
							Value:    "murks",
						},
					},
				},
				query.Node{
					Operator: query.ANDOP,
					Conditions: []interface{}{
						query.Condition{
							Field:    "field1",
							Operator: query.NO,
							Value:    "Max",
						},
						query.Condition{
							Field:    "field2",
							Operator: query.LE,
							Value:    100,
						},
						query.Condition{
							Field:    "field3",
							Operator: query.NE,
							Value:    "murks",
						},
					},
				},
			},
		},
	}

	ast.NotNil(q)

	s := ToMongoQuery(q)
	fmt.Println(str)
	fmt.Println(s)
	ast.Equal(str, s)
}
