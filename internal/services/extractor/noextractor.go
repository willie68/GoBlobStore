package extractor

import (
	"fmt"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

var _ interfaces.Extractor = &NoExtractor{}

// NoExtractor simple do nothing service for text extraction
type NoExtractor struct {
	stg interfaces.BlobStorage
}

// Init do nothing on init
func (n *NoExtractor) Init(_ config.Extractor, stg interfaces.BlobStorage) error {
	n.stg = stg
	return nil
}

// Metadata returning some metadata of the blob, here nothing
func (n *NoExtractor) Metadata(id string) (map[string]any, error) {
	bd, err := n.stg.GetBlobDescription(id)
	if err != nil {
		return nil, err
	}
	meta := bd.Map()
	return meta, nil
}

// Fulltext returning the full text of the blob, here nothing
func (n *NoExtractor) Fulltext(id string) (string, error) {
	ok, err := n.stg.HasBlob(id)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("blob not found: %s", id)
	}
	return "", nil
}

// Close closing this service
func (n *NoExtractor) Close() error {
	return nil
}
