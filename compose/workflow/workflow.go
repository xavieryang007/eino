package workflow

import "github.com/cloudwego/eino/compose"

type Mapping struct {
	From string

	FromField  string
	FromMapKey string

	ToField  string
	ToMapKey string
}

type Node struct {
	Key    string
	Inputs []*Mapping
}

type Workflow[I, O any] struct {
	Nodes []*Node
	End   []*Mapping
}

func NewWorkflow[I, O any]() *Workflow[I, O] {
	return &Workflow[I, O]{}
}

func (wf *Workflow[I, O]) AddLambdaNode(key string, lambda *compose.Lambda, opts ...compose.GraphAddNodeOpt) *Node {
	node := &Node{Key: key}
	wf.Nodes = append(wf.Nodes, node)
	return node
}

func (n *Node) AddInput(inputs ...*Mapping) *Node {
	n.Inputs = append(n.Inputs, inputs...)
	return n
}

func (wf *Workflow[I, O]) AddEnd(inputs []*Mapping) {
	wf.End = inputs
}
