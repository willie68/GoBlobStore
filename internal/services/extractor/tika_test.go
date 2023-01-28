package extractor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTikaExtractorTxt(t *testing.T) {
	ast := assert.New(t)

	ext := NewExtractor("tika")
	_, ok := ext.(*Tika)
	ast.True(ok)
}
