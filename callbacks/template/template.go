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

package template

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// NewHandlerHelper creates a new component template handler builder.
// This builder can be used to configure and build a component template handler,
// which can handle callback events for different components with its own struct definition,
// and fallbackTemplate can be used to handle scenarios where none of the cases are hit as a fallback.
func NewHandlerHelper() *HandlerHelper {
	return &HandlerHelper{
		composeTemplates: map[components.Component]*DefaultCallbackHandler{},
	}
}

// HandlerHelper is a builder for creating a callbacks.Handler with specific handlers for different component types.
// create a handler with template.NewHandlerHelper().
// eg.
//
//	helper := template.NewHandlerHelper().
//		ChatModel(&model.CallbackHandler{}).
//		Prompt(&prompt.CallbackHandler{}).
//		Handler()
//
// then use the handler with runnable.Invoke(ctx, input, compose.WithCallbacks(handler))
type HandlerHelper struct {
	promptHandler      *prompt.CallbackHandler
	chatModelHandler   *model.CallbackHandler
	embeddingHandler   *embedding.CallbackHandler
	indexerHandler     *indexer.CallbackHandler
	retrieverHandler   *retriever.CallbackHandler
	loaderHandler      *document.LoaderCallbackHandler
	transformerHandler *document.TransformerCallbackHandler
	toolHandler        *tool.CallbackHandler
	composeTemplates   map[components.Component]*DefaultCallbackHandler
	fallbackTemplate   *DefaultCallbackHandler // execute when not matching any other condition
}

// Handler returns the callbacks.Handler created by HandlerHelper.
func (c *HandlerHelper) Handler() callbacks.Handler {
	return &handlerTemplate{c}
}

// Prompt sets the prompt handler for the handler helper, which will be called when the prompt component is executed.
func (c *HandlerHelper) Prompt(handler *prompt.CallbackHandler) *HandlerHelper {
	c.promptHandler = handler
	return c
}

// ChatModel sets the chat model handler for the handler helper, which will be called when the chat model component is executed.
func (c *HandlerHelper) ChatModel(handler *model.CallbackHandler) *HandlerHelper {
	c.chatModelHandler = handler
	return c
}

// Embedding sets the embedding handler for the handler helper, which will be called when the embedding component is executed.
func (c *HandlerHelper) Embedding(handler *embedding.CallbackHandler) *HandlerHelper {
	c.embeddingHandler = handler
	return c
}

// Indexer sets the indexer handler for the handler helper, which will be called when the indexer component is executed.
func (c *HandlerHelper) Indexer(handler *indexer.CallbackHandler) *HandlerHelper {
	c.indexerHandler = handler
	return c
}

// Retriever sets the retriever handler for the handler helper, which will be called when the retriever component is executed.
func (c *HandlerHelper) Retriever(handler *retriever.CallbackHandler) *HandlerHelper {
	c.retrieverHandler = handler
	return c
}

// Loader sets the loader handler for the handler helper, which will be called when the loader component is executed.
func (c *HandlerHelper) Loader(handler *document.LoaderCallbackHandler) *HandlerHelper {
	c.loaderHandler = handler
	return c
}

// Transformer sets the transformer handler for the handler helper, which will be called when the transformer component is executed.
func (c *HandlerHelper) Transformer(handler *document.TransformerCallbackHandler) *HandlerHelper {
	c.transformerHandler = handler
	return c
}

// Tool sets the tool handler for the handler helper, which will be called when the tool component is executed.
func (c *HandlerHelper) Tool(handler *tool.CallbackHandler) *HandlerHelper {
	c.toolHandler = handler
	return c
}

// Graph sets the graph handler for the handler helper, which will be called when the graph is executed.
func (c *HandlerHelper) Graph(handler *DefaultCallbackHandler) *HandlerHelper {
	c.composeTemplates[compose.ComponentOfGraph] = handler
	return c
}

// StateGraph sets the state graph handler for the handler helper, which will be called when the state graph is executed.
func (c *HandlerHelper) StateGraph(handler *DefaultCallbackHandler) *HandlerHelper {
	c.composeTemplates[compose.ComponentOfStateGraph] = handler
	return c
}

// Chain sets the chain handler for the handler helper, which will be called when the chain is executed.
func (c *HandlerHelper) Chain(handler *DefaultCallbackHandler) *HandlerHelper {
	c.composeTemplates[compose.ComponentOfChain] = handler
	return c
}

// Passthrough sets the passthrough handler for the handler helper, which will be called when the passthrough is executed.
func (c *HandlerHelper) Passthrough(handler *DefaultCallbackHandler) *HandlerHelper {
	c.composeTemplates[compose.ComponentOfPassthrough] = handler
	return c
}

// ToolsNode sets the tools node handler for the handler helper, which will be called when the tools node is executed.
func (c *HandlerHelper) ToolsNode(handler *DefaultCallbackHandler) *HandlerHelper {
	c.composeTemplates[compose.ComponentOfToolsNode] = handler
	return c
}

// Lambda sets the lambda handler for the handler helper, which will be called when the lambda is executed.
func (c *HandlerHelper) Lambda(handler *DefaultCallbackHandler) *HandlerHelper {
	c.composeTemplates[compose.ComponentOfLambda] = handler
	return c
}

// Fallback sets the fallback handler for the handler helper, which will be called when no other handlers are matched.
func (c *HandlerHelper) Fallback(handler *DefaultCallbackHandler) *HandlerHelper {
	c.fallbackTemplate = handler
	return c
}

type handlerTemplate struct {
	*HandlerHelper
}

// OnStart is the callback function for the start event of a component.
// implement the callbacks Handler interface.
func (c *handlerTemplate) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info == nil {
		return ctx
	}

	match := false

	switch info.Component {
	case components.ComponentOfPrompt:
		if c.promptHandler != nil && c.promptHandler.OnStart != nil {
			match = true
			ctx = c.promptHandler.OnStart(ctx, info, prompt.ConvCallbackInput(input))
		}
	case components.ComponentOfChatModel:
		if c.chatModelHandler != nil && c.chatModelHandler.OnStart != nil {
			match = true
			ctx = c.chatModelHandler.OnStart(ctx, info, model.ConvCallbackInput(input))
		}
	case components.ComponentOfEmbedding:
		if c.embeddingHandler != nil && c.embeddingHandler.OnStart != nil {
			match = true
			ctx = c.embeddingHandler.OnStart(ctx, info, embedding.ConvCallbackInput(input))
		}
	case components.ComponentOfIndexer:
		if c.indexerHandler != nil && c.indexerHandler.OnStart != nil {
			match = true
			ctx = c.indexerHandler.OnStart(ctx, info, indexer.ConvCallbackInput(input))
		}
	case components.ComponentOfRetriever:
		if c.retrieverHandler != nil && c.retrieverHandler.OnStart != nil {
			match = true
			ctx = c.retrieverHandler.OnStart(ctx, info, retriever.ConvCallbackInput(input))
		}
	case components.ComponentOfLoader:
		if c.loaderHandler != nil && c.loaderHandler.OnStart != nil {
			match = true
			ctx = c.loaderHandler.OnStart(ctx, info, document.ConvLoaderCallbackInput(input))
		}
	case components.ComponentOfTransformer:
		if c.transformerHandler != nil && c.transformerHandler.OnStart != nil {
			match = true
			ctx = c.transformerHandler.OnStart(ctx, info, document.ConvTransformerCallbackInput(input))
		}
	case components.ComponentOfTool:
		if c.toolHandler != nil && c.toolHandler.OnStart != nil {
			match = true
			ctx = c.toolHandler.OnStart(ctx, info, tool.ConvCallbackInput(input))
		}
	case compose.ComponentOfGraph,
		compose.ComponentOfStateGraph,
		compose.ComponentOfChain,
		compose.ComponentOfPassthrough,
		compose.ComponentOfToolsNode,
		compose.ComponentOfLambda:

		if c.composeTemplates[info.Component] != nil && c.composeTemplates[info.Component].OnStart != nil {
			match = true
			ctx = c.composeTemplates[info.Component].OnStart(ctx, info, input)
		}
	default:

	}

	if !match && c.fallbackTemplate != nil && c.fallbackTemplate.OnStart != nil {
		ctx = c.fallbackTemplate.OnStart(ctx, info, input)
	}

	return ctx
}

// OnEnd is the callback function for the end event of a component.
// implement the callbacks Handler interface.
func (c *handlerTemplate) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info == nil {
		return ctx
	}

	match := false

	switch info.Component {
	case components.ComponentOfPrompt:
		if c.promptHandler != nil && c.promptHandler.OnEnd != nil {
			match = true
			ctx = c.promptHandler.OnEnd(ctx, info, prompt.ConvCallbackOutput(output))
		}
	case components.ComponentOfChatModel:
		if c.chatModelHandler != nil && c.chatModelHandler.OnEnd != nil {
			match = true
			ctx = c.chatModelHandler.OnEnd(ctx, info, model.ConvCallbackOutput(output))
		}
	case components.ComponentOfEmbedding:
		if c.embeddingHandler != nil && c.embeddingHandler.OnEnd != nil {
			match = true
			ctx = c.embeddingHandler.OnEnd(ctx, info, embedding.ConvCallbackOutput(output))
		}
	case components.ComponentOfIndexer:
		if c.indexerHandler != nil && c.indexerHandler.OnEnd != nil {
			match = true
			ctx = c.indexerHandler.OnEnd(ctx, info, indexer.ConvCallbackOutput(output))
		}
	case components.ComponentOfRetriever:
		if c.retrieverHandler != nil && c.retrieverHandler.OnEnd != nil {
			match = true
			ctx = c.retrieverHandler.OnEnd(ctx, info, retriever.ConvCallbackOutput(output))
		}
	case components.ComponentOfLoader:
		if c.loaderHandler != nil && c.loaderHandler.OnEnd != nil {
			match = true
			ctx = c.loaderHandler.OnEnd(ctx, info, document.ConvLoaderCallbackOutput(output))
		}
	case components.ComponentOfTransformer:
		if c.transformerHandler != nil && c.transformerHandler.OnEnd != nil {
			match = true
			ctx = c.transformerHandler.OnEnd(ctx, info, document.ConvTransformerCallbackOutput(output))
		}
	case components.ComponentOfTool:
		if c.toolHandler != nil && c.toolHandler.OnEnd != nil {
			match = true
			ctx = c.toolHandler.OnEnd(ctx, info, tool.ConvCallbackOutput(output))
		}
	case compose.ComponentOfGraph,
		compose.ComponentOfStateGraph,
		compose.ComponentOfChain,
		compose.ComponentOfPassthrough,
		compose.ComponentOfToolsNode,
		compose.ComponentOfLambda:

		if c.composeTemplates[info.Component] != nil && c.composeTemplates[info.Component].OnEnd != nil {
			match = true
			ctx = c.composeTemplates[info.Component].OnEnd(ctx, info, output)
		}
	default:

	}

	if !match && c.fallbackTemplate != nil && c.fallbackTemplate.OnEnd != nil {
		ctx = c.fallbackTemplate.OnEnd(ctx, info, output)
	}

	return ctx
}

// OnError is the callback function for the error event of a component.
// implement the callbacks Handler interface.
func (c *handlerTemplate) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info == nil {
		return ctx
	}

	match := false

	switch info.Component {
	case components.ComponentOfPrompt:
		if c.promptHandler != nil && c.promptHandler.OnError != nil {
			match = true
			ctx = c.promptHandler.OnError(ctx, info, err)
		}
	case components.ComponentOfChatModel:
		if c.chatModelHandler != nil && c.chatModelHandler.OnError != nil {
			match = true
			ctx = c.chatModelHandler.OnError(ctx, info, err)
		}
	case components.ComponentOfEmbedding:
		if c.embeddingHandler != nil && c.embeddingHandler.OnError != nil {
			match = true
			ctx = c.embeddingHandler.OnError(ctx, info, err)
		}
	case components.ComponentOfIndexer:
		if c.indexerHandler != nil && c.indexerHandler.OnError != nil {
			match = true
			ctx = c.indexerHandler.OnError(ctx, info, err)
		}
	case components.ComponentOfRetriever:
		if c.retrieverHandler != nil && c.retrieverHandler.OnError != nil {
			match = true
			ctx = c.retrieverHandler.OnError(ctx, info, err)
		}
	case components.ComponentOfLoader:
		if c.loaderHandler != nil && c.loaderHandler.OnError != nil {
			match = true
			ctx = c.loaderHandler.OnError(ctx, info, err)
		}
	case components.ComponentOfTransformer:
		if c.transformerHandler != nil && c.transformerHandler.OnError != nil {
			match = true
			ctx = c.transformerHandler.OnError(ctx, info, err)
		}
	case components.ComponentOfTool:
		if c.toolHandler != nil && c.toolHandler.OnError != nil {
			match = true
			ctx = c.toolHandler.OnError(ctx, info, err)
		}
	case compose.ComponentOfGraph,
		compose.ComponentOfStateGraph,
		compose.ComponentOfChain,
		compose.ComponentOfPassthrough,
		compose.ComponentOfToolsNode,
		compose.ComponentOfLambda:

		if c.composeTemplates[info.Component] != nil && c.composeTemplates[info.Component].OnError != nil {
			match = true
			ctx = c.composeTemplates[info.Component].OnError(ctx, info, err)
		}
	default:

	}

	if !match && c.fallbackTemplate != nil && c.fallbackTemplate.OnError != nil {
		ctx = c.fallbackTemplate.OnError(ctx, info, err)
	}

	return ctx
}

// OnStartWithStreamInput is the callback function for the start event of a component with stream input.
// implement the callbacks Handler interface.
func (c *handlerTemplate) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	match := false
	defer func() {
		if !match {
			input.Close()
		}
	}()

	if info == nil {
		return ctx
	}

	switch info.Component {
	// currently no components.Component receive stream as input
	case compose.ComponentOfGraph,
		compose.ComponentOfStateGraph,
		compose.ComponentOfChain,
		compose.ComponentOfPassthrough,
		compose.ComponentOfToolsNode,
		compose.ComponentOfLambda:
		if c.composeTemplates[info.Component] != nil && c.composeTemplates[info.Component].OnStartWithStreamInput != nil {
			match = true
			ctx = c.composeTemplates[info.Component].OnStartWithStreamInput(ctx, info, input)
		}
	default:

	}

	if !match && c.fallbackTemplate != nil && c.fallbackTemplate.OnStartWithStreamInput != nil {
		match = true
		ctx = c.fallbackTemplate.OnStartWithStreamInput(ctx, info, input)
	}

	return ctx
}

// OnEndWithStreamOutput is the callback function for the end event of a component with stream output.
// implement the callbacks Handler interface.
func (c *handlerTemplate) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	match := false
	defer func() {
		if !match {
			output.Close()
		}
	}()

	if info == nil {
		return ctx
	}

	switch info.Component {
	case components.ComponentOfChatModel:
		if c.chatModelHandler != nil && c.chatModelHandler.OnEndWithStreamOutput != nil {
			match = true
			ctx = c.chatModelHandler.OnEndWithStreamOutput(ctx, info,
				schema.StreamReaderWithConvert(output, func(item callbacks.CallbackOutput) (*model.CallbackOutput, error) {
					return model.ConvCallbackOutput(item), nil
				}))
		}

	case components.ComponentOfTool:
		if c.toolHandler != nil && c.toolHandler.OnEndWithStreamOutput != nil {
			match = true
			ctx = c.toolHandler.OnEndWithStreamOutput(ctx, info,
				schema.StreamReaderWithConvert(output, func(item callbacks.CallbackOutput) (*tool.CallbackOutput, error) {
					return tool.ConvCallbackOutput(item), nil
				}))
		}
	case compose.ComponentOfGraph,
		compose.ComponentOfStateGraph,
		compose.ComponentOfChain,
		compose.ComponentOfPassthrough,
		compose.ComponentOfToolsNode,
		compose.ComponentOfLambda:
		if c.composeTemplates[info.Component] != nil && c.composeTemplates[info.Component].OnEndWithStreamOutput != nil {
			match = true
			ctx = c.composeTemplates[info.Component].OnEndWithStreamOutput(ctx, info, output)
		}

	default:

	}

	if !match && c.fallbackTemplate != nil && c.fallbackTemplate.OnEndWithStreamOutput != nil {
		match = true
		ctx = c.fallbackTemplate.OnEndWithStreamOutput(ctx, info, output)
	}

	return ctx
}

// Needed checks if the callback handler is needed for the given timing.
func (c *handlerTemplate) Needed(ctx context.Context, info *callbacks.RunInfo, timing callbacks.CallbackTiming) bool {
	switch info.Component {
	case components.ComponentOfChatModel:
		if c.chatModelHandler != nil && c.chatModelHandler.Needed(ctx, info, timing) {
			return true
		}
	case components.ComponentOfEmbedding:
		if c.embeddingHandler != nil && c.embeddingHandler.Needed(ctx, info, timing) {
			return true
		}
	case components.ComponentOfIndexer:
		if c.indexerHandler != nil && c.indexerHandler.Needed(ctx, info, timing) {
			return true
		}
	case components.ComponentOfLoader:
		if c.loaderHandler != nil && c.loaderHandler.Needed(ctx, info, timing) {
			return true
		}
	case components.ComponentOfPrompt:
		if c.promptHandler != nil && c.promptHandler.Needed(ctx, info, timing) {
			return true
		}
	case components.ComponentOfRetriever:
		if c.retrieverHandler != nil && c.retrieverHandler.Needed(ctx, info, timing) {
			return true
		}
	case components.ComponentOfTool:
		if c.toolHandler != nil && c.toolHandler.Needed(ctx, info, timing) {
			return true
		}
	case components.ComponentOfTransformer:
		if c.transformerHandler != nil && c.transformerHandler.Needed(ctx, info, timing) {
			return true
		}
	case compose.ComponentOfGraph,
		compose.ComponentOfStateGraph,
		compose.ComponentOfChain,
		compose.ComponentOfPassthrough,
		compose.ComponentOfToolsNode,
		compose.ComponentOfLambda:
		template := c.composeTemplates[info.Component]
		if template != nil && template.Needed(ctx, info, timing) {
			return true
		}
	default:

	}

	if c.fallbackTemplate != nil {
		return c.fallbackTemplate.Needed(ctx, info, timing)
	}

	return false
}
