package opscoor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstancingSN(t *testing.T) {
	ast := assert.New(t)

	snoc := &SingleNodeOpsCoor{}

	ast.NotNil(snoc)
}
