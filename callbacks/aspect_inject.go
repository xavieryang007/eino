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

// OnStart Fast inject callback input / output aspect for component developer
// e.g.
//
//	func (t *testchatmodel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (resp *schema.Message, err error) {
//		defer func() {
//			if err != nil {
//				callbacks.OnEnd(ctx, err)
//			}
//		}()
//
//		ctx = callbacks.OnStart(ctx, &model.CallbackInput{
//			Messages: input,
//			Tools:    nil,
//			Extra:    nil,
//		})
//
//		// do smt
//
//		ctx = callbacks.OnEnd(ctx, &model.CallbackOutput{
//			Message: resp,
//			Extra:   nil,
//		})
//
//		return resp, nil
//	}
//
// OnStart invokes the OnStart logic for the particular context, ensuring that all registered
// handlers are executed in reverse order (compared to add order) when a process begins.
func OnStart(ctx context.Context, input CallbackInput) context.Context {
	mgr, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}

	for i := len(mgr.handlers) - 1; i >= 0; i-- {
		handler := mgr.handlers[i]
		timingChecker, ok := handler.(TimingChecker)
		if !ok || timingChecker.Needed(ctx, mgr.runInfo, TimingOnStart) {
			ctx = handler.OnStart(ctx, mgr.runInfo, input)
		}
	}

	return ctx
}

// OnEnd invokes the OnEnd logic of the particular context, allowing for proper cleanup
// and finalization when a process ends.
// handlers are executed in normal order (compared to add order).
func OnEnd(ctx context.Context, output CallbackOutput) context.Context {
	mgr, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}

	for i := 0; i < len(mgr.handlers); i++ {
		handler := mgr.handlers[i]
		timingChecker, ok := handler.(TimingChecker)
		if !ok || timingChecker.Needed(ctx, mgr.runInfo, TimingOnEnd) {
			ctx = handler.OnEnd(ctx, mgr.runInfo, output)
		}
	}

	return ctx
}

// OnStartWithStreamInput invokes the OnStartWithStreamInput logic of the particular context, ensuring that
// every input stream should be closed properly in handler.
// handlers are executed in reverse order (compared to add order).
func OnStartWithStreamInput[T any](ctx context.Context, input *schema.StreamReader[T]) (
	nextCtx context.Context, newStreamReader *schema.StreamReader[T]) {

	mgr, ok := managerFromCtx(ctx)
	if !ok {
		return ctx, input
	}

	if len(mgr.handlers) == 0 {
		return ctx, input
	}

	var neededHandlers []Handler
	for i := range mgr.handlers {
		h := mgr.handlers[i]
		timingChecker, ok := h.(TimingChecker)
		if !ok || timingChecker.Needed(ctx, mgr.runInfo, TimingOnStartWithStreamInput) {
			neededHandlers = append(neededHandlers, h)
		}
	}

	if len(neededHandlers) == 0 {
		return ctx, input
	}

	cp := input.Copy(len(neededHandlers) + 1)
	for i := len(neededHandlers) - 1; i >= 0; i-- {
		h := neededHandlers[i]
		ctx = h.OnStartWithStreamInput(ctx, mgr.runInfo, schema.StreamReaderWithConvert(cp[i], func(src T) (CallbackInput, error) {
			return src, nil
		}))
	}

	return ctx, cp[len(cp)-1]
}

// OnEndWithStreamOutput invokes the OnEndWithStreamOutput logic of the particular, ensuring that
// every input stream should be closed properly in handler.
// handlers are executed in normal order (compared to add order).
func OnEndWithStreamOutput[T any](ctx context.Context, output *schema.StreamReader[T]) (
	nextCtx context.Context, newStreamReader *schema.StreamReader[T]) {

	mgr, ok := managerFromCtx(ctx)
	if !ok {
		return ctx, output
	}

	if len(mgr.handlers) == 0 {
		return ctx, output
	}

	var neededHandlers []Handler
	for i := range mgr.handlers {
		h := mgr.handlers[i]
		timingChecker, ok := h.(TimingChecker)
		if !ok || timingChecker.Needed(ctx, mgr.runInfo, TimingOnEndWithStreamOutput) {
			neededHandlers = append(neededHandlers, h)
		}
	}

	if len(neededHandlers) == 0 {
		return ctx, output
	}

	cp := output.Copy(len(neededHandlers) + 1)
	for i := 0; i < len(neededHandlers); i++ {
		h := neededHandlers[i]
		ctx = h.OnEndWithStreamOutput(ctx, mgr.runInfo, schema.StreamReaderWithConvert(cp[i], func(src T) (CallbackOutput, error) {
			return src, nil
		}))
	}

	return ctx, cp[len(cp)-1]
}

// OnError invokes the OnError logic of the particular, notice that error in stream will not represent here.
// handlers are executed in normal order (compared to add order).
func OnError(ctx context.Context, err error) context.Context {
	mgr, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}

	for i := 0; i < len(mgr.handlers); i++ {
		handler := mgr.handlers[i]
		timingChecker, ok := handler.(TimingChecker)
		if !ok || timingChecker.Needed(ctx, mgr.runInfo, TimingOnError) {
			ctx = handler.OnError(ctx, mgr.runInfo, err)
		}
	}

	return ctx
}

// SetRunInfo sets the RunInfo to be passed to Handler.
func SetRunInfo(ctx context.Context, info *RunInfo) context.Context {
	cbm, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}

	return ctxWithManager(ctx, cbm.withRunInfo(info))
}

// InitCallbacks initializes a new context with the provided RunInfo and handlers.
// Any previously set RunInfo and Handlers for this ctx will be overwritten.
func InitCallbacks(ctx context.Context, info *RunInfo, handlers ...Handler) context.Context {
	mgr, ok := newManager(info, handlers...)
	if ok {
		return ctxWithManager(ctx, mgr)
	}

	return ctxWithManager(ctx, nil)
}
