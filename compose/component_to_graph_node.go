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
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
)

func toEmbeddingNode(node embedding.Embedder, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(components.ComponentOfEmbedding, node)
	info := getNodeInfo(opts...)
	run := runnableLambda(node.EmbedStrings, nil, nil, nil,
		!meta.isComponentCallbackEnabled,
	)
	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toRetrieverNode(node retriever.Retriever, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(components.ComponentOfRetriever, node)
	info := getNodeInfo(opts...)
	run := runnableLambda(node.Retrieve, nil, nil, nil,
		!meta.isComponentCallbackEnabled,
	)

	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toLoaderSplitterNode(node document.LoaderSplitter, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(components.ComponentOfLoaderSplitter, node)
	info := getNodeInfo(opts...)
	run := runnableLambda(node.LoadAndSplit, nil, nil, nil,
		!meta.isComponentCallbackEnabled,
	)

	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toLoaderNode(node document.Loader, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(components.ComponentOfLoader, node)
	info := getNodeInfo(opts...)
	run := runnableLambda(node.Load, nil, nil, nil,
		!meta.isComponentCallbackEnabled,
	)

	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toIndexerNode(node indexer.Indexer, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(components.ComponentOfIndexer, node)
	info := getNodeInfo(opts...)
	run := runnableLambda(node.Store, nil, nil, nil,
		!meta.isComponentCallbackEnabled,
	)

	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toChatModelNode(node model.ChatModel, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(components.ComponentOfChatModel, node)
	info := getNodeInfo(opts...)

	run := runnableLambda(node.Generate, node.Stream, nil, nil,
		!meta.isComponentCallbackEnabled,
	)

	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toChatTemplateNode(node prompt.ChatTemplate, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(components.ComponentOfPrompt, node)
	info := getNodeInfo(opts...)
	run := runnableLambda(node.Format, nil, nil, nil,
		!meta.isComponentCallbackEnabled,
	)

	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toDocumentTransformerNode(node document.Transformer, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(components.ComponentOfTransformer, node)
	info := getNodeInfo(opts...)
	run := runnableLambda(node.Transform, nil, nil, nil,
		!meta.isComponentCallbackEnabled,
	)

	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toToolsNode(node *ToolsNode, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(ComponentOfToolsNode, node)
	info := getNodeInfo(opts...)
	run := runnableLambda(node.Invoke, node.Stream, nil, nil,
		true,
	)

	gn := toNode(info, run, nil, meta, node, opts...)

	return gn
}

func toLambdaNode(node *Lambda, opts ...GraphAddNodeOpt) *graphNode {
	info := getNodeInfo(opts...)

	gn := toNode(info, node.executor, nil, node.executor.meta, node, opts...)

	return gn
}

func toAnyGraphNode(node AnyGraph, opts ...GraphAddNodeOpt) *graphNode {
	meta := parseExecutorInfoFromComponent(node.component(), node)
	info := getNodeInfo(opts...)

	gn := toNode(info, nil, node, meta, node, opts...)

	return gn
}

func toPassthroughNode(opts ...GraphAddNodeOpt) *graphNode {
	node := composablePassthrough()
	info := getNodeInfo(opts...)
	gn := toNode(info, node, nil, node.meta, node, opts...)
	return gn
}

func toNode(nodeInfo *nodeInfo, executor *composableRunnable, graph AnyGraph, meta *executorMeta, instance any, opts ...GraphAddNodeOpt) *graphNode { // nolint: byted_s_args_length_limit
	if meta == nil {
		meta = &executorMeta{}
	}

	gn := &graphNode{
		nodeInfo: nodeInfo,

		cr:           executor,
		g:            graph,
		executorMeta: meta,

		instance: instance,
		opts:     opts,
	}

	gn.nodeInfo.name = gn.getNodeName()

	return gn
}
