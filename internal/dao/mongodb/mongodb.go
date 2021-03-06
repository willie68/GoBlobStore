package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
	"github.com/willie68/GoBlobStore/pkg/model/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	driver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const MONGO_INDEX = "mongodb"

var _ interfaces.Index = &Index{}
var _ interfaces.IndexBatch = &IndexBatch{}

type Index struct {
	Tenant string
	col    driver.Collection
	qsync  sync.Mutex
}

type IndexBatch struct {
	docs  []model.BlobDescription
	index *Index
}

type Config struct {
	Hosts        []string `yaml:"hosts"`
	Database     string   `yaml:"database"`
	AuthDatabase string   `yaml:"authdatabase"`
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
}

type MongoBlobDescription struct {
	model.BlobDescription
	ID primitive.ObjectID `bson:"_id,omitempty"`
}

// DefaultConfig sets the default config
var (
	mcnfg    Config
	client   *driver.Client
	database *driver.Database
	ctx      context.Context
)

func InitMongoDB(p map[string]interface{}) error {
	jsonStr, err := json.Marshal(p)
	if err != nil {
		log.Logger.Errorf("%v", err)
		return err
	}
	json.Unmarshal(jsonStr, &mcnfg)
	if len(mcnfg.Hosts) == 0 {
		return errors.New("no mongo hosts found. check config")
	}
	rb := bson.NewRegistryBuilder()
	rb.RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{}))

	uri := fmt.Sprintf("mongodb://%s", mcnfg.Hosts[0])
	opts := options.Client().SetRegistry(rb.Build())
	opts.ApplyURI(uri)
	if mcnfg.Username != "" {
		opts.Auth = &options.Credential{
			Username:   mcnfg.Username,
			Password:   mcnfg.Password,
			AuthSource: mcnfg.AuthDatabase}
	}
	ctx = context.TODO()
	client, err = driver.Connect(ctx, opts)
	if err != nil {
		log.Logger.Errorf("%v", err)
		return err
	}

	database = client.Database(mcnfg.Database)
	return nil
}

func CloseMongoDB() {
	client.Disconnect(ctx)
}

func (m *Index) Init() error {
	m.Tenant = strings.ToLower(m.Tenant)
	m.col = *database.Collection("c_" + m.Tenant)
	// check for indexes
	idx := m.col.Indexes()
	opts := options.ListIndexes().SetMaxTime(2 * time.Second)
	cursor, err := idx.List(context.TODO(), opts)
	if err != nil {
		return err
	}
	var result []bson.M
	if err = cursor.All(context.TODO(), &result); err != nil {
		return err
	}
	found := false
	for _, i := range result {
		if strings.EqualFold(i["name"].(string), "blobid") {
			found = true
		}
	}
	if !found {
		log.Logger.Info("no index found, creating one")
		mod := driver.IndexModel{
			Keys: bson.M{
				"blobid": 1, // index in ascending order
			},
			Options: options.Index().SetUnique(true).SetName("blobid"),
		}
		// Create an Index using the CreateOne() method
		_, err := m.col.Indexes().CreateOne(ctx, mod)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Index) Search(query string, callback func(id string) bool) error {
	var bd bson.M

	if !strings.HasPrefix(query, "#") {
		// parse query string to Mongo query
		q, err := m.buildAST(query)
		if err != nil {
			return err
		}
		query = ToMongoQuery(*q)
	}

	query = strings.TrimPrefix(query, "#")
	query = strings.TrimSpace(query)
	err := bson.UnmarshalExtJSON([]byte(query), true, &bd)
	if err != nil {
		return err
	}

	if bd != nil {

		cur, err := m.col.Find(context.TODO(), bd, options.Find())
		if err != nil {
			return err
		}
		defer cur.Close(context.TODO())
		//Finding multiple documents returns a cursor
		//Iterate through the cursor allows us to decode documents one at a time
		for cur.Next(context.TODO()) {
			//Create a value into which the single document can be decoded
			elem := struct {
				BlobId string `bson:"blobid"`
			}{}
			err := cur.Decode(&elem)
			if err != nil {
				return err
			}
			ok := callback(elem.BlobId)
			if !ok {
				break
			}
		}

		if err := cur.Err(); err != nil {
			return err
		}

		//Close the cursor once finished
		return nil
	}
	return errors.New("no filter defined!")
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
	// checking if a blob with this id already exists
	ctx, can1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer can1()
	var result bson.M
	err := m.col.FindOne(ctx, bson.M{"blobid": id}).Decode(&result)
	if err != nil {
		if err == driver.ErrNoDocuments {
			// dosen't exists, create
			var bd bson.D
			bd = append(bd, bson.E{Key: "blobid", Value: id})
			for k, v := range b.Map() {
				key := strings.TrimPrefix(k, config.Get().HeaderMapping[api.HeaderPrefixKey])
				bd = append(bd, bson.E{Key: key, Value: v})
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			res, err := m.col.InsertOne(ctx, bd)
			if err != nil {
				return err
			}
			mid := res.InsertedID
			log.Logger.Infof("insert: %v", mid)
			return nil
		}
		fmt.Printf("err: %v", err)
		return err
	} else {
		// update with new description
		var bd bson.D
		bd = append(bd, bson.E{Key: "blobid", Value: id})
		for k, v := range b.Map() {
			key := strings.TrimPrefix(k, config.Get().HeaderMapping[api.HeaderPrefixKey])
			bd = append(bd, bson.E{Key: key, Value: v})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := m.col.ReplaceOne(ctx, bson.M{"blobid": id}, bd)
		if err != nil {
			return err
		}
		mid := res.ModifiedCount
		log.Logger.Infof("mod count: %vd", mid)
		return nil

	}
	// return errors.New("blob already exists")
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
	for _, bd := range i.docs {
		err := i.index.Index(bd.BlobID, bd)
		if err != nil {
			return err
		}
	}
	i.docs = make([]model.BlobDescription, 0)
	return nil
}

func ToMongoQuery(q query.Query) string {
	var b strings.Builder
	b.WriteString("#")
	c := q.Condition
	b.WriteString(xToMdb(c))
	return b.String()
}

//cToMdb converting a condition into a mongo query string
func cToMdb(c query.Condition) string {
	var b strings.Builder
	f := c.Field
	cv := oToMdb(c)
	if c.Invert {
		cv = fmt.Sprintf(`{"$not": %s}`, cv)
	}
	b.WriteString(fmt.Sprintf(`{"%s": %s}`, f, cv))
	return b.String()
}

//oToMdb converting the operator part of a condition into a mongo query string
func oToMdb(c query.Condition) string {
	v := c.VtoS()
	switch c.Operator {
	case query.NO:
		return v
	case query.EQ:
		return fmt.Sprintf(`{"$eq": %s}`, v)
	case query.LT:
		return fmt.Sprintf(`{"$lt": %s}`, v)
	case query.LE:
		return fmt.Sprintf(`{"$lte": %s}`, v)
	case query.GT:
		return fmt.Sprintf(`{"$gt": %s}`, v)
	case query.GE:
		return fmt.Sprintf(`{"$gte": %s}`, v)
	case query.NE:
		return fmt.Sprintf(`{"$ne": %s}`, v)
	}
	return ""
}

//nToMdb converting a node into a mongo query string
func nToMdb(n query.Node) string {
	var b strings.Builder
	op := fmt.Sprintf("$%s", strings.ToLower(string(n.Operator)))
	wh := xsToMdb(n.Conditions)
	b.WriteString(fmt.Sprintf(`{"%s": [%s]}`, op, wh))
	return b.String()
}

//xsToMdb converting an array of nodes/conditions to a mongo json string
func xsToMdb(xs []interface{}) string {
	var b strings.Builder
	for i, x := range xs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(xToMdb(x))
	}
	return b.String()
}

//xToMdb converting a node/condition to a mongo json string
func xToMdb(x interface{}) string {
	switch v := x.(type) {
	case query.Condition:
		return cToMdb(v)
	case *query.Condition:
		return cToMdb(*v)
	case query.Node:
		return nToMdb(v)
	case *query.Node:
		return nToMdb(*v)
	}
	return ""
}
