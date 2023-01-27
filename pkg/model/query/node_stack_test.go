package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeStackNew(t *testing.T) {
	ast := assert.New(t)

	ns := NodeStack{}
	ns.Init()

	ast.NotNil(ns.CurrentNode())

	n := ns.NewNode()

	ast.NotNil(n)
	ast.NotNil(ns.CurrentNode())
	ast.Equal(NOOP, ns.CurrentNode().Operator)

	n.Operator = OROP
	ast.Equal(OROP, ns.CurrentNode().Operator)
}

func TestNodeStackCondition(t *testing.T) {
	ast := assert.New(t)

	ns := NodeStack{}
	ns.Init()

	ast.NotNil(N.CurrentNode())

	n := ns.NewNode()

	ast.NotNil(n)
	ast.NotNil(ns.CurrentNode())
	ast.NotNil(ns.CurrentCondition())

	c := ns.currentCondition
	ast.NotNil(c)

	c = ns.NewCondition()
	ast.NotNil(c)

	c.Operator = LT
	c.Field = "Willie"
	ast.Equal(LT, ns.CurrentCondition().Operator)
	ast.Equal("Willie", c.Field)
}
