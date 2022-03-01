package query

import log "github.com/willie68/GoBlobStore/internal/logging"

type NodeStack struct {
	InvertGroup      bool
	currentNode      *Node
	currentCondition *Condition
}

var N NodeStack

func init() {
	N.Init()
}

func (ns *NodeStack) Init() {
}

func (ns *NodeStack) Reset() {
	ns.currentNode = nil
	ns.currentCondition = nil
}

func (ns *NodeStack) Query() Query {
	log.Logger.Info("get query")
	var c interface{}
	c = ns.currentNode
	if ns.currentNode == nil {
		c = ns.currentCondition
	} else {
		ns.currentNode.Conditions = append(ns.currentNode.Conditions, ns.currentCondition)
	}
	q := Query{
		Sorting:   []string{""},
		Condition: c,
	}
	return q
}

func (ns *NodeStack) NewNode() *Node {
	log.Logger.Info("new node")
	n := Node{
		Operator:   NOOP,
		Conditions: make([]interface{}, 0),
	}
	if ns.currentNode != nil {
		ns.currentNode.Conditions = append(ns.currentNode.Conditions, ns.currentCondition)
		n.Conditions = append(n.Conditions, ns.currentNode)
		ns.currentCondition = nil
	} else {
		if ns.currentCondition != nil {
			n.Conditions = append(n.Conditions, ns.currentCondition)
			ns.currentCondition = nil
		}
	}
	ns.currentNode = &n
	return &n
}

func (ns *NodeStack) CurrentNode() *Node {
	if ns.currentNode == nil {
		ns.NewNode()
	}
	return ns.currentNode
}

func (ns *NodeStack) NewCondition() *Condition {
	log.Logger.Info("new condition")
	if ns.currentCondition != nil {
		if ns.currentNode != nil {
			ns.currentNode.Conditions = append(ns.currentNode.Conditions, ns.currentCondition)
		} else {
			ns.NewNode()
		}
	}
	ns.currentCondition = nil
	c := Condition{
		Operator: NO,
		Field:    "",
		Invert:   false,
	}
	ns.currentCondition = &c
	return &c
}

func (ns *NodeStack) CurrentCondition() *Condition {
	if ns.currentCondition == nil {
		return ns.NewCondition()
	}
	return ns.currentCondition
}
