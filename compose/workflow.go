package compose

import (
	"context"
	"fmt"
	"reflect"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/generic"
)

type Mapping struct {
	From string

	FromField  string
	FromMapKey string

	ToField  string
	ToMapKey string
}

func (m *Mapping) empty() bool {
	return len(m.FromField) == 0 && len(m.FromMapKey) == 0 && len(m.ToField) == 0 && len(m.ToMapKey) == 0
}

type WorkflowNode struct {
	key         string
	inputs      []*Mapping
	fieldMapper fieldMapper
}

type Workflow[I, O any] struct {
	gg *Graph[I, O]

	nodes          []*WorkflowNode
	end            []*Mapping
	endFieldMapper fieldMapper
	err            error
}

func NewWorkflow[I, O any](opts ...NewGraphOption) *Workflow[I, O] {
	wf := &Workflow[I, O]{
		gg: NewGraph[I, O](opts...),
	}

	wf.gg.cmp = ComponentOfWorkflow
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

	if err := wf.addEdgesWithMapping(); err != nil {
		return nil, err
	}

	return wf.gg.Compile(ctx, gCompileOpts...)
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
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[*schema.Message]{}}
	wf.nodes = append(wf.nodes, node)

	if wf.err != nil {
		return node
	}

	err := wf.gg.AddChatModelNode(key, chatModel, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddChatTemplateNode(key string, chatTemplate prompt.ChatTemplate, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]*schema.Message]{}}
	wf.nodes = append(wf.nodes, node)

	if wf.err != nil {
		return node
	}

	err := wf.gg.AddChatTemplateNode(key, chatTemplate, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}

	return node
}

func (wf *Workflow[I, O]) AddToolsNode(key string, tools *ToolsNode, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]*schema.Message]{}}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.gg.AddToolsNode(key, tools, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddRetrieverNode(key string, retriever retriever.Retriever, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]*schema.Document]{}}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.gg.AddRetrieverNode(key, retriever, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddEmbeddingNode(key string, embedding embedding.Embedder, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[][]float64]{}}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.gg.AddEmbeddingNode(key, embedding, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddIndexerNode(key string, indexer indexer.Indexer, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]string]{}}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.gg.AddIndexerNode(key, indexer, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddLoaderNode(key string, loader document.Loader, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]*schema.Document]{}}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.gg.AddLoaderNode(key, loader, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddDocumentTransformerNode(key string, transformer document.Transformer, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]*schema.Document]{}}
	wf.nodes = append(wf.nodes, node)
	if wf.err != nil {
		return node
	}

	err := wf.gg.AddDocumentTransformerNode(key, transformer, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddGraphNode(key string, graph AnyGraph, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: graph.fieldMapper()}
	wf.nodes = append(wf.nodes, node)

	if wf.err != nil {
		return node
	}

	err := wf.gg.AddGraphNode(key, graph, convertAddNodeOpts(opts)...)
	if err != nil {
		wf.err = err
		return node
	}
	return node
}

func (wf *Workflow[I, O]) AddLambdaNode(key string, lambda *Lambda, opts ...WorkflowAddNodeOpt) *WorkflowNode {
	node := &WorkflowNode{key: key, fieldMapper: lambda.fieldMapper}
	wf.nodes = append(wf.nodes, node)

	if wf.err != nil {
		return node
	}

	err := wf.gg.AddLambdaNode(key, lambda, convertAddNodeOpts(opts)...)
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
	wf.endFieldMapper = defaultFieldMapper[O]{}
}

func (wf *Workflow[I, O]) compile(ctx context.Context, options *graphCompileOptions) (*composableRunnable, error) {
	options.nodeTriggerMode = AllPredecessor
	if err := wf.addEdgesWithMapping(); err != nil {
		return nil, err
	}
	return wf.gg.compile(ctx, options)
}

func (wf *Workflow[I, O]) inputType() reflect.Type {
	return generic.TypeOf[I]()
}

func (wf *Workflow[I, O]) outputType() reflect.Type {
	return generic.TypeOf[O]()
}

func (wf *Workflow[I, O]) component() component {
	return wf.gg.component()
}

func (wf *Workflow[I, O]) fieldMapper() fieldMapper {
	return wf.gg.fieldMapper()
}

func (wf *Workflow[I, O]) addEdgesWithMapping() (err error) {
	for _, node := range wf.nodes {
		nodeKey := node.key

		if len(node.inputs) == 0 {
			return fmt.Errorf("workflow node = %s has no input", nodeKey)
		}

		fromNode2Mappings := make(map[string][]*Mapping, len(node.inputs))
		for i := range node.inputs {
			input := node.inputs[i]
			fromNodeKey := input.From
			fromNode2Mappings[fromNodeKey] = append(fromNode2Mappings[fromNodeKey], input)
		}

		for fromNode, mappings := range fromNode2Mappings {
			if err = checkMappingGroup(mappings); err != nil {
				return err
			}

			if err = wf.gg.addEdgeWithMappings(fromNode, nodeKey, mappings...); err != nil {
				return err
			}
		}
	}

	fromNode2EndMappings := make(map[string][]*Mapping, len(wf.end))
	for i := range wf.end {
		input := wf.end[i]
		fromNodeKey := input.From
		fromNode2EndMappings[fromNodeKey] = append(fromNode2EndMappings[fromNodeKey], input)
	}

	for fromNode, mappings := range fromNode2EndMappings {
		if err = checkMappingGroup(mappings); err != nil {
			return err
		}

		if err = wf.gg.addEdgeWithMappings(fromNode, END, mappings...); err != nil {
			return err
		}
	}

	return nil
}
