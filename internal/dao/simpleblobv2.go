package dao

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/willie68/GoBlobStore/pkg/model"
)

func (s *SimpleFileBlobStorageDao) getBlobsV2(offset int, limit int) ([]string, error) {
	var files []string
	err := filepath.Walk(s.filepath, func(path string, info os.FileInfo, err error) error {
		if !strings.Contains(path, "retention") && strings.HasSuffix(path, ".json") {
			files = append(files, info.Name()[:len(info.Name())-5])
		}
		if len(files) >= limit {
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return nil, err
	}
	return files, nil
}

func (s *SimpleFileBlobStorageDao) getBlobDescriptionV2(id string) (*model.BlobDescription, error) {
	return nil, os.ErrNotExist
}
