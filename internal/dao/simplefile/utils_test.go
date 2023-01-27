package simplefile

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	zipfile  = "../../../testdata/mig.zip"
	rootpath = "../../../testdata/sfbs"
	tenant   = "MCS"
)

func initTest(t *testing.T) {
	ast := assert.New(t)

	if _, err := os.Stat(rootpath); err == nil {
		err := os.RemoveAll(rootpath)
		ast.Nil(err)
	}
	// getting the zip file and extracting it into the file system
	err := os.MkdirAll(rootpath, os.ModePerm)
	ast.Nil(err)

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
			ast.Nil(err)
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

		_, err = io.Copy(dstFile, fileInArchive)
		ast.Nil(err)

		err = dstFile.Close()
		ast.Nil(err)

		err = fileInArchive.Close()
		ast.Nil(err)
	}
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
		_ = os.RemoveAll(filepath.Join(dir, name))
	}
	return nil
}
