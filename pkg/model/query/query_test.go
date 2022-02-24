package query

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQJson(t *testing.T) {
	ast := assert.New(t)

	q := Query{
		Sorting: []string{"field_1"},
		Condition: Node{
			Operator: NOOP,
			Conditions: []interface{}{
				Condition{
					Field:    "field_1",
					Operator: EQ,
					Value:    "Willie",
				},
			},
		},
	}

	jss, _ := json.MarshalIndent(q, "", "  ")
	ast.NotEmpty(jss)
	fmt.Println(string(jss))
}

func TestQSearching(t *testing.T) {
	ast := assert.New(t)
	nodes := []struct {
		n Node
		r string
	}{
		{
			n: Node{
				Operator: NOOP,
				Conditions: []interface{}{
					Condition{
						Field:    "field1",
						Operator: EQ,
						Value:    "Willie",
					},
				},
			},
			r: `field1:="Willie"`,
		},
		{
			n: Node{
				Operator: ANDOP,
				Conditions: []interface{}{
					Condition{
						Field:    "field1",
						Operator: NO,
						Value:    "Willie",
					},
					Condition{
						Field:    "field2",
						Operator: GT,
						Invert:   true,
						Value:    100,
					},
				},
			},
			r: `(field1:"Willie" AND !(field2:>100))`,
		},
		{
			n: Node{
				Operator: ANDOP,
				Conditions: []interface{}{
					Condition{
						Field:    "field1",
						Operator: NO,
						Value:    "Willie",
					},
					Condition{
						Field:    "field2",
						Operator: GT,
						Invert:   true,
						Value:    100,
					},
				},
			},
			r: `(field1:"Willie" AND !(field2:>100))`,
		},
		{
			n: Node{
				Operator: OROP,
				Conditions: []interface{}{
					Node{
						Operator: ANDOP,
						Conditions: []interface{}{
							Condition{
								Field:    "field1",
								Operator: NO,
								Value:    "Willie",
							},
							Condition{
								Field:    "field2",
								Operator: GT,
								Value:    100,
							},
						},
					},
					Node{
						Operator: ANDOP,
						Conditions: []interface{}{
							Condition{
								Field:    "field1",
								Operator: NO,
								Value:    "Max",
							},
							Condition{
								Field:    "field2",
								Operator: LE,
								Invert:   true,
								Value:    100,
							},
						},
					},
				},
			},
			r: `((field1:"Willie" AND field2:>100) OR (field1:"Max" AND !(field2:<=100)))`,
		},
	}

	for _, n := range nodes {
		s := n.n.String()
		ast.NotEmpty(s)
		ast.Equal(n.r, s)
		fmt.Println(s)
	}
}

func TestCCondition(t *testing.T) {
	conditions := []struct {
		c Condition
		r string
	}{
		{
			c: Condition{
				Field:    "field_1",
				Operator: EQ,
				Value:    "Willie",
			},
			r: "field_1:=\"Willie\"",
		},
		{
			c: Condition{
				Field:    "field_1",
				Operator: LT,
				Value:    "Willie",
			},
			r: "field_1:<\"Willie\"",
		},
		{
			c: Condition{
				Field:    "field_1",
				Operator: GT,
				Value:    "Willie",
			},
			r: "field_1:>\"Willie\"",
		},
		{
			c: Condition{
				Field:    "field_1",
				Operator: LE,
				Value:    1234,
			},
			r: "field_1:<=1234",
		},
		{
			c: Condition{
				Field:    "field_1",
				Operator: GE,
				Value:    1234.567,
			},
			r: "field_1:>=1234.56700000",
		},
		{
			c: Condition{
				Field:    "field_1",
				Operator: GE,
				Value:    1234.123456789,
			},
			r: "field_1:>=1234.12345679",
		},
		{
			c: Condition{
				Field:    "field_1",
				Operator: GE,
				Invert:   true,
				Value:    1234.123456789,
			},
			r: "!(field_1:>=1234.12345679)",
		},
	}

	ast := assert.New(t)
	for _, ct := range conditions {
		s := ct.c.String()
		ast.NotEmpty(s)
		ast.Equal(ct.r, s)
		fmt.Println(s)
	}
}

func TestQParsing(t *testing.T) {
	ast := assert.New(t)

	nodes := []struct {
		n Node
		s string
	}{
		{
			n: Node{
				Operator: NOOP,
				Conditions: []interface{}{
					Condition{
						Field:    "field1",
						Operator: EQ,
						Value:    "Willie",
					},
				},
			},
			s: `field1:="Willie"`,
		},
		{
			n: Node{
				Operator: ANDOP,
				Conditions: []interface{}{
					Condition{
						Field:    "field1",
						Operator: NO,
						Value:    "Willie",
					},
					Condition{
						Field:    "field2",
						Operator: GT,
						Invert:   true,
						Value:    100,
					},
				},
			},
			s: `(field1:"Willie" AND !(field2:>100))`,
		},
		{
			n: Node{
				Operator: ANDOP,
				Conditions: []interface{}{
					Condition{
						Field:    "field1",
						Operator: NO,
						Value:    "Willie",
					},
					Condition{
						Field:    "field2",
						Operator: GT,
						Invert:   true,
						Value:    100,
					},
				},
			},
			s: `(field1:"Willie" AND !(field2:>100))`,
		},
		{
			n: Node{
				Operator: OROP,
				Conditions: []interface{}{
					Node{
						Operator: ANDOP,
						Conditions: []interface{}{
							Condition{
								Field:    "field1",
								Operator: NO,
								Value:    "Willie",
							},
							Condition{
								Field:    "field2",
								Operator: GT,
								Value:    100,
							},
						},
					},
					Node{
						Operator: ANDOP,
						Conditions: []interface{}{
							Condition{
								Field:    "field1",
								Operator: NO,
								Value:    "Max",
							},
							Condition{
								Field:    "field2",
								Operator: LE,
								Invert:   true,
								Value:    100,
							},
						},
					},
				},
			},
			s: `((field1:"Willie" AND field2:>100) OR (field1:"Max" AND !(field2:<=100)))`,
		},
	}

	for _, n := range nodes {
		fmt.Println(n.s)
		st, err := ParseMe(n.s)
		ast.Nil(err)
		ast.NotNil(st)
		ast.NotNil(st.Condition)
		sn, ok := st.Condition.(Node)
		ast.True(ok)

		sns := sn.String()
		fmt.Println(sns)
		ast.Equal(n.s, sns)
	}

}

func TestQParse(t *testing.T) {
	ast := assert.New(t)
	s := `field:"Willie"`
	//s := `event:"sent" AND subject:"A special offer just for you!"`
	res, err := Parse("query", []byte(s))
	t.Log(res)
	if err != nil {
		t.Logf("Error: %v", err)
	}
	ast.Nil(err)
}
