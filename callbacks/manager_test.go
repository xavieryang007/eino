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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudwego/eino/schema"
)

func TestManager(t *testing.T) {

	t.Run("usable_manager", func(t *testing.T) {
		defer func() {
			globalHandlers = nil
		}()

		var startCnt, endCnt, errCnt int
		var globalKey, sessionKey = "global", "session"

		globalHandlers = []Handler{
			NewHandlerBuilder().
				OnStartFn(func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
					startCnt++
					return context.WithValue(ctx, globalKey, "start")
				}).
				OnEndFn(func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
					if ctx.Value(globalKey).(string) == "start" {
						endCnt++
						return context.WithValue(ctx, globalKey, "end")
					}
					return ctx
				}).
				OnErrorFn(func(ctx context.Context, info *RunInfo, err error) context.Context {
					if ctx.Value(globalKey).(string) == "start" {
						errCnt++
						return context.WithValue(ctx, globalKey, "error")
					}
					return ctx
				}).Build(),
		}

		sessionHandler := NewHandlerBuilder().
			OnStartFn(func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
				startCnt++
				return context.WithValue(ctx, sessionKey, "start")
			}).
			OnEndFn(func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
				if ctx.Value(sessionKey).(string) == "start" {
					endCnt++
					return context.WithValue(ctx, sessionKey, "end")
				}
				return ctx
			}).
			OnErrorFn(func(ctx context.Context, info *RunInfo, err error) context.Context {
				if ctx.Value(sessionKey).(string) == "start" {
					errCnt++
					return context.WithValue(ctx, sessionKey, "error")
				}
				return ctx
			}).Build()

		manager, ok := newManager(&RunInfo{}, sessionHandler)
		assert.True(t, ok)
		c0 := context.Background()
		c1 := ctxWithManager(c0, manager)
		c2 := ctxWithManager(c1, nil)

		m0, ok := managerFromCtx(c0)
		assert.False(t, ok)
		assert.Nil(t, m0)

		m1, ok := managerFromCtx(c1)
		assert.True(t, ok)
		assert.NotNil(t, m1)
		m2, ok := managerFromCtx(c2)
		assert.False(t, ok)
		assert.Nil(t, m2)

		c3 := manager.OnStart(context.Background(), nil)
		c4 := manager.OnError(c3, fmt.Errorf("mock err"))
		c5 := manager.OnEnd(c3, nil)
		assert.Equal(t, startCnt, 2)
		assert.Equal(t, endCnt, 2)
		assert.Equal(t, errCnt, 2)
		assert.Equal(t, c3.Value(globalKey).(string), "start")
		assert.Equal(t, c3.Value(sessionKey).(string), "start")
		assert.Equal(t, c4.Value(globalKey).(string), "error")
		assert.Equal(t, c4.Value(sessionKey).(string), "error")
		assert.Equal(t, c5.Value(globalKey).(string), "end")
		assert.Equal(t, c5.Value(sessionKey).(string), "end")
	})

	t.Run("empty manager", func(t *testing.T) {
		ctx := context.Background()
		globalHandlers = nil
		mgr, ok := newManager(nil)
		assert.False(t, ok)
		assert.Nil(t, mgr)

		nCtx := mgr.OnStart(ctx, nil)
		assert.IsType(t, ctx, nCtx)

		ctx = mgr.OnEnd(ctx, nil)
		assert.IsType(t, ctx, nCtx)

		ctx = mgr.OnError(ctx, fmt.Errorf("mock err"))
		assert.IsType(t, ctx, nCtx)

		sri, _ := schema.Pipe[CallbackInput](1)
		ctx = mgr.OnStartWithStreamInput(ctx, sri)
		assert.IsType(t, ctx, nCtx)

		sro, _ := schema.Pipe[CallbackOutput](1)
		ctx = mgr.OnEndWithStreamOutput(ctx, sro)
		assert.IsType(t, ctx, nCtx)

		ctx = mgr.OnStartWithStreamInput(ctx, nil)
		assert.IsType(t, ctx, nCtx)

		ctx = mgr.OnEndWithStreamOutput(ctx, nil)
		assert.IsType(t, ctx, nCtx)
	})
}
