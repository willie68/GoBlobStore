package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"bitbucket.easy.de/dm/service-blobstore-go/internal/config"
	log "bitbucket.easy.de/dm/service-blobstore-go/internal/logging"
	"bitbucket.easy.de/dm/service-blobstore-go/internal/services/simplefile"
	flag "github.com/spf13/pflag"
	"github.com/willie68/GoBlobStore/internal/services"
)

var (
	configFile string
	stgPath    string
)

func init() {
	// variables for parameter override
	flag.StringVarP(&configFile, "config", "c", "", "this is the path and filename to the config file")
}

func main() {
	fmt.Println("start migration")
	flag.Parse()
	fmt.Printf("config: %s\r\n", configFile)
	config.File = configFile
	config.Load()

	stgCnf := config.Get().Engine.Storage
	services.Init(config.Get().Engine)
	stgPath = stgCnf.Properties["rootpath"].(string)
	fmt.Printf("storagepath: %s", stgPath)

	fmt.Println("creating tenants")
	dirs, err := os.ReadDir(stgPath)
	if err != nil {
		fmt.Printf("error scanning dir: %s %v\r\n", stgPath, err)
		panic(1)
	}
	fmt.Println("found dirs")
	tenants := make([]string, 0)
	for _, dir := range dirs {
		if dir.IsDir() {
			name := dir.Name()
			if strings.HasPrefix(name, "_") {
				continue
			}
			if strings.Contains(name, "-") {
				continue
			}
			tenants = append(tenants, name)
		}
	}

	fmt.Printf("found %d tenants\r\n", len(tenants))
	fmt.Println("checking files")

	for _, tnt := range tenants {
		checkTenant(tnt)
	}

}

func checkTenant(tenant string) {
	ids := make([]string, 0)
	rootPath := filepath.Join(stgPath, tenant)
	filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".json") {
			id := strings.TrimSuffix(d.Name(), ".json")
			ids = append(ids, id)
		}
		return nil
	})
	log.Logger.Infof("ids for tenant: %s, len: %d\r\n", tenant, len(ids))

	srv := simplefile.SimpleFileBlobStorage{
		RootPath: stgPath,
		Tenant:   tenant,
	}
	err := srv.Init()
	if err != nil {
		fmt.Printf("init error: %v \r\n", err)
		return
	}
	errIds := make([]string, 0)
	logline := ""
	for _, id := range ids {
		logline += fmt.Sprintf("%s -> ", id)
		ok, err := srv.HasBlob(id)
		if err != nil {
			logline += fmt.Sprintf("error: %v", err)
		}
		if ok {
			logline += "ok"
			_, err := srv.GetBlobDescription(id)
			if err != nil {
				logline += fmt.Sprintf("db error: %v", err)
			}
		} else {
			logline += "not found"
			errIds = append(errIds, id)
		}
		if !ok {
			log.Logger.Error(logline)
		}
	}
	if len(errIds) == 0 {
		log.Logger.Infof("no errors found for tenant: %s\r\n", tenant)
	}
}
