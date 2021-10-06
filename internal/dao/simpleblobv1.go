package dao

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/willie68/GoBlobStore/pkg/model"
)

func (s *SimpleFileBlobStorageDao) getBlobDescriptionV1(id string) (*model.BlobDescription, error) {
	var info model.BlobDescription
	jsonFile := filepath.Join(s.filepath, fmt.Sprintf("%s.json", id))

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
