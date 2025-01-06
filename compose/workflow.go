/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package compose

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

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
	fromNodeKey string

	fromField  string
	fromMapKey string

	toField  string
	toMapKey string
}

func (m *Mapping) empty() bool {
	return len(m.fromField) == 0 && len(m.fromMapKey) == 0 && len(m.toField) == 0 && len(m.toMapKey) == 0
}

func (m *Mapping) FromField(fieldName string) *Mapping {
	m.fromField = fieldName
	return m
}

func (m *Mapping) ToField(fieldName string) *Mapping {
	m.toField = fieldName
	return m
}

func (m *Mapping) FromMapKey(mapKey string) *Mapping {
	m.fromMapKey = mapKey
	return m
}

func (m *Mapping) ToMapKey(mapKey string) *Mapping {
	m.toMapKey = mapKey
	return m
}

func (m *Mapping) String() string {
	var sb strings.Builder
	sb.WriteString("from ")

	if m.fromMapKey != "" {
		sb.WriteString(m.fromMapKey)
		sb.WriteString("(map key) of ")
	}

	if m.fromField != "" {
		sb.WriteString(m.fromField)
		sb.WriteString("(field) of ")
	}

	sb.WriteString("node '")
	sb.WriteString(m.fromNodeKey)
	sb.WriteString("'")

	if m.toField != "" {
		sb.WriteString(" to ")
		sb.WriteString(m.toField)
		sb.WriteString("(field)")
	}

	if m.toMapKey != "" {
		sb.WriteString(" to ")
		sb.WriteString(m.toMapKey)
		sb.WriteString("(map key)")
	}

	sb.WriteString("; ")
	return sb.String()
}

func NewMapping(fromNodeKey string) *Mapping {
	return &Mapping{fromNodeKey: fromNodeKey}
}

type WorkflowNode struct {
	key         string
	inputs      []*Mapping
	fieldMapper fieldMapper
}

type Workflow[I, O any] struct {
	gg *Graph[I, O]

	nodes          map[string]*WorkflowNode
	end            []*Mapping
	endFieldMapper fieldMapper
	err            error
}

func NewWorkflow[I, O any](opts ...NewGraphOption) *Workflow[I, O] {
	wf := &Workflow[I, O]{
		gg:    NewGraph[I, O](opts...),
		nodes: make(map[string]*WorkflowNode),
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
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]*schema.Message]{}}
	wf.nodes[key] = node

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
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[map[string]any]{}}
	wf.nodes[key] = node

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
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[*schema.Message]{}}
	wf.nodes[key] = node

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
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[string]{}}
	wf.nodes[key] = node

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
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]string]{}}
	wf.nodes[key] = node

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
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[[]*schema.Document]{}}
	wf.nodes[key] = node

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
	node := &WorkflowNode{key: key, fieldMapper: defaultFieldMapper[document.Source]{}}
	wf.nodes[key] = node

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
	wf.nodes[key] = node

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
	wf.nodes[key] = node

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
	wf.nodes[key] = node

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

func (wf *Workflow[I, O]) AddEnd(inputs ...*Mapping) {
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
	var toNode string
	for _, node := range wf.nodes {
		toNode = node.key
		fm := node.fieldMapper

		if len(node.inputs) == 0 {
			return fmt.Errorf("workflow node = %s has no input", toNode)
		}

		fromNode2Mappings := make(map[string][]*Mapping, len(node.inputs))
		for i := range node.inputs {
			input := node.inputs[i]
			fromNodeKey := input.fromNodeKey
			fromNode2Mappings[fromNodeKey] = append(fromNode2Mappings[fromNodeKey], input)
		}

		for fromNode, mappings := range fromNode2Mappings {
			if err = checkMappingGroup(mappings); err != nil {
				return err
			}

			if mappings[0].empty() {
				if err = wf.gg.AddEdge(fromNode, toNode); err != nil {
					return err
				}
			} else if err = wf.gg.addEdgeWithMappings(fromNode, toNode, fm.fieldMap(mappings), fm.streamFieldMap(mappings), mappings...); err != nil {
				return err
			}
		}
	}

	if len(wf.end) == 0 {
		return errors.New("workflow END has no input mapping")
	}

	fm := wf.endFieldMapper

	fromNode2EndMappings := make(map[string][]*Mapping, len(wf.end))
	for i := range wf.end {
		input := wf.end[i]
		fromNodeKey := input.fromNodeKey
		fromNode2EndMappings[fromNodeKey] = append(fromNode2EndMappings[fromNodeKey], input)
	}

	for fromNode, mappings := range fromNode2EndMappings {
		if err = checkMappingGroup(mappings); err != nil {
			return err
		}

		if mappings[0].empty() {
			if err = wf.gg.AddEdge(fromNode, END); err != nil {
				return err
			}
		} else if err = wf.gg.addEdgeWithMappings(fromNode, END, fm.fieldMap(mappings), fm.streamFieldMap(mappings), mappings...); err != nil {
			return err
		}
	}

	return nil
}
