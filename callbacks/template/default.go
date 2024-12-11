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
	"github.com/cloudwego/eino/schema"
)

// DefaultCallbackHandler is the default callback handler implementation, can be used for callback handler builder in template.HandlerHelper (for example, Graph, StateGraph, Chain, Lambda, etc.).
type DefaultCallbackHandler struct {
	OnStart                func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context
	OnStartWithStreamInput func(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context
	OnEnd                  func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context
	OnEndWithStreamOutput  func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context
	OnError                func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context
}

// Needed checks if the callback handler is needed for the given timing.
func (d *DefaultCallbackHandler) Needed(_ context.Context, _ *callbacks.RunInfo, timing callbacks.CallbackTiming) bool {
	switch timing {
	case callbacks.TimingOnStart:
		return d.OnStart != nil
	case callbacks.TimingOnEnd:
		return d.OnEnd != nil
	case callbacks.TimingOnError:
		return d.OnError != nil
	case callbacks.TimingOnStartWithStreamInput:
		return d.OnStartWithStreamInput != nil
	case callbacks.TimingOnEndWithStreamOutput:
		return d.OnEndWithStreamOutput != nil
	default:
		return false
	}
}
