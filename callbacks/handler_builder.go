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

package callbacks

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// HandlerBuilder can be used to build a Handler with callback functions.
// e.g.
//
//	handler := &HandlerBuilder{
//		OnStartFn: func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {} // self defined start callback function
//	}
//
//	graph := compose.NewGraph[inputType, outputType]()
//	runnable, err := graph.Compile()
//	if err != nil {...}
//	runnable.Invoke(ctx, params, compose.WithCallback(handler)) // => only implement functions which you want to override
//
// Deprecated: In most situations, it is preferred to use template.NewHandlerHelper. Otherwise, use NewHandlerBuilder().OnStartFn()...Build().
type HandlerBuilder struct {
	OnStartFn                func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
	OnEndFn                  func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
	OnErrorFn                func(ctx context.Context, info *RunInfo, err error) context.Context
	OnStartWithStreamInputFn func(ctx context.Context, info *RunInfo, input *schema.StreamReader[CallbackInput]) context.Context
	OnEndWithStreamOutputFn  func(ctx context.Context, info *RunInfo, output *schema.StreamReader[CallbackOutput]) context.Context
}

func (h *HandlerBuilder) OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
	if h.OnStartFn != nil {
		return h.OnStartFn(ctx, info, input)
	}

	return ctx
}

func (h *HandlerBuilder) OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
	if h.OnEndFn != nil {
		return h.OnEndFn(ctx, info, output)
	}

	return ctx
}

func (h *HandlerBuilder) OnError(ctx context.Context, info *RunInfo, err error) context.Context {
	if h.OnErrorFn != nil {
		return h.OnErrorFn(ctx, info, err)
	}

	return ctx
}

func (h *HandlerBuilder) OnStartWithStreamInput(ctx context.Context, info *RunInfo, input *schema.StreamReader[CallbackInput]) context.Context {
	if h.OnStartWithStreamInputFn != nil {
		return h.OnStartWithStreamInputFn(ctx, info, input)
	}

	input.Close()

	return ctx
}

func (h *HandlerBuilder) OnEndWithStreamOutput(ctx context.Context, info *RunInfo, output *schema.StreamReader[CallbackOutput]) context.Context {
	if h.OnEndWithStreamOutputFn != nil {
		return h.OnEndWithStreamOutputFn(ctx, info, output)
	}

	output.Close()

	return ctx
}

type handlerBuilder struct {
	onStartFn                func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
	onEndFn                  func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
	onErrorFn                func(ctx context.Context, info *RunInfo, err error) context.Context
	onStartWithStreamInputFn func(ctx context.Context, info *RunInfo, input *schema.StreamReader[CallbackInput]) context.Context
	onEndWithStreamOutputFn  func(ctx context.Context, info *RunInfo, output *schema.StreamReader[CallbackOutput]) context.Context
}

func (hb *handlerBuilder) OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
	if hb.onStartFn != nil {
		return hb.onStartFn(ctx, info, input)
	}

	return ctx
}

func (hb *handlerBuilder) OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
	if hb.onEndFn != nil {
		return hb.onEndFn(ctx, info, output)
	}

	return ctx
}

func (hb *handlerBuilder) OnError(ctx context.Context, info *RunInfo, err error) context.Context {
	if hb.onErrorFn != nil {
		return hb.onErrorFn(ctx, info, err)
	}

	return ctx
}

func (hb *handlerBuilder) OnStartWithStreamInput(ctx context.Context, info *RunInfo, input *schema.StreamReader[CallbackInput]) context.Context {
	if hb.onStartWithStreamInputFn != nil {
		return hb.onStartWithStreamInputFn(ctx, info, input)
	}

	input.Close()

	return ctx
}

func (hb *handlerBuilder) OnEndWithStreamOutput(ctx context.Context, info *RunInfo, output *schema.StreamReader[CallbackOutput]) context.Context {
	if hb.onEndWithStreamOutputFn != nil {
		return hb.onEndWithStreamOutputFn(ctx, info, output)
	}

	output.Close()

	return ctx
}

func (hb *handlerBuilder) Needed(_ context.Context, _ *RunInfo, timing CallbackTiming) bool {
	switch timing {
	case TimingOnStart:
		return hb.onStartFn != nil
	case TimingOnEnd:
		return hb.onEndFn != nil
	case TimingOnError:
		return hb.onErrorFn != nil
	case TimingOnStartWithStreamInput:
		return hb.onStartWithStreamInputFn != nil
	case TimingOnEndWithStreamOutput:
		return hb.onEndWithStreamOutputFn != nil
	default:
		return false
	}
}

func NewHandlerBuilder() *handlerBuilder {
	return &handlerBuilder{}
}

func (hb *handlerBuilder) OnStartFn(fn func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context) *handlerBuilder {
	hb.onStartFn = fn
	return hb
}

func (hb *handlerBuilder) OnEndFn(fn func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context) *handlerBuilder {
	hb.onEndFn = fn
	return hb
}

func (hb *handlerBuilder) OnErrorFn(fn func(ctx context.Context, info *RunInfo, err error) context.Context) *handlerBuilder {
	hb.onErrorFn = fn
	return hb
}

func (hb *handlerBuilder) OnStartWithStreamInputFn(fn func(ctx context.Context, info *RunInfo, input *schema.StreamReader[CallbackInput]) context.Context) *handlerBuilder {
	hb.onStartWithStreamInputFn = fn
	return hb
}

func (hb *handlerBuilder) OnEndWithStreamOutputFn(fn func(ctx context.Context, info *RunInfo, output *schema.StreamReader[CallbackOutput]) context.Context) *handlerBuilder {
	hb.onEndWithStreamOutputFn = fn
	return hb
}

// Build returns a Handler with the functions set in the builder.
func (hb *handlerBuilder) Build() Handler {
	return hb
}
