package dao

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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
	var info model.BlobDescription
	fp := s.filepath
	fp = filepath.Join(fp, id[:2])
	fp = filepath.Join(fp, id[2:4])
	jsonFile := filepath.Join(fp, fmt.Sprintf("%s.json", id))

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

func (s *SimpleFileBlobStorageDao) storeBlobV2(b *model.BlobDescription, f io.Reader) (string, error) {
	uuid := uuid.NewString()
	b.BlobID = uuid
	size, err := s.writeBinFileV2(uuid, f)
	if err != nil {
		return "", err
	}
	if (b.ContentLength > 0) && b.ContentLength != size {
		s.deleteFileV2(uuid)
		return "", fmt.Errorf("wrong content length %d=%d", b.ContentLength, size)
	}
	b.ContentLength = size
	err = s.writeJsonFileV2(b)
	if err != nil {
		s.deleteFileV2(uuid)
		return "", err
	}

	if b.Retention > 0 {
		err = s.writeRetentionFile(b)
		if err != nil {
			s.deleteFileV2(uuid)
			return "", err
		}
	}

	return uuid, nil
}

func (s *SimpleFileBlobStorageDao) writeBinFileV2(id string, r io.Reader) (int64, error) {
	binFile, err := s.buildFilename(id, ".bin")
	if err != nil {
		return 0, err
	}

	f, err := os.Create(binFile)

	if err != nil {
		return 0, err
	}
	size, err := f.ReadFrom(r)
	if err != nil {
		f.Close()
		os.Remove(binFile)
		return 0, err
	}
	f.Close()

	return size, nil
}

//TODO implement error handling
func (s *SimpleFileBlobStorageDao) deleteFileV2(id string) error {
	binFile, _ := s.buildFilename(id, ".bin")
	os.Remove(binFile)
	jsonFile, _ := s.buildFilename(id, ".json")
	os.Remove(jsonFile)
	return nil
}

func (s *SimpleFileBlobStorageDao) writeJsonFileV2(b *model.BlobDescription) error {
	jsonFile, err := s.buildFilename(b.BlobID, ".json")
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

func (s *SimpleFileBlobStorageDao) writeRetentionFile(b *model.BlobDescription) error {
	jsonFile, err := s.buildRetentionFilename(b.BlobID)
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

func (s *SimpleFileBlobStorageDao) buildFilename(id string, ext string) (string, error) {
	fp := s.filepath
	fp = filepath.Join(fp, id[:2])
	fp = filepath.Join(fp, id[2:4])
	err := os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		return "", err
	}
	return filepath.Join(fp, fmt.Sprintf("%s%s", id, ext)), nil
}
