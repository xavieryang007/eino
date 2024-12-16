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

package compose

import (
	"context"
	"fmt"
	"reflect"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

func mergeMap(vs []any) (any, error) {
	typ := reflect.TypeOf(vs[0])
	merged := reflect.MakeMap(typ)
	for _, v := range vs {
		if reflect.TypeOf(v) != typ {
			return nil, fmt.Errorf(
				"(mergeMap) field type mismatch. expected: '%v', got: '%v'", typ, reflect.TypeOf(v))
		}

		iter := reflect.ValueOf(v).MapRange()
		for iter.Next() {
			key, val := iter.Key(), iter.Value()
			if merged.MapIndex(key).IsValid() {
				return nil, fmt.Errorf("(mergeMap) duplicated key ('%v') found", key.Interface())
			}
			merged.SetMapIndex(key, val)
		}
	}

	return merged.Interface(), nil
}

// the caller should ensure len(vs) > 1
func mergeValues(vs []any) (any, error) {
	v0 := reflect.ValueOf(vs[0])
	t0 := v0.Type()
	k0 := t0.Kind()

	if k0 == reflect.Map {
		return mergeMap(vs)
	}

	if s, ok := vs[0].(streamReader); ok {
		if s.getChunkType().Kind() != reflect.Map {
			return nil, fmt.Errorf("(mergeValues | stream type)"+
				" unsupported chunk type: %v", s.getChunkType())
		}

		ss := make([]streamReader, len(vs)-1)
		for i := 0; i < len(ss); i++ {
			s_, ok_ := vs[i+1].(streamReader)
			if !ok_ {
				return nil, fmt.Errorf("(mergeStream) unexpected type. "+
					"expect: %v, got: %v", t0, reflect.TypeOf(vs[i]))
			}

			if s_.getChunkType() != s.getChunkType() {
				return nil, fmt.Errorf("(mergeStream) chunk type mismatch. "+
					"expect: %v, got: %v", s.getChunkType(), s_.getChunkType())
			}

			ss[i] = s_
		}

		ms := s.merge(ss)

		return ms, nil
	}

	return nil, fmt.Errorf("(mergeValues) unsupported type: %v", t0)
}

func invokeWithCallbacks[I, O, TOption any](i Invoke[I, O, TOption]) Invoke[I, O, TOption] {
	return func(ctx context.Context, input I, opts ...TOption) (output O, err error) {
		if !callbacks.Needed(ctx) {
			return i(ctx, input, opts...)
		}

		defer func() {
			if err != nil {
				_ = callbacks.OnError(ctx, err)
				return
			}

			_ = callbacks.OnEnd(ctx, output)

		}()

		ctx = callbacks.OnStart(ctx, input)

		return i(ctx, input, opts...)
	}
}

func genericInvokeWithCallbacks(i invoke) invoke {
	return func(ctx context.Context, input any, opts ...any) (output any, err error) {
		if !callbacks.Needed(ctx) {
			return i(ctx, input, opts...)
		}

		defer func() {
			if err != nil {
				_ = callbacks.OnError(ctx, err)
				return
			}

			_ = callbacks.OnEnd(ctx, output)

		}()

		ctx = callbacks.OnStart(ctx, input)

		return i(ctx, input, opts...)
	}
}

func streamWithCallbacks[I, O, TOption any](s Stream[I, O, TOption]) Stream[I, O, TOption] {
	return func(ctx context.Context, input I, opts ...TOption) (output *schema.StreamReader[O], err error) {
		if !callbacks.Needed(ctx) {
			return s(ctx, input, opts...)
		}

		ctx = callbacks.OnStart(ctx, input)

		output, err = s(ctx, input, opts...)
		if err != nil {
			_ = callbacks.OnError(ctx, err)
			return output, err
		}

		_, newS := callbacks.OnEndWithStreamOutput(ctx, output)

		return newS, nil
	}
}

func collectWithCallbacks[I, O, TOption any](c Collect[I, O, TOption]) Collect[I, O, TOption] {
	return func(ctx context.Context, input *schema.StreamReader[I], opts ...TOption) (output O, err error) {
		if !callbacks.Needed(ctx) {
			return c(ctx, input, opts...)
		}

		defer func() {
			if err != nil {
				_ = callbacks.OnError(ctx, err)
				return
			}
			_ = callbacks.OnEnd(ctx, output)
		}()

		ctx, newS := callbacks.OnStartWithStreamInput(ctx, input)

		return c(ctx, newS, opts...)
	}
}

func transformWithCallbacks[I, O, TOption any](t Transform[I, O, TOption]) Transform[I, O, TOption] {
	return func(ctx context.Context, input *schema.StreamReader[I],
		opts ...TOption) (output *schema.StreamReader[O], err error) {

		if !callbacks.Needed(ctx) {
			return t(ctx, input, opts...)
		}

		ctx, input = callbacks.OnStartWithStreamInput(ctx, input)

		output, err = t(ctx, input, opts...)
		if err != nil {
			_ = callbacks.OnError(ctx, err)
			return output, err
		}

		_, output = callbacks.OnEndWithStreamOutput(ctx, output)

		return output, nil
	}
}

func genericTransformWithCallbacks(t transform) transform {
	return func(ctx context.Context, input streamReader, opts ...any) (output streamReader, err error) {
		if !callbacks.Needed(ctx) {
			return t(ctx, input, opts...)
		}

		inArr := input.copy(2)
		is, ok := unpackStreamReader[callbacks.CallbackInput](inArr[1])
		if !ok { // unexpected
			return t(ctx, inArr[0], opts...)
		}

		ctx, is = callbacks.OnStartWithStreamInput(ctx, is)
		is.Close() // goroutine free copy buffer release

		output, err = t(ctx, inArr[0], opts...)
		if err != nil {
			_ = callbacks.OnError(ctx, err)
			return output, err
		}

		outArr := output.copy(2)
		os, ok := unpackStreamReader[callbacks.CallbackOutput](outArr[1])
		if !ok { // unexpected
			return outArr[0], nil
		}

		_, os = callbacks.OnEndWithStreamOutput(ctx, os)
		os.Close()

		return outArr[0], nil
	}
}

func initGraphCallbacks(ctx context.Context, info *nodeInfo, meta *executorMeta, opts ...Option) context.Context {
	ri := &callbacks.RunInfo{}
	if meta != nil {
		ri.Component = meta.component
		ri.Type = meta.componentImplType
	}

	if info != nil {
		ri.Name = info.name
	}

	var cbs []callbacks.Handler
	for i := range opts {
		if len(opts[i].graphHandler) != 0 {
			cbs = append(cbs, opts[i].graphHandler...)
		}

		if len(opts[i].handler) != 0 && len(opts[i].keys) == 0 {
			cbs = append(cbs, opts[i].handler...)
		}
	}

	return callbacks.InitCallbacks(ctx, ri, cbs...)
}

func streamChunkConvertForCBOutput[O any](o O) (callbacks.CallbackOutput, error) {
	return o, nil
}

func streamChunkConvertForCBInput[I any](i I) (callbacks.CallbackInput, error) {
	return i, nil
}

func toAnyList[T any](in []T) []any {
	ret := make([]any, len(in))
	for i := range in {
		ret[i] = in[i]
	}
	return ret
}

type assignableType uint8

const (
	assignableTypeMustNot assignableType = iota
	assignableTypeMust
	assignableTypeMay
)

func checkAssignable(input, arg reflect.Type) assignableType {
	if arg == nil || input == nil {
		return assignableTypeMustNot
	}

	if arg == input {
		return assignableTypeMust
	}

	if arg.Kind() == reflect.Interface && input.Implements(arg) {
		return assignableTypeMust
	}
	if input.Kind() == reflect.Interface {
		if arg.Implements(input) {
			return assignableTypeMay
		}
		return assignableTypeMustNot
	}

	return assignableTypeMustNot
}

func extractOption(nodes map[string]*chanCall, opts ...Option) (map[string][]any, error) {
	optMap := map[string][]any{}
	for _, opt := range opts {
		if len(opt.options) == 0 {
			continue
		}
		if len(opt.keys) == 0 {
			// common option, check type
			for name, c := range nodes {
				if reflect.TypeOf(opt.options[0]) == c.action.optionType { // assume that types of options are the same
					optMap[name] = append(optMap[name], opt.options...)
				}
			}
		}
		for _, key := range opt.keys {
			if _, ok := nodes[key]; !ok {
				return nil, fmt.Errorf("option has designated an unknown node: %s", key)
			}
			if nodes[key].action.optionType != reflect.TypeOf(opt.options[0]) { // assume that types of options are the same
				return nil, fmt.Errorf("option type[%s] is different from which the designated node[%s] expects[%s]",
					reflect.TypeOf(opt.options[0]).String(), key, nodes[key].action.optionType.String())
			}
			optMap[key] = append(optMap[key], opt.options...)
		}
	}
	for k, v := range nodes {
		if v.action.optionType == nil {
			// sub graph
			optMap[k] = toAnyList(opts)
		}
	}
	return optMap, nil
}

func mapToList(m map[string]any) []any {
	ret := make([]any, 0, len(m))
	for _, v := range m {
		ret = append(ret, v)
	}
	return ret
}
