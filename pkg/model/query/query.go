package query

import (
	"fmt"
	"strconv"
	"strings"
)

// Query holding the parsed query
type Query struct {
	Sorting   []string    // sorting the result of the query
	Condition interface{} // the condition or a Node
}

// NodeOperator defining as type
type NodeOperator string

// some node operators
const (
	NOOP  NodeOperator = "NOP"
	ANDOP NodeOperator = "AND"
	OROP  NodeOperator = "OR"
)

// Node a implementation of a single node
type Node struct {
	Operator   NodeOperator
	Conditions []interface{} // could be a node or a condition
}

// FieldOperator as type
type FieldOperator string

// defining field operators
const (
	NO FieldOperator = ""   // equal
	EQ FieldOperator = "="  // equals
	LT FieldOperator = "<"  // lesser than
	GT FieldOperator = ">"  // greater than
	LE FieldOperator = "<=" // less or equal
	GE FieldOperator = ">=" // greater or equal
	NE FieldOperator = "!=" // not equal
)

// Condition as struct
type Condition struct {
	Field    string
	Operator FieldOperator
	Value    interface{}
	Invert   bool
}

func (q *Query) String() string {
	var c string
	switch v := q.Condition.(type) {
	case *Condition:
		c = v.String()
	case *Node:
		c = v.String()
	}
	return fmt.Sprintf("query: %s, sort %s", c, strings.Join(q.Sorting, ","))
}

func (n *Node) String() string {
	var b strings.Builder
	cl := len(n.Conditions)
	if cl > 1 {
		b.WriteString("(")
	}
	f := true
	for _, c := range n.Conditions {
		if !f {
			b.WriteString(" ")
			b.WriteString(string(n.Operator))
			b.WriteString(" ")
		}
		switch v := c.(type) {
		case Condition:
			b.WriteString(v.String())
		case *Condition:
			b.WriteString(v.String())
		case Node:
			b.WriteString(v.String())
		case *Node:
			b.WriteString(v.String())
		default:

		}
		f = false
	}
	if cl > 1 {
		b.WriteString(")")
	}
	return b.String()
}

func (c *Condition) String() string {
	op := string(c.Operator)
	v := c.VtoS()
	lc := fmt.Sprintf("%s:%s%s", c.Field, op, v)
	if c.Invert {
		lc = fmt.Sprintf(`!(%s)`, lc)
	}
	return lc
}

// VtoS return a formatted string from the different values
func (c *Condition) VtoS() string {
	switch v := c.Value.(type) {
	case string:
		if !strings.HasPrefix(v, `"`) {
			v = `"` + v
		}
		if !strings.HasSuffix(v, `"`) {
			v = v + `"`
		}
		return v
	case int, int64:
		return fmt.Sprintf("%d", v)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', 8, 32)
	case float64:
		return strconv.FormatFloat(float64(v), 'f', 8, 64)
	}
	return ""
}

// HasWildcard checking if a condition contains a wildcard
func (c *Condition) HasWildcard() bool {
	switch v := c.Value.(type) {
	case string:
		return strings.ContainsAny(v, "*?")
	default:
		return false
	}
}

/*
func ParseMe(s string) (*Query, error) {
	res, err := Parse("query", []byte(s))
	fmt.Printf("%v", res)
	if err != nil {
		return nil, err
	}
	q, ok := res.(Query)
	if ok {
		return &q, nil
	}
	return nil, errors.New("error on parsing")
}
*/
