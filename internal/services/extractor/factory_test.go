package extractor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoExtractorByType(t *testing.T) {
	ast := assert.New(t)

	ext := NewExtractor("nn")
	_, ok := ext.(*NoExtractor)
	ast.True(ok)
}

func TestNoExtractorByDefault(t *testing.T) {
	ast := assert.New(t)

	ext := NewExtractor("")
	_, ok := ext.(*NoExtractor)
	ast.True(ok)

	ext = NewExtractor("muckefuck")
	_, ok = ext.(*NoExtractor)
	ast.True(ok)
}

func TestTikaExtractor(t *testing.T) {
	ast := assert.New(t)

	ext := NewExtractor("tika")
	_, ok := ext.(*Tika)
	ast.True(ok)
}
