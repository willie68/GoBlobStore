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

	rets = make([]string, 0)
	err = idx.Search(`#x-user: Hallo2`, func(id string) bool {
		rets = append(rets, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(rets))
}

/*
func TestQueryConvertion(t *testing.T) {
	ast := assert.New(t)

	str := `#{"$or": [{"$and": [{"field1": "Willie"}, {"field2": {"$gt": 100}}, {"field3": {"$not": {"$eq": "murks"}}}]}, {"$and": [{"field1": "Max"}, {"field2": {"$lte": 100}}, {"field3": {"$ne": "murks"}}]}]}`
	//#{"$or": [ {"$and": [ {"field1":"Willie"},{field2:>100}) OR (field1:"Max" AND field2:<=100))`

	q := model.Query{
		Condition: model.Node{
			Operator: model.OROP,
			Conditions: []interface{}{
				model.Node{
					Operator: model.ANDOP,
					Conditions: []interface{}{
						model.Condition{
							Field:    "field1",
							Operator: model.NO,
							Value:    "Willie",
						},
						model.Condition{
							Field:    "field2",
							Operator: model.GT,
							Value:    100,
						},
						model.Condition{
							Field:    "field3",
							Operator: model.EQ,
							Invert:   true,
							Value:    "murks",
						},
					},
				},
				model.Node{
					Operator: model.ANDOP,
					Conditions: []interface{}{
						model.Condition{
							Field:    "field1",
							Operator: model.NO,
							Value:    "Max",
						},
						model.Condition{
							Field:    "field2",
							Operator: model.LE,
							Value:    100,
						},
						model.Condition{
							Field:    "field3",
							Operator: model.NE,
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
*/
