package mongo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	driver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const MONGO_INDEX = "mongodb"

var _ interfaces.Index = &Index{}

var MongoCnfg config.Storage

type Index struct {
	Tenant string
	col    driver.Collection
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
		mod := mongo.IndexModel{
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
	if strings.HasPrefix(query, "#") {
		query = strings.TrimPrefix(query, "#")
		query = strings.TrimSpace(query)
		var bd bson.M
		err := bson.UnmarshalExtJSON([]byte(query), true, &bd)
		if err != nil {
			return err
		}
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

	return errors.New("not implemented yet")
}

func (m *Index) Index(id string, b model.BlobDescription) error {
	// checking if a blob with this id already exists
	ctx, can1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer can1()
	var result bson.M
	err := m.col.FindOne(ctx, bson.M{"blobid": id}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// dosen't exists, create
			var bd bson.D
			bd = append(bd, bson.E{"blobid", id})
			for k, v := range b.Map() {
				bd = append(bd, bson.E{strings.ToLower(k), v})
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
		bd = append(bd, bson.E{"blobid", id})
		for k, v := range b.Map() {
			bd = append(bd, bson.E{strings.ToLower(k), v})
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
	return errors.New("blob already exists")
}
