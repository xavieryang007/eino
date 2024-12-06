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

package tool

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

// CallbackInput is the input for the tool callback.
type CallbackInput struct {
	// ArgumentsInJSON is the arguments in json format for the tool.
	ArgumentsInJSON string
	// Extra is the extra information for the tool.
	Extra map[string]any
}

// CallbackOutput is the output for the tool callback.
type CallbackOutput struct {
	// Response is the response for the tool.
	Response string
	// Extra is the extra information for the tool.
	Extra map[string]any
}

// ConvCallbackInput converts the callback input to the tool callback input.
func ConvCallbackInput(src callbacks.CallbackInput) *CallbackInput {
	switch t := src.(type) {
	case *CallbackInput:
		return t
	case string:
		return &CallbackInput{ArgumentsInJSON: t}
	default:
		return nil
	}
}

// ConvCallbackOutput converts the callback output to the tool callback output.
func ConvCallbackOutput(src callbacks.CallbackOutput) *CallbackOutput {
	switch t := src.(type) {
	case *CallbackOutput:
		return t
	case string:
		return &CallbackOutput{Response: t}
	default:
		return nil
	}
}

// CallbackHandler is the handler for the tool callback.
type CallbackHandler struct {
	OnStart               func(ctx context.Context, info *callbacks.RunInfo, input *CallbackInput) context.Context
	OnEnd                 func(ctx context.Context, info *callbacks.RunInfo, input *CallbackOutput) context.Context
	OnEndWithStreamOutput func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*CallbackOutput]) context.Context
	OnError               func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context
}

// Needed checks if the callback handler is needed for the given timing.
func (ch *CallbackHandler) Needed(ctx context.Context, runInfo *callbacks.RunInfo, timing callbacks.CallbackTiming) bool {
	switch timing {
	case callbacks.TimingOnStart:
		return ch.OnStart != nil
	case callbacks.TimingOnEnd:
		return ch.OnEnd != nil
	case callbacks.TimingOnEndWithStreamOutput:
		return ch.OnEndWithStreamOutput != nil
	case callbacks.TimingOnError:
		return ch.OnError != nil
	default:
		return false
	}
}
