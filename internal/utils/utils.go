package utils

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

func GenerateID() string {
	uuidStr := uuid.NewString()
	uuidStr = strings.ReplaceAll(uuidStr, "-", "")
	return uuidStr
}

func BuildHash(id string, stg interfaces.BlobStorageDao) (string, error) {
	h := sha256.New()
	err := stg.RetrieveBlob(id, h)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha-256:%x", h.Sum(nil)), nil
}

func CheckBlob(id string, s interfaces.BlobStorageDao) (*model.CheckInfo, error) {
	ok, err := s.HasBlob(id)
	if !ok {
		err = os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	bd, err := s.GetBlobDescription(id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	res := model.CheckInfo{
		LastCheck: &now,
		Healthy:   true,
	}
	// checking the hash of the primary blob
	hash, err := BuildHash(id, s)
	if err != nil {
		return nil, err
	}
	if hash != bd.Hash {
		res.Healthy = false
		res.Message = "hash not correct"
	}
	return &res, nil
}
