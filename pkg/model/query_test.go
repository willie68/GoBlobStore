package model

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJson(t *testing.T) {
	ast := assert.New(t)

	q := Query{
		Sorting: []string{"field_1"},
		Condition: Node{
			Operator: NOOP,
			Conditions: []Condition{
				{
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

type TCondition struct {
	C   Condition
	Res string
}

func TestCondition(t *testing.T) {
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
	}

	ast := assert.New(t)
	for _, ct := range conditions {
		s := ct.c.String()
		ast.NotEmpty(s)
		ast.Equal(ct.r, s)
		fmt.Println(s)
	}
}
