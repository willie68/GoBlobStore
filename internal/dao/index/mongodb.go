package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	driver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const MONGO_INDEX = "mongodb"

var _ interfaces.Index = &MongoIndex{}

var MongoCnfg config.Storage

type MongoIndex struct {
	Tenant string
}

type MongoConfig struct {
	Hosts        []string `yaml:"hosts"`
	Database     string   `yaml:"database"`
	AuthDatabase string   `yaml:"authdatabase"`
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
}

// DefaultConfig sets the default config
var mcnfg MongoConfig

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
	ctx := context.TODO()
	client, err := driver.Connect(ctx, opts)
	if err != nil {
		log.Logger.Errorf("%v", err)
		return err
	}
	defer client.Disconnect(ctx)

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Logger.Errorf("%v", err)
		return err
	}
	fmt.Println(databases)
	return nil
}

func (m *MongoIndex) Init() error {
	return errors.New("not implemented yet")
}

func (m *MongoIndex) Search(query string, callback func(id string) bool) error {
	return errors.New("not implemented yet")
}

func (m *MongoIndex) Index(id string, b model.BlobDescription) error {
	return errors.New("not implemented yet")
}
