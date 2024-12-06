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

package react

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
)

type nodeState struct {
	Messages []*schema.Message
}

// MessageModifier modify the input messages before the model is called.
type MessageModifier func(ctx context.Context, input []*schema.Message) []*schema.Message

// AgentConfig is the config for react agent.
type AgentConfig struct {
	// Model is the chat model to be used for handling user messages.
	Model model.ChatModel
	// ToolsConfig is the config for tools node.
	ToolsConfig compose.ToolsNodeConfig

	// MessageModifier.
	// modify the input messages before the model is called, it's useful when you want to add some system prompt or other messages.
	MessageModifier MessageModifier

	// MaxStep.
	// default 12 of steps in pregel (node num + 10).
	MaxStep int `json:"max_step"`

	// Tools that will make agent return directly when the tool is called.
	ToolReturnDirectly map[string]struct{}
}

// NewPersonaModifier add the system prompt as persona before the model is called.
// example:
//
//	persona := "You are an expert in golang."
//	config := AgentConfig{
//		Model: model,
//		MessageModifier: NewPersonaModifier(persona),
//	}
//	agent, err := NewAgent(ctx, config)
//	if err != nil {return}
//	msg, err := agent.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "how to build agent with eino"}})
//	if err != nil {return}
//	println(msg.Content)
func NewPersonaModifier(persona string) MessageModifier {
	return func(ctx context.Context, input []*schema.Message) []*schema.Message {
		res := make([]*schema.Message, 0, len(input)+1)

		res = append(res, schema.SystemMessage(persona))
		res = append(res, input...)
		return res
	}
}

// NewAgent creates a react agent.
func NewAgent(ctx context.Context, config *AgentConfig) (*Agent, error) {
	agent := &Agent{}

	runnable, err := agent.build(ctx, config)
	if err != nil {
		return nil, err
	}

	agent.runnable = runnable

	return agent, nil
}

// Agent is the react agent.
// React agent is a simple agent that handles user messages with a chat model and tools.
// react will call the chat model, if the message contains tool calls, it will call the tools.
// if the tool is configured to return directly, react will return directly.
// otherwise, react will continue to call the chat model until the message contains no tool calls.
// eg.
//
//	agent, err := react.NewAgent(ctx, &react.AgentConfig{})
//	if err != nil {...}
//	msg, err := agent.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "how to build agent with eino"}})
//	if err != nil {...}
//	println(msg.Content)
type Agent struct {
	runnable compose.Runnable[[]*schema.Message, *schema.Message]
}

func (r *Agent) build(ctx context.Context, config *AgentConfig) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
	var (
		nodeKeyTools     = "tools"
		nodeKeyChatModel = "chat"
	)

	if config.MessageModifier == nil {
		config.MessageModifier = func(ctx context.Context, input []*schema.Message) []*schema.Message {
			return input
		}
	}

	toolInfos := make([]*schema.ToolInfo, 0, len(config.ToolsConfig.Tools))
	for _, t := range config.ToolsConfig.Tools {
		tl, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}

		toolInfos = append(toolInfos, tl)
	}

	err := config.Model.BindTools(toolInfos)
	if err != nil {
		return nil, err
	}

	// graph
	graph := compose.NewStateGraph[[]*schema.Message, *schema.Message](func(ctx context.Context) *nodeState {
		s := &nodeState{
			Messages: make([]*schema.Message, 0, 3),
		}
		return s
	})

	err = graph.AddChatModelNode(nodeKeyChatModel, config.Model,
		compose.WithStatePreHandler(func(ctx context.Context, input []*schema.Message, state *nodeState) ([]*schema.Message, error) {
			state.Messages = append(state.Messages, input...)

			modifiedInput := make([]*schema.Message, 0, len(input))
			modifiedInput = append(modifiedInput, state.Messages...)
			modifiedInput = config.MessageModifier(ctx, modifiedInput)

			return modifiedInput, nil
		}),
	)
	if err != nil {
		return nil, err
	}

	toolsNode, err := compose.NewToolNode(ctx, &config.ToolsConfig)
	if err != nil {
		return nil, err
	}

	err = graph.AddToolsNode(nodeKeyTools, toolsNode, compose.WithStatePreHandler(func(ctx context.Context, input *schema.Message, state *nodeState) (*schema.Message, error) {
		state.Messages = append(state.Messages, input)

		if len(config.ToolReturnDirectly) > 0 {
			if err := checkReturnDirectlyBeforeToolsNode(input, config); err != nil {
				return nil, err
			}
		}

		if err := cacheToolCallInfo(ctx, input.ToolCalls); err != nil {
			return nil, err
		}

		return input, nil
	}))
	if err != nil {
		return nil, err
	}

	err = graph.AddEdge(compose.START, nodeKeyChatModel)
	if err != nil {
		return nil, err
	}

	err = graph.AddBranch(nodeKeyChatModel, compose.NewStreamGraphBranch(func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (endNode string, err error) {
		msg, err := sr.Recv()
		if err != nil {
			return "", err
		}
		defer sr.Close()

		if len(msg.ToolCalls) == 0 {
			return compose.END, nil
		}

		return nodeKeyTools, nil
	}, map[string]bool{nodeKeyTools: true, compose.END: true}))
	if err != nil {
		return nil, err
	}

	if len(config.ToolReturnDirectly) > 0 {
		returnDirectlyConvertor := func(ctx context.Context, msgs *schema.StreamReader[[]*schema.Message]) (*schema.StreamReader[*schema.Message], error) {
			flattened := schema.StreamReaderWithConvert(msgs, func(msgs []*schema.Message) (*schema.Message, error) {
				if len(msgs) != 1 {
					return nil, fmt.Errorf("return directly tools node output expected to have only one msg, but got %d", len(msgs))
				}
				return msgs[0], nil
			})

			return flattened, nil
		}

		nodeKeyConvertor := "convertor"
		err = graph.AddLambdaNode(nodeKeyConvertor, compose.TransformableLambda(returnDirectlyConvertor))
		if err != nil {
			return nil, err
		}

		err = graph.AddBranch(nodeKeyTools, compose.NewStreamGraphBranch(func(ctx context.Context, msgsStream *schema.StreamReader[[]*schema.Message]) (endNode string, err error) {
			defer msgsStream.Close()

			msgs, err := msgsStream.Recv()
			if err != nil {
				return "", fmt.Errorf("receive first packet from tools node result returns err: %w", err)
			}

			if len(msgs) == 0 {
				return "", errors.New("receive first package from tools node result returns empty msgs")
			}

			msg := msgs[0]
			toolCallID := msg.ToolCallID
			if len(toolCallID) == 0 {
				return "", errors.New("receive first package from tools node result returns empty tool call id")
			}

			toolCall, err := getToolCallInfo(ctx, toolCallID)
			if err != nil {
				return "", fmt.Errorf("get tool call info for tool call id: %s returns err: %w", toolCallID, err)
			}

			if _, ok := config.ToolReturnDirectly[toolCall.Function.Name]; ok { // return directly will appear in first message
				return nodeKeyConvertor, nil
			}

			return nodeKeyChatModel, nil
		}, map[string]bool{nodeKeyChatModel: true, nodeKeyConvertor: true}))
		if err != nil {
			return nil, err
		}

		if err = graph.AddEdge(nodeKeyConvertor, compose.END); err != nil {
			return nil, err
		}
	} else {
		if err = graph.AddEdge(nodeKeyTools, nodeKeyChatModel); err != nil {
			return nil, err
		}
	}

	var opts []compose.GraphCompileOption
	if config.MaxStep > 0 {
		opts = append(opts, compose.WithMaxRunSteps(config.MaxStep))
	}

	runnable, err := graph.Compile(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return runnable, nil
}

type toolCallInfoKey struct{}

func cacheToolCallInfo(ctx context.Context, toolCalls []schema.ToolCall) error {
	info := ctx.Value(toolCallInfoKey{})
	if info == nil {
		return errors.New("tool call info not found in context")
	}

	toolCallInfo, ok := info.(*map[string]schema.ToolCall)
	if !ok {
		return fmt.Errorf("tool call info type error, not atomic.Value: %v", reflect.TypeOf(info))
	}

	m := make(map[string]schema.ToolCall, len(toolCalls))
	for i := range toolCalls {
		m[toolCalls[i].ID] = toolCalls[i]
	}

	*toolCallInfo = m

	return nil
}

func getToolCallInfo(ctx context.Context, toolCallID string) (*schema.ToolCall, error) {
	info := ctx.Value(toolCallInfoKey{})
	if info == nil {
		return nil, errors.New("tool call info not found in context")
	}

	toolCallInfo, ok := info.(*map[string]schema.ToolCall)
	if !ok {
		return nil, fmt.Errorf("tool call info type error, not map[string]schema.ToolCall: %v", reflect.TypeOf(info))
	}

	if toolCallInfo == nil {
		return nil, errors.New("tool call info is nil")
	}

	toolCall, ok := (*toolCallInfo)[toolCallID]
	if !ok {
		return nil, fmt.Errorf("tool call info not found for tool call id: %s", toolCallID)
	}

	return &toolCall, nil
}

func checkReturnDirectlyBeforeToolsNode(input *schema.Message, config *AgentConfig) error {
	if len(input.ToolCalls) > 1 { // check if a return directly tool call belongs to a batch of parallel tool calls, which is not supported for now
		var returnDirectly bool
		toolCalls := input.ToolCalls
		toolNames := make([]string, 0, len(toolCalls))
		for i := range toolCalls {
			toolNames = append(toolNames, toolCalls[i].Function.Name)

			if _, ok := config.ToolReturnDirectly[toolCalls[i].Function.Name]; ok {
				returnDirectly = true
			}
		}

		if returnDirectly {
			return fmt.Errorf("return directly tool call is not allowed when there are parallel tool calls: %v", toolNames)
		}
	}

	return nil
}

// Generate generates a response from the agent.
func (r *Agent) Generate(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (output *schema.Message, err error) {
	m := make(map[string]schema.ToolCall, 0)
	ctx = context.WithValue(ctx, toolCallInfoKey{}, &m)

	output, err = r.runnable.Invoke(ctx, input, agent.GetComposeOptions(opts...)...)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// Stream calls the agent and returns a stream response.
func (r *Agent) Stream(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (
	output *schema.StreamReader[*schema.Message], err error) {
	m := make(map[string]schema.ToolCall, 0)
	ctx = context.WithValue(ctx, toolCallInfoKey{}, &m)

	res, err := r.runnable.Stream(ctx, input, agent.GetComposeOptions(opts...)...)
	if err != nil {
		return nil, err
	}

	return res, nil
}
