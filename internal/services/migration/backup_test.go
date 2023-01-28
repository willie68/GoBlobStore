package migration

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/internal/services/simplefile"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	testdata = "../../../testdata"
	zipfile  = testdata + "/mig.zip"
	rootpath = testdata + "/bck/blobstorage"
	bckpath  = testdata + "/bck/bckstorage"
	migTnt   = "migtnt"
)

type MockStorage struct {
	mock.Mock
}

func initBckTest(t *testing.T) {
	err := os.RemoveAll(rootpath)
	assert.Nil(t, err)
	err = os.MkdirAll(rootpath, os.ModePerm)
	assert.Nil(t, err)

	err = os.RemoveAll(bckpath)
	assert.Nil(t, err)
	err = os.MkdirAll(bckpath, os.ModePerm)
	assert.Nil(t, err)

	// getting the zip file and extracting it into the file system
	archive, err := zip.OpenReader(zipfile)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(rootpath, f.Name)

		if !strings.HasPrefix(filePath, filepath.Clean(rootpath)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, os.ModePerm)
			assert.Nil(t, err)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		err = dstFile.Close()
		assert.Nil(t, err)
		err = fileInArchive.Close()
		assert.Nil(t, err)
	}
}

func getBlobCount(stg interfaces.BlobStorage) (int, error) {
	count := 0
	err := stg.GetBlobs(func(id string) bool {
		count++
		return true
	})
	return count, err
}

func getRntCount(stg interfaces.BlobStorage) (int, error) {
	count := 0
	err := stg.GetAllRetentions(func(r model.RetentionEntry) bool {
		count++
		return true
	})
	return count, err
}

func TestSyncForward(t *testing.T) {
	initBckTest(t)

	ast := assert.New(t)
	mainStg := &simplefile.BlobStorage{
		RootPath: rootpath,
		Tenant:   migTnt,
	}
	err := mainStg.Init()
	ast.Nil(err)
	ast.NotNil(mainStg)

	count, err := getBlobCount(mainStg)
	ast.Nil(err)
	ast.Equal(7, count)

	count, err = getRntCount(mainStg)
	ast.Nil(err)
	ast.Equal(7, count)

	bckStg := &simplefile.BlobStorage{
		RootPath: bckpath,
		Tenant:   migTnt,
	}
	err = bckStg.Init()
	ast.Nil(err)
	ast.NotNil(bckStg)

	count, err = getBlobCount(bckStg)
	ast.Nil(err)
	ast.Equal(0, count)

	count, err = getRntCount(bckStg)
	ast.Nil(err)
	ast.Equal(0, count)

	err = migrateBckTnt(mainStg, bckStg)
	ast.Nil(err)
	wg.Wait()

	count, err = getBlobCount(mainStg)
	ast.Nil(err)
	ast.Equal(7, count)

	count, err = getBlobCount(bckStg)
	ast.Nil(err)
	ast.Equal(7, count)

	count, err = getRntCount(bckStg)
	ast.Nil(err)
	ast.Equal(7, count)
}
