package model

import (
	"fmt"
	"strconv"
)

// examples

type Query struct {
	Sorting   []string // sorting the result of the query
	Condition Node     // the condition as a Node
}

type NodeOperator string

const (
	NOOP  NodeOperator = "NOP"
	ANDOP NodeOperator = "AND"
	OROP  NodeOperator = "OR"
)

type Node struct {
	Operator   NodeOperator
	Conditions []interface{} // could be a node or a condition
}

type FieldOperator string

const (
	EQ FieldOperator = "="  // equals
	LT FieldOperator = "<"  // lesser than
	GT FieldOperator = ">"  // greater than
	LE FieldOperator = "<=" // less or equal
	GE FieldOperator = ">=" // greater or equal
	NE FieldOperator = "!=" // not equal
)

type Condition struct {
	Field    string
	Operator FieldOperator
	Value    interface{}
}

func (c *Condition) String() string {
	switch v := c.Value.(type) {
	case string:
		return fmt.Sprintf("%s:%s\"%s\"", c.Field, c.Operator, v)
	case int, int64:
		return fmt.Sprintf("%s:%s%d", c.Field, c.Operator, v)
	case float32:
		s := strconv.FormatFloat(float64(v), 'f', 8, 32)
		return fmt.Sprintf("%s:%s%s", c.Field, c.Operator, s)
	case float64:
		s := strconv.FormatFloat(float64(v), 'f', 8, 64)
		return fmt.Sprintf("%s:%s%s", c.Field, c.Operator, s)
	}
	return ""
}
