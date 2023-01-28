package extractor

import (
	"fmt"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

var _ interfaces.Extractor = &Tika{}

// Tika service calling the tika service to extract metadata and fulltext
type Tika struct {
	stg interfaces.BlobStorage
}

// Init do nothing on init
func (n *Tika) Init(_ config.Extractor, stg interfaces.BlobStorage) error {
	n.stg = stg
	return nil
}

// Metadata returning some metadata of the blob, normally the blob description and some other metadata tika can extract
func (n *Tika) Metadata(id string) (map[string]any, error) {
	bd, err := n.stg.GetBlobDescription(id)
	if err != nil {
		return nil, err
	}
	meta := bd.Map()
	return meta, nil
}

// Fulltext returning the full text of the blob
func (n *Tika) Fulltext(id string) (string, error) {
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
func (n *Tika) Close() error {
	return nil
}
