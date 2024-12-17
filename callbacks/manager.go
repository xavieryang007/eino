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

	"github.com/cloudwego/eino/callbacks/internal"
	"github.com/cloudwego/eino/schema"
)

// Manager is a callback manager of one running node.
// Deprecated: Manager will become the inner conception, use methods in aspect_inject.go instead
type Manager struct {
	*manager
}

type manager struct {
	handlers []Handler
	runInfo  *RunInfo
}

// NewManager creates a callback manager.
// It will return a nil manager if no callback handler is provided, please check the return value first before using.
// Deprecated: Manager will become the inner conception, use methods in aspect_inject.go instead
func NewManager(runInfo *RunInfo, handlers ...Handler) (*Manager, bool) {
	m, ok := newManager(runInfo, handlers...)
	if !ok {
		return nil, false
	}

	return &Manager{
		manager: m,
	}, true
}

func newManager(runInfo *RunInfo, handlers ...Handler) (*manager, bool) {
	l := len(handlers) + len(globalHandlers)
	if l == 0 {
		return nil, false
	}
	hs := make([]Handler, 0, l)
	hs = append(hs, globalHandlers...)
	hs = append(hs, handlers...)

	return &manager{
		handlers: hs,
		runInfo:  runInfo,
	}, true
}

func (m *manager) Handlers() []Handler {
	return m.handlers
}

// Deprecated: Manager will become the inner conception, use methods in aspect_inject.go instead
func (mm *Manager) WithRunInfo(runInfo *RunInfo) *Manager {
	if mm == nil {
		return nil
	}

	m := mm.manager.withRunInfo(runInfo)

	return &Manager{
		manager: m,
	}
}

func (m *manager) withRunInfo(runInfo *RunInfo) *manager {
	if m == nil {
		return nil
	}

	return &manager{
		handlers: m.handlers,
		runInfo:  runInfo,
	}
}

func (m *manager) appendHandlers(handlers ...Handler) *manager {
	if m == nil {
		return nil
	}

	return &manager{
		handlers: append(m.handlers, handlers...),
		runInfo:  m.runInfo,
	}
}

// Deprecated: Manager will become the inner conception, use methods in aspect_inject.go instead
func ManagerFromCtx(ctx context.Context) (*Manager, bool) {
	internalM, ok := managerFromCtx(ctx)
	if ok {
		return &Manager{
			manager: internalM,
		}, true
	}

	return nil, false
}

func managerFromCtx(ctx context.Context) (*manager, bool) {
	v := ctx.Value(internal.CtxManagerKey{})
	m, ok := v.(*manager)
	if ok && m != nil {
		return &manager{
			handlers: m.handlers,
			runInfo:  m.runInfo,
		}, true
	}

	return nil, false
}

// Deprecated: Manager will become the inner conception, use methods in aspect_inject.go instead
func CtxWithManager(ctx context.Context, manager *Manager) context.Context {
	return ctxWithManager(ctx, manager.manager)
}

func ctxWithManager(ctx context.Context, manager *manager) context.Context {
	return context.WithValue(ctx, internal.CtxManagerKey{}, manager)
}

func (m *manager) OnStart(ctx context.Context, input CallbackInput) context.Context {
	if m == nil {
		return ctx
	}

	for i := len(m.handlers) - 1; i >= 0; i-- {
		handler := m.handlers[i]
		ctx = handler.OnStart(ctx, m.runInfo, input)
	}

	return ctx
}

func (m *manager) OnEnd(ctx context.Context, output CallbackOutput) context.Context {
	if m == nil {
		return ctx
	}

	for i := 0; i < len(m.handlers); i++ {
		handler := m.handlers[i]
		ctx = handler.OnEnd(ctx, m.runInfo, output)
	}

	return ctx
}

func (m *manager) OnError(ctx context.Context, err error) context.Context {
	if m == nil {
		return ctx
	}

	for i := 0; i < len(m.handlers); i++ {
		handler := m.handlers[i]
		ctx = handler.OnError(ctx, m.runInfo, err)
	}

	return ctx
}

func (m *manager) OnStartWithStreamInput(
	ctx context.Context, input *schema.StreamReader[CallbackInput]) context.Context {
	if m == nil {
		if input != nil {
			input.Close()
		}
		return ctx
	}

	if len(m.handlers) == 0 {
		input.Close()
		return ctx
	}

	ins := input.Copy(len(m.handlers))
	for i := len(m.handlers) - 1; i >= 0; i-- {
		handler := m.handlers[i]
		ctx = handler.OnStartWithStreamInput(ctx, m.runInfo, ins[i])
	}

	return ctx
}

func (m *manager) OnEndWithStreamOutput(
	ctx context.Context, output *schema.StreamReader[CallbackOutput]) context.Context {
	if m == nil {
		if output != nil {
			output.Close()
		}
		return ctx
	}

	if len(m.handlers) == 0 {
		output.Close()
		return ctx
	}

	outs := output.Copy(len(m.handlers))
	for i := 0; i < len(m.handlers); i++ {
		handler := m.handlers[i]
		ctx = handler.OnEndWithStreamOutput(ctx, m.runInfo, outs[i])
	}

	return ctx
}
