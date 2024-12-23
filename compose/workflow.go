package compose

type Mapping struct {
	From string

	FromField  string
	FromMapKey string

	ToField  string
	ToMapKey string
}

type WorkflowNode struct {
	Key    string
	Inputs []*Mapping
}

type Workflow[I, O any] struct {
	Nodes []*WorkflowNode
	End   []*Mapping
}

func NewWorkflow[I, O any]() *Workflow[I, O] {
	return &Workflow[I, O]{}
}

func (wf *Workflow[I, O]) AddLambdaNode(key string, lambda *Lambda, opts ...GraphAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{Key: key}
	wf.Nodes = append(wf.Nodes, node)
	return node
}

func (n *WorkflowNode) AddInput(inputs ...*Mapping) *WorkflowNode {
	n.Inputs = append(n.Inputs, inputs...)
	return n
}

func (wf *Workflow[I, O]) AddEnd(inputs []*Mapping) {
	wf.End = inputs
}
