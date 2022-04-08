package bluge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blugelabs/bluge"
	"github.com/willie68/GoBlobStore/pkg/model/query"
)

func toBlugeQuery(q query.Query) (bluge.Query, error) {
	c := q.Condition
	bq, err := xToBq(c)
	return bq, err
}

//xToBdb converting a node/condition to a bluge string
func xToBq(x interface{}) (bluge.Query, error) {
	switch v := x.(type) {
	case query.Condition:
		return cToBq(v)
	case *query.Condition:
		return cToBq(*v)
	case query.Node:
		return nToBq(v)
	case *query.Node:
		return nToBq(*v)
	}
	return nil, fmt.Errorf("can't convert %V to bluge query", x)
}

//nToMdb converting a node into a mongo query string
func nToBq(n query.Node) (bluge.Query, error) {
	bq := bluge.NewBooleanQuery()
	for _, c := range n.Conditions {
		q, err := xToBq(c)
		if err != nil {
			return nil, err
		}
		if n.Operator == query.ANDOP {
			bq.AddMust(q)
		} else {
			bq.AddShould(q)
		}
	}
	return bq, nil
}

//cToMdb converting a condition into a mongo query string
func cToBq(c query.Condition) (bluge.Query, error) {
	bq := bluge.NewBooleanQuery()
	var q bluge.Query
	switch c.Operator {
	case query.EQ:
		v, err := cToFloat(c)
		if err != nil {
			q = bluge.NewMatchQuery(cToStr(c)).SetField(c.Field)
		} else {
			q = bluge.NewNumericRangeInclusiveQuery(v, v, true, true).SetField(c.Field)
		}
	case query.NE:
		v, err := cToFloat(c)
		if err != nil {
			return nil, err
		} else {
			q = bluge.NewBooleanQuery().AddMustNot(bluge.NewNumericRangeInclusiveQuery(v, v, true, true).SetField(c.Field))
		}
	case query.GT:
		v, err := cToFloat(c)
		if err != nil {
			return nil, err
		}
		q = bluge.NewNumericRangeInclusiveQuery(v, bluge.MaxNumeric, false, false).SetField(c.Field)
	case query.GE:
		v, err := cToFloat(c)
		if err != nil {
			return nil, err
		}
		q = bluge.NewNumericRangeInclusiveQuery(v, bluge.MaxNumeric, true, false).SetField(c.Field)
	case query.LT:
		v, err := cToFloat(c)
		if err != nil {
			return nil, err
		}
		q = bluge.NewNumericRangeInclusiveQuery(bluge.MinNumeric, v, false, false).SetField(c.Field)
	case query.LE:
		v, err := cToFloat(c)
		if err != nil {
			return nil, err
		}
		q = bluge.NewNumericRangeInclusiveQuery(bluge.MinNumeric, v, false, true).SetField(c.Field)
	default:
		if c.HasWildcard() {
			cq := cToStr(c)
			cq = strings.ToLower(cq)
			q = bluge.NewWildcardQuery(cq).SetField(c.Field)
		} else {
			q = bluge.NewMatchQuery(cToStr(c)).SetField(c.Field)
		}
	}
	if c.Invert {
		bq.AddMustNot(q)
	} else {
		bq.AddMust(q)
	}
	return bq, nil
}

func cToFloat(c query.Condition) (float64, error) {
	switch v := c.Value.(type) {
	case float64:
		return v, nil
	default:
		vs := c.VtoS()
		vs = strings.TrimPrefix(vs, `"`)
		vs = strings.TrimSuffix(vs, `"`)
		nv, err := strconv.ParseFloat(vs, 64)
		return nv, err
	}
}

func cToStr(c query.Condition) string {
	vs := c.VtoS()
	vs = strings.TrimPrefix(vs, `"`)
	vs = strings.TrimSuffix(vs, `"`)
	return vs
}
