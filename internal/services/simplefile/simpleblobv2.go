package simplefile

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

func (s *BlobStorage) getBlobsV2(callback func(id string) bool) error {
	err := filepath.Walk(s.filepath, func(path string, info os.FileInfo, err error) error {
		if !strings.Contains(path, RetentionPath) && strings.HasSuffix(path, DescriptionExt) {
			id := info.Name()[:len(info.Name())-5]
			ok := callback(id)
			if !ok {
				return io.EOF
			}
		}
		return nil
	})
	return err
}

func (s *BlobStorage) hasBlobV2(id string) bool {
	binFile := s.getBinV2(id)
	if _, err := os.Stat(binFile); os.IsNotExist(err) {
		return false
	}
	descFile := s.getDescV2(id)
	if _, err := os.Stat(descFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func (s *BlobStorage) getBinV2(id string) string {
	file, _ := s.buildFilenameV2(id, BinaryExt)
	return file
}

func (s *BlobStorage) getDescV2(id string) string {
	file, _ := s.buildFilenameV2(id, DescriptionExt)
	return file
}

func (s *BlobStorage) getBlobDescriptionV2(id string) (*model.BlobDescription, error) {
	var info model.BlobDescription
	s.cm.RLock()
	bdc, ok := s.bdCch[id]
	s.cm.RUnlock()
	if ok {
		return &bdc, nil
	}
	descFile := s.getDescV2(id)
	if _, err := os.Stat(descFile); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}

	dat, err := os.ReadFile(descFile)
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
func (s *BlobStorage) updateBlobDescriptionV2(_ string, b *model.BlobDescription) error {
	err := s.writeJSONFileV2(b)
	if err != nil {
		return err
	}
	s.cm.Lock()
	defer s.cm.Unlock()
	s.bdCch[b.BlobID] = *b
	return nil
}

func (s *BlobStorage) getBlobV2(id string, w io.Writer) error {
	binFile, err := s.buildFilenameV2(id, BinaryExt)
	if err != nil {
		log.Logger.Errorf("error building filename: %v", err)
		return err
	}
	if _, err := os.Stat(binFile); os.IsNotExist(err) {
		log.Logger.Errorf("error not exists: %v", err)
		return os.ErrNotExist
	}
	f, err := os.Open(binFile)
	if err != nil {
		log.Logger.Errorf("error opening file: %v", err)
		return err
	}
	defer f.Close()
	if _, err = io.Copy(w, f); err != nil {
		log.Logger.Errorf("error on copy: %v", err)
		return err
	}
	return nil
}

func (s *BlobStorage) storeBlobV2(b *model.BlobDescription, f io.Reader) (string, error) {
	if b.BlobID == "" {
		uuid := utils.GenerateID()
		b.BlobID = uuid
	}
	size, hash, err := s.writeBinFileV2(b.BlobID, f)
	if err != nil {
		return "", err
	}
	if (b.ContentLength > 0) && b.ContentLength != size {
		_ = s.deleteFilesV2(b.BlobID)
		return "", fmt.Errorf("wrong content length %d=%d", b.ContentLength, size)
	}
	b.Hash = hash
	b.ContentLength = size
	err = s.writeJSONFileV2(b)
	if err != nil {
		_ = s.deleteFilesV2(b.BlobID)
		return "", err
	}
	s.cm.Lock()
	defer s.cm.Unlock()
	s.bdCch[b.BlobID] = *b
	return b.BlobID, nil
}

func (s *BlobStorage) writeBinFileV2(id string, r io.Reader) (int64, string, error) {
	binFile, err := s.buildFilenameV2(id, BinaryExt)
	if err != nil {
		return 0, "", err
	}

	err = s.createFilePathV2(id)
	if err != nil {
		return 0, "", err
	}

	f, err := os.Create(binFile)
	if err != nil {
		return 0, "", err
	}
	h := sha256.New()
	w := io.MultiWriter(f, h)

	size, err := io.Copy(w, r)

	if err != nil {
		_ = f.Close()
		_ = os.Remove(binFile)
		return 0, "", err
	}
	err = f.Close()
	hash := fmt.Sprintf("sha-256:%x", h.Sum(nil))
	return size, hash, err
}

func (s *BlobStorage) deleteFilesV2(id string) error {
	binFile := s.getBinV2(id)
	err := os.Remove(binFile)
	if err != nil {
		return err
	}
	jsonFile := s.getDescV2(id)
	err = os.Remove(jsonFile)
	if err != nil {
		return err
	}
	jsonFile, _ = s.buildRetentionFilename(id)
	err = os.Remove(jsonFile)
	if (err != nil) && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *BlobStorage) writeJSONFileV2(b *model.BlobDescription) error {
	jsonFile, err := s.buildFilenameV2(b.BlobID, DescriptionExt)
	if err != nil {
		return err
	}

	jsn, err := json.Marshal(b)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(jsonFile, jsn, os.ModePerm)
	if err != nil {
		_ = os.Remove(jsonFile)
		return err
	}

	return nil
}

func (s *BlobStorage) writeRetentionFile(b *model.BlobDescription) error {
	jsonFile, err := s.buildRetentionFilename(b.BlobID)
	if err != nil {
		return err
	}

	jsn, err := json.Marshal(b)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(jsonFile, jsn, os.ModePerm)
	if err != nil {
		_ = os.Remove(jsonFile)
		return err
	}

	return nil
}

func (s *BlobStorage) getRetention(id string) (*model.RetentionEntry, error) {
	jsonFile, err := s.buildRetentionFilename(id)
	if err != nil {
		return nil, err
	}
	dat, err := os.ReadFile(jsonFile)
	if err != nil {
		log.Logger.Errorf("GetRetention: error getting file data for: %s\r\n%v", jsonFile, err)
		return nil, err
	}
	r := model.RetentionEntry{}
	err = json.Unmarshal(dat, &r)
	if err != nil {
		log.Logger.Errorf("GetRetention: deserialization error: %s\r\n%v", jsonFile, err)
		return nil, err
	}
	return &r, nil
}

func (s *BlobStorage) deleteRetentionFile(id string) error {
	jsonFile, err := s.buildRetentionFilename(id)
	if err != nil {
		return err
	}
	err = os.Remove(jsonFile)
	return err
}

func (s *BlobStorage) buildFilenameV2(id string, ext string) (string, error) {
	fp := s.filepath
	fp = filepath.Join(fp, id[:2])
	fp = filepath.Join(fp, id[2:4])
	return filepath.Join(fp, fmt.Sprintf("%s%s", id, ext)), nil
}

func (s *BlobStorage) createFilePathV2(id string) error {
	fp := s.filepath
	fp = filepath.Join(fp, id[:2])
	fp = filepath.Join(fp, id[2:4])
	return os.MkdirAll(fp, os.ModePerm)
}
