package interfaces

import (
	"github.com/willie68/GoBlobStore/internal/config"
)

// Extractor interface for extraction
type Extractor interface {
	Init(cnfg config.Extractor, stg BlobStorage) error
	Metadata(id string) (map[string]any, error)
	Fulltext(id string) (string, error)
	Close() error
}
