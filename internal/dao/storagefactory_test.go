package dao

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	rootFilePrefix = "R:/mainstg"
	//rootFilePrefix = "../../testdata/main"
	tntcount = 1000
	blbcount = tntcount * 10
)

var (
	blbPath = filepath.Join(rootFilePrefix, "blbstg")
	cchPath = filepath.Join(rootFilePrefix, "blbcch")
	bckPath = filepath.Join(rootFilePrefix, "bckstg")
	main    interfaces.BlobStorageDao
)

func initTest(_ *testing.T) {
	config := config.Engine{
		RetentionManager: "SingleRetention",
		Tenantautoadd:    true,
		BackupSyncmode:   false,
		AllowTntBackup:   false,
		Storage: config.Storage{
			Storageclass: "SimpleFile",
			Properties: map[string]interface{}{
				"rootpath": blbPath,
			},
		},
	}
	Init(config)
}

func clear(_ *testing.T) {
	removeContents(rootFilePrefix)
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		os.RemoveAll(filepath.Join(dir, name))
	}
	return nil
}

func createBlobDescription(id string, tenant string) model.BlobDescription {
	uuid := utils.GenerateID()

	b := model.BlobDescription{
		BlobID:        uuid,
		StoreID:       tenant,
		TenantID:      tenant,
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      fmt.Sprintf("test_%s.txt", id),
		LastAccess:    time.Now().UnixMilli(),
		Retention:     180000,
		Properties:    make(map[string]interface{}),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = tenant
	b.Properties["X-id"] = uuid
	return b
}

func TestManyTenants(t *testing.T) {
	log.Logger.SetLevel(log.Info)
	clear(t)
	initTest(t)
	ast := assert.New(t)
	stgf, err := GetStorageFactory()
	ast.Nil(err)
	ast.NotNil(stgf)
	tntDao, err := GetTenantDao()
	ast.Nil(err)
	ast.NotNil(tntDao)

	fmt.Println("create tenants")
	tnts := make([]string, 0)
	for i := 0; i < tntcount; i++ {
		tnt := randTenantName()
		tnts = append(tnts, tnt)
		err = tntDao.AddTenant(tnt)
		ast.Nil(err)
	}

	fmt.Println("store blobs")
	ids := make(map[string]string, 0)
	for i := 0; i < blbcount; i++ {
		tnt := tnts[rand.Intn(len(tnts))]
		bd := createBlobDescription(strconv.Itoa(i), tnt)
		r := strings.NewReader("this is a blob content")
		dao, err := stgf.GetStorageDao(tnt)
		ast.Nil(err)
		ast.NotNil(dao)
		id, err := dao.StoreBlob(&bd, r)
		ast.Nil(err)
		ast.NotEmpty(id)
		ids[id] = tnt
	}

	fmt.Println("check blobs")
	for id, tnt := range ids {
		dao, err := stgf.GetStorageDao(tnt)
		ast.Nil(err)
		ast.NotNil(dao)
		ok, err := dao.HasBlob(id)
		ast.Nil(err)
		ast.True(ok, "blob not found tenant:%s, id: %s", tnt, id)
	}

	fmt.Println("delete tenants")
	for _, tnt := range tnts {
		id, err := tntDao.RemoveTenant(tnt)
		ast.Nil(err)
		ast.Empty(id)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890_")

func randTenantName() string {
	b := make([]rune, 12)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
