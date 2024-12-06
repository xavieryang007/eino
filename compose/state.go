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

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/generic"
)

// NewStateGraph creates a new state graph. It requires a func of GenLocalState to generate the state.
// eg.
//
//	type testState struct {
//		UserInfo *UserInfo
//		KVs map[string]any
//	}
//
//	genStateFunc := func(ctx context.Context) *testState {
//		return &testState{}
//	}
//
//	graph := compose.NewStateGraph[string, string, testState](genStateFunc)
//
//	// you can use WithPreHandler and WithPostHandler to do something with state of this graph.
//	graph.AddNode("node1", someNode, compose.WithPreHandler(func(ctx context.Context, in string, state *testState) (string, error) {
//		// do something with state
//		return in, nil
//	}), compose.WithPostHandler(func(ctx context.Context, out string, state *testState) (string, error) {
//		// do something with state
//		return out, nil
//	}))
func NewStateGraph[I, O, S any](gen GenLocalState[S]) *StateGraph[I, O, S] {
	sg := &StateGraph[I, O, S]{NewGraph[I, O]()}

	sg.graph.runtimeGraphKey = defaultGraphKey()
	sg.runCtx = func(ctx context.Context) context.Context {
		state := gen(ctx)
		return context.WithValue(ctx, stateKey{}, state)
	}

	sg.addNodeChecker = nodeCheckerOfForbidNodeKey(baseNodeChecker)

	sg.compileChecker = func(options *graphCompileOptions) error {
		return nil
	}

	return sg
}

// StateGraph is a graph that shares state between nodes. It's useful when you want to share some data across nodes.
type StateGraph[I, O, S any] struct {
	*Graph[I, O]
}

func (s *StateGraph[I, O, S]) component() component {
	return ComponentOfStateGraph
}

// Compile the graph to runnable.
func (s *StateGraph[I, O, S]) Compile(ctx context.Context, opts ...GraphCompileOption) (Runnable[I, O], error) {
	opts = append(opts, withComponent(s.component()))

	return s.Graph.Compile(ctx, opts...)
}

// NewStateChain creates a new state chain. It requires a func of GenLocalState to generate the state.
// eg.
//
//	genStateFunc := func(ctx context.Context) *testState {
//		// or may be you can create the state by params in ctx.
//		return &testState{}
//	}
//
//	chain := compose.NewStateChain[string, string, testState](genStateFunc)
//
//	chain.AppendXXX(someNode, compose.WithPreHandler(func(ctx context.Context, in string, state *testState) (string, error) {
//		// do something with state
//		return in, nil
//	}), compose.WithPostHandler(func(ctx context.Context, out string, state *testState) (string, error) {
//		// do something with state
//		return out, nil
//	}))
func NewStateChain[I, O, S any](gen GenLocalState[S]) *StateChain[I, O, S] {
	sc := &StateChain[I, O, S]{NewChain[I, O]()}

	sc.gg.runCtx = func(ctx context.Context) context.Context {
		state := gen(ctx)
		return context.WithValue(ctx, stateKey{}, state)
	}

	sc.gg.addNodeChecker = baseNodeChecker

	sc.gg.compileChecker = func(options *graphCompileOptions) error {
		return nil
	}

	return sc
}

// StateChain is a chain that shares state between nodes. State is shared between nodes in the chain.
// It's useful when you want to share some data across nodes in a chain.
// you can use WithPreHandler and WithPostHandler to do something with state of this chain.
type StateChain[I, O, S any] struct {
	*Chain[I, O]
}

func (s *StateChain[I, O, S]) component() component {
	return ComponentOfStateChain
}

// Compile the chain to runnable.
func (s *StateChain[I, O, S]) Compile(ctx context.Context, opts ...GraphCompileOption) (Runnable[I, O], error) {
	opts = append(opts, withComponent(s.component()))

	return s.Chain.Compile(ctx, opts...)
}

// GenLocalState is a function that generates the state.
type GenLocalState[S any] func(ctx context.Context) (state S)

type stateKey struct{}

// StatePreHandler is a function that is called before the node is executed.
// Notice: if user called Stream but with StatePreHandler, the StatePreHandler will read all stream chunks and merge them into a single object.
type StatePreHandler[I, S any] func(ctx context.Context, in I, state S) (I, error)

// StatePostHandler is a function that is called after the node is executed.
// Notice: if user called Stream but with StatePostHandler, the StatePostHandler will read all stream chunks and merge them into a single object.
type StatePostHandler[O, S any] func(ctx context.Context, out O, state S) (O, error)

// StreamStatePreHandler is a function that is called before the node is executed with stream input and output.
type StreamStatePreHandler[I, S any] func(ctx context.Context, in *schema.StreamReader[I], state S) (*schema.StreamReader[I], error)

// StreamStatePostHandler is a function that is called after the node is executed with stream input and output.
type StreamStatePostHandler[O, S any] func(ctx context.Context, out *schema.StreamReader[O], state S) (*schema.StreamReader[O], error)

func convertPreHandler[I, S any](handler StatePreHandler[I, S]) *composableRunnable {
	rf := func(ctx context.Context, in I, opts ...any) (I, error) {
		cState, err := GetState[S](ctx)
		if err != nil {
			return in, err
		}

		return handler(ctx, in, cState)
	}

	return runnableLambda[I, I](rf, nil, nil, nil, false)
}

func convertPostHandler[O, S any](handler StatePostHandler[O, S]) *composableRunnable {
	rf := func(ctx context.Context, out O, opts ...any) (O, error) {
		cState, err := GetState[S](ctx)
		if err != nil {
			return out, err
		}

		return handler(ctx, out, cState)
	}

	return runnableLambda[O, O](rf, nil, nil, nil, false)
}

func streamConvertPreHandler[I, S any](handler StreamStatePreHandler[I, S]) *composableRunnable {
	rf := func(ctx context.Context, in *schema.StreamReader[I], opts ...any) (*schema.StreamReader[I], error) {
		cState, err := GetState[S](ctx)
		if err != nil {
			return in, err
		}

		return handler(ctx, in, cState)
	}

	return runnableLambda[I, I](nil, nil, nil, rf, false)
}

func streamConvertPostHandler[O, S any](handler StreamStatePostHandler[O, S]) *composableRunnable {
	rf := func(ctx context.Context, out *schema.StreamReader[O], opts ...any) (*schema.StreamReader[O], error) {
		cState, err := GetState[S](ctx)
		if err != nil {
			return out, err
		}

		return handler(ctx, out, cState)
	}

	return runnableLambda[O, O](nil, nil, nil, rf, false)
}

// GetState gets the state from the context.
// When using this method to read or write state in custom nodes, it may lead to data race because other nodes may concurrently access the state.
// You need to be aware of and resolve this situation, typically by adding a mutex.
// It's recommended to only READ the returned state. If you want to WRITE to state, consider using StatePreHandler / StatePostHandler because they are concurrency safe out of the box.
// eg.
//
//	lambdaFunc := func(ctx context.Context, in string, opts ...any) (string, error) {
//		state, err := compose.GetState[*testState](ctx)
//		if err != nil {
//			return "", err
//		}
//		// do something with state
//		return in, nil
//	}
//
//	stateGraph := compose.NewStateGraph[string, string, testState](genStateFunc)
//	stateGraph.AddNode("node1", lambdaFunc)
func GetState[S any](ctx context.Context) (S, error) {
	state := ctx.Value(stateKey{})

	cState, ok := state.(S)
	if !ok {
		var s S
		return s, fmt.Errorf("unexpected state type. expected: %v, got: %v",
			generic.TypeOf[S](), reflect.TypeOf(state))
	}

	return cState, nil
}
