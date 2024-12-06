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

package indexer

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

// CallbackInput is the input for the indexer callback.
type CallbackInput struct {
	// Docs is the documents to be indexed.
	Docs []*schema.Document
	// Extra is the extra information for the callback.
	Extra map[string]any
}

// CallbackOutput is the output for the indexer callback.
type CallbackOutput struct {
	// IDs is the ids of the indexed documents returned by the indexer.
	IDs []string
	// Extra is the extra information for the callback.
	Extra map[string]any
}

// ConvCallbackInput converts the callback input to the indexer callback input.
func ConvCallbackInput(src callbacks.CallbackInput) *CallbackInput {
	switch t := src.(type) {
	case *CallbackInput:
		return t
	case []*schema.Document:
		return &CallbackInput{
			Docs: t,
		}
	default:
		return nil
	}
}

// ConvCallbackOutput converts the callback output to the indexer callback output.
func ConvCallbackOutput(src callbacks.CallbackOutput) *CallbackOutput {
	switch t := src.(type) {
	case *CallbackOutput:
		return t
	case []string:
		return &CallbackOutput{
			IDs: t,
		}
	default:
		return nil
	}
}

// CallbackHandler is the handler for the indexer callback.
type CallbackHandler struct {
	OnStart func(ctx context.Context, runInfo *callbacks.RunInfo, input *CallbackInput) context.Context
	OnEnd   func(ctx context.Context, runInfo *callbacks.RunInfo, output *CallbackOutput) context.Context
	OnError func(ctx context.Context, runInfo *callbacks.RunInfo, err error) context.Context
}

// Needed checks if the callback handler is needed for the given timing.
func (ch *CallbackHandler) Needed(ctx context.Context, runInfo *callbacks.RunInfo, timing callbacks.CallbackTiming) bool {
	switch timing {
	case callbacks.TimingOnStart:
		return ch.OnStart != nil
	case callbacks.TimingOnEnd:
		return ch.OnEnd != nil
	case callbacks.TimingOnError:
		return ch.OnError != nil
	default:
		return false
	}
}
