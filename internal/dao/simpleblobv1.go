package dao

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	DESCRIPTION_EXT = ".json"
	BINARY_EXT      = ".bin"
	RETENTION_EXT   = ".json"
	RETENTION_PATH  = "retention"
)

func (s *SimpleFileBlobStorageDao) getBlobDescriptionV1(id string) (*model.BlobDescription, error) {
	var info model.BlobDescription
	jsonFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, DESCRIPTION_EXT))
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	dat, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(dat, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *SimpleFileBlobStorageDao) getBlobV1(id string, w io.Writer) error {
	binFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, BINARY_EXT))
	if _, err := os.Stat(binFile); os.IsNotExist(err) {
		return os.ErrNotExist
	}
	f, err := os.Open(binFile)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	if err != nil {
		return err
	}
	return nil
}

func (s *SimpleFileBlobStorageDao) buildRetentionFilename(id string) (string, error) {
	fp := s.filepath
	fp = filepath.Join(fp, RETENTION_PATH)
	err := os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		return "", err
	}
	return filepath.Join(fp, fmt.Sprintf("%s%s", id, RETENTION_EXT)), nil
}

//TODO implement error handling
func (s *SimpleFileBlobStorageDao) deleteFilesV1(id string) error {
	binFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, BINARY_EXT))
	os.Remove(binFile)
	jsonFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, DESCRIPTION_EXT))
	os.Remove(jsonFile)
	jsonFile, _ = s.buildRetentionFilename(id)
	os.Remove(jsonFile)
	return nil
}
