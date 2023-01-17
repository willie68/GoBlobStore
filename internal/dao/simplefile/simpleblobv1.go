// Package simplefile implemet a storage system on a mounted device with simple files as storage objects
package simplefile

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	// DescriptionExt extension of the description file
	DescriptionExt = ".json"
	// BinaryExt extension of the binary file
	BinaryExt = ".bin"
	// RetentionExt extension of the retention file
	RetentionExt = ".json"
	// RetentionPath path to the retention files
	RetentionPath = "retention"
)

func (s *SimpleFileBlobStorageDao) getBlobDescriptionV1(id string) (*model.BlobDescription, error) {
	var info model.BlobDescription
	jsonFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, DescriptionExt))
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

// updating the blob description
func (s *SimpleFileBlobStorageDao) updateBlobDescriptionV1(id string, b *model.BlobDescription) error {
	if s.hasBlobV1(id) {
		err := s.writeJSONFileV1(b)
		if err != nil {
			return err
		}
		s.cm.Lock()
		defer s.cm.Unlock()
		s.bdCch[b.BlobID] = *b
		return nil
	}
	return os.ErrNotExist
}

func (s *SimpleFileBlobStorageDao) hasBlobV1(id string) bool {
	binFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, BinaryExt))
	if _, err := os.Stat(binFile); os.IsNotExist(err) {
		return false
	}
	descFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, DescriptionExt))
	if _, err := os.Stat(descFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func (s *SimpleFileBlobStorageDao) getBlobV1(id string, w io.Writer) error {
	binFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, BinaryExt))
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
	fp = filepath.Join(fp, RetentionPath)
	err := os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		return "", err
	}
	return filepath.Join(fp, fmt.Sprintf("%s%s", id, RetentionExt)), nil
}

// TODO implement error handling
func (s *SimpleFileBlobStorageDao) deleteFilesV1(id string) error {
	binFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, BinaryExt))
	os.Remove(binFile)
	jsonFile := filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, DescriptionExt))
	os.Remove(jsonFile)
	jsonFile, _ = s.buildRetentionFilename(id)
	os.Remove(jsonFile)
	return nil
}

func (s *SimpleFileBlobStorageDao) writeJSONFileV1(b *model.BlobDescription) error {
	jsonFile, err := s.buildFilenameV1(b.BlobID, DescriptionExt)
	if err != nil {
		return err
	}

	json, err := json.Marshal(b)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(jsonFile, json, os.ModePerm)
	if err != nil {
		os.Remove(jsonFile)
		return err
	}

	return nil
}

func (s *SimpleFileBlobStorageDao) buildFilenameV1(id string, ext string) (string, error) {
	return filepath.Join(s.filepath, fmt.Sprintf("%s%s", id, ext)), nil
}
