package compose

import (
	"context"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
)

type WorkflowNode struct {
	key    string
	inputs []*Mapping
}

type Workflow[I, O any] struct {
	g *Graph[I, O]

	nodes []*WorkflowNode
	end   []*Mapping
	err   error
}

func NewWorkflow[I, O any](opts ...NewGraphOption) *Workflow[I, O] {
	wf := &Workflow[I, O]{
		g: NewGraph[I, O](opts...),
	}

	wf.g.cmp = ComponentOfWorkflow
	return wf
}

type WorkflowCompileOption GraphCompileOption

func WithWorkflowMaxRunStep(maxSteps int) WorkflowCompileOption {
	return WorkflowCompileOption(WithMaxRunSteps(maxSteps))
}

func WithWorkflowName(name string) WorkflowCompileOption {
	return WorkflowCompileOption(WithGraphName(name))
}

func (wf *Workflow[I, O]) Compile(ctx context.Context, opts ...WorkflowCompileOption) (Runnable[I, O], error) {
	if wf.err != nil {
		return nil, wf.err
	}

	gCompileOpts := make([]GraphCompileOption, 0, len(opts)+1)
	for _, opt := range opts {
		gCompileOpts = append(gCompileOpts, GraphCompileOption(opt))
	}
	gCompileOpts = append(gCompileOpts, WithNodeTriggerMode(AllPredecessor))

	return wf.g.Compile(ctx, gCompileOpts...)
}

type WorkflowAddNodeOpt GraphAddNodeOpt

func WithWorkflowNodeName(name string) WorkflowAddNodeOpt {
	return WorkflowAddNodeOpt(WithNodeName(name))
}

func WithWorkflowStatePreHandler[I, S any](pre StatePreHandler[I, S]) WorkflowAddNodeOpt {
	return WorkflowAddNodeOpt(WithStatePreHandler(pre))
}

func WithWorkflowStatePostHandler[O, S any](post StatePostHandler[O, S]) WorkflowAddNodeOpt {
	return WorkflowAddNodeOpt(WithStatePostHandler(post))
}

func WithWorkflowStreamStatePreHandler[I, S any](pre StreamStatePreHandler[I, S]) WorkflowAddNodeOpt {
	return WorkflowAddNodeOpt(WithStreamStatePreHandler(pre))
}

func WithWorkflowStreamStatePostHandler[O, S any](post StreamStatePostHandler[O, S]) WorkflowAddNodeOpt {
	return WorkflowAddNodeOpt(WithStreamStatePostHandler(post))
}

func convertAddNodeOpts(opts []WorkflowAddNodeOpt) []GraphAddNodeOpt {
	graphOpts := make([]GraphAddNodeOpt, 0, len(opts))
	for _, opt := range opts {
		graphOpts = append(graphOpts, GraphAddNodeOpt(opt))
	}
	return graphOpts
}

func (wf *Workflow[I, O]) AddChatModelNode(key string, chatModel model.ChatModel, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)

	if wf.err != nil {
		return node
	}

	err := wf.g.AddChatModelNode(key, chatModel, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddChatTemplateNode(key string, chatTemplate prompt.ChatTemplate, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)

	if wf.err != nil {
		return node
	}

	err := wf.g.AddChatTemplateNode(key, chatTemplate, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}

	return node
}

func (wf *Workflow[I, O]) AddToolsNode(key string, tools *ToolsNode, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.g.AddToolsNode(key, tools, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddRetrieverNode(key string, retriever retriever.Retriever, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.g.AddRetrieverNode(key, retriever, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddEmbeddingNode(key string, embedding embedding.Embedder, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.g.AddEmbeddingNode(key, embedding, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddIndexerNode(key string, indexer indexer.Indexer, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.g.AddIndexerNode(key, indexer, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddLoaderNode(key string, loader document.Loader, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.g.AddLoaderNode(key, loader, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddDocumentTransformerNode(key string, transformer document.Transformer, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.g.AddDocumentTransformerNode(key, transformer, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddGraphNode(key string, graph AnyGraph, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)

	if wf.err != nil {
		return node
	}

	err := wf.g.AddGraphNode(key, graph, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddLambdaNode(key string, lambda *Lambda, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key}
	wf.nodes = append(wf.nodes, node)

	if wf.err != nil {
		return node
	}

	err := wf.g.AddLambdaNode(key, lambda, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}

	return node
}

func (n *WorkflowNode) AddInput(inputs ...*Mapping) *WorkflowNode {
	n.inputs = append(n.inputs, inputs...)
	return n
}

func (wf *Workflow[I, O]) AddEnd(inputs []*Mapping) {
	wf.end = inputs
}
