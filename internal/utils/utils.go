package utils

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
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
