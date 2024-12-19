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

import "context"

type manager struct {
	handlers []Handler
	runInfo  *RunInfo
}

var GlobalHandlers []Handler

func newManager(runInfo *RunInfo, handlers ...Handler) (*manager, bool) {
	l := len(handlers) + len(GlobalHandlers)
	if l == 0 {
		return nil, false
	}
	hs := make([]Handler, 0, l)
	hs = append(hs, GlobalHandlers...)
	hs = append(hs, handlers...)

	return &manager{
		handlers: hs,
		runInfo:  runInfo,
	}, true
}

func ctxWithManager(ctx context.Context, manager *manager) context.Context {
	return context.WithValue(ctx, CtxManagerKey{}, manager)
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

func managerFromCtx(ctx context.Context) (*manager, bool) {
	v := ctx.Value(CtxManagerKey{})
	m, ok := v.(*manager)
	if ok && m != nil {
		return &manager{
			handlers: m.handlers,
			runInfo:  m.runInfo,
		}, true
	}

	return nil, false
}
