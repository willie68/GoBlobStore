package extractor

import (
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

// NewExtractor creates a new extractor service based on the type string
func NewExtractor(etype string) interfaces.Extractor {
	var ext interfaces.Extractor
	switch etype {
	case "tika":
		ext = &Tika{}
	default:
		ext = &NoExtractor{}
	}
	return ext
}
