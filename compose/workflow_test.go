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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/internal/mock/components/document"
	"github.com/cloudwego/eino/internal/mock/components/embedding"
	"github.com/cloudwego/eino/internal/mock/components/indexer"
	"github.com/cloudwego/eino/internal/mock/components/model"
	"github.com/cloudwego/eino/internal/mock/components/retriever"
	"github.com/cloudwego/eino/schema"
)

func TestWorkflow(t *testing.T) {
	ctx := context.Background()

	type structA struct {
		Field1 string
		Field2 int
		Field3 []any
	}

	type structB struct {
		Field1 string
		Field2 int
	}

	type structE struct {
		Field1 string
		Field2 string
		Field3 []any
	}

	type state struct {
		temp string
	}

	type structEnd struct {
		Field1 string
	}

	subGraph := NewGraph[string, *structB]()
	_ = subGraph.AddLambdaNode(
		"1",
		InvokableLambda(func(ctx context.Context, input string) (*structB, error) {
			return &structB{Field1: input, Field2: 33}, nil
		}),
	)
	_ = subGraph.AddEdge(START, "1")
	_ = subGraph.AddEdge("1", END)

	subChain := NewChain[any, any]().
		AppendLambda(InvokableLambda(func(_ context.Context, in any) (any, error) {
			return map[string]string{"key2": fmt.Sprintf("%d", in)}, nil
		}))

	subWorkflow := NewWorkflow[[]any, []any]()
	subWorkflow.AddLambdaNode(
		"1",
		InvokableLambda(func(_ context.Context, in []any) ([]any, error) {
			return in, nil
		}),
		WithOutputKey("key")).
		AddInput(NewMapping(START))
	subWorkflow.AddLambdaNode(
		"2",
		InvokableLambda(func(_ context.Context, in []any) ([]any, error) {
			return in, nil
		}),
		WithInputKey("key")).
		AddInput(NewMapping("1"))
	subWorkflow.AddEnd(NewMapping("2"))

	w := NewWorkflow[*structA, *structEnd](WithGenLocalState(func(context.Context) *state { return &state{} }))

	w.
		AddGraphNode("B", subGraph,
			WithStatePostHandler(func(ctx context.Context, out *structB, state *state) (*structB, error) {
				state.temp = out.Field1
				return out, nil
			})).
		AddInput(NewMapping(START).FromField("Field1"))

	w.
		AddGraphNode("C", subChain).
		AddInput(NewMapping(START).FromField("Field2"))

	w.
		AddGraphNode("D", subWorkflow).
		AddInput(NewMapping(START).FromField("Field3"))

	w.
		AddLambdaNode(
			"E",
			TransformableLambda(func(_ context.Context, in *schema.StreamReader[structE]) (*schema.StreamReader[structE], error) {
				return schema.StreamReaderWithConvert(in, func(in structE) (structE, error) {
					if len(in.Field1) > 0 {
						in.Field1 = "E:" + in.Field1
					}
					if len(in.Field2) > 0 {
						in.Field2 = "E:" + in.Field2
					}

					return in, nil
				}), nil
			}),
			WithStreamStatePreHandler(func(ctx context.Context, in *schema.StreamReader[structE], state *state) (*schema.StreamReader[structE], error) {
				temp := state.temp
				return schema.StreamReaderWithConvert(in, func(v structE) (structE, error) {
					if len(v.Field3) > 0 {
						v.Field3 = append(v.Field3, "Pre:"+temp)
					}

					return v, nil
				}), nil
			}),
			WithStreamStatePostHandler(func(ctx context.Context, out *schema.StreamReader[structE], state *state) (*schema.StreamReader[structE], error) {
				return schema.StreamReaderWithConvert(out, func(v structE) (structE, error) {
					if len(v.Field1) > 0 {
						v.Field1 = v.Field1 + "+Post"
					}
					return v, nil
				}), nil
			})).
		AddInput(
			NewMapping("B").FromField("Field1").ToField("Field1"),
			NewMapping("C").FromMapKey("key2").ToField("Field2"),
			NewMapping("D").ToField("Field3"),
		)

	w.
		AddLambdaNode(
			"F",
			InvokableLambda(func(ctx context.Context, in map[string]any) (string, error) {
				return fmt.Sprintf("%v_%v_%v_%v_%v", in["key1"], in["key2"], in["key3"], in["B"], in["state_temp"]), nil
			}),
			WithStatePreHandler(func(ctx context.Context, in map[string]any, state *state) (map[string]any, error) {
				in["state_temp"] = state.temp
				return in, nil
			}),
		).
		AddInput(
			NewMapping("B").FromField("Field2").ToMapKey("B"),
			NewMapping("E").FromField("Field1").ToMapKey("key1"),
			NewMapping("E").FromField("Field2").ToMapKey("key2"),
			NewMapping("E").FromField("Field3").ToMapKey("key3"),
		)

	w.AddEnd(NewMapping("F").ToField("Field1"))

	compiled, err := w.Compile(ctx)
	assert.NoError(t, err)

	input := &structA{
		Field1: "1",
		Field2: 2,
		Field3: []any{
			1, "good",
		},
	}
	out, err := compiled.Invoke(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, &structEnd{"E:1+Post_E:2_[1 good Pre:1]_33_1"}, out)

	outStream, err := compiled.Stream(ctx, input)
	assert.NoError(t, err)
	defer outStream.Close()
	for {
		chunk, err := outStream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}

			t.Error(err)
			return
		}

		assert.Equal(t, &structEnd{"E:1+Post_E:2_[1 good Pre:1]_33_1"}, chunk)
	}
}

func TestWorkflowCompile(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	t.Run("a node has no input", func(t *testing.T) {
		w := NewWorkflow[string, string]()
		w.AddToolsNode("1", &ToolsNode{})
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "workflow node = 1 has no input")
	})

	t.Run("compile without add end", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(NewMapping(START))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "workflow END has no input mapping")
	})

	t.Run("type mismatch", func(t *testing.T) {
		w := NewWorkflow[string, string]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(NewMapping(START))
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "mismatch")
	})

	t.Run("upstream not map, mapping has FromMapKey", func(t *testing.T) {
		w := NewWorkflow[map[string]any, string]()

		w.AddChatTemplateNode("prompt", prompt.FromMessages(schema.Jinja2, schema.SystemMessage("your name is {{ name }}"))).AddInput(NewMapping(START))

		w.AddEnd(NewMapping("prompt").FromMapKey("key1"))

		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "type[[]*schema.Message] is not a map")
	})

	t.Run("upstream not struct/struct ptr, mapping has FromField", func(t *testing.T) {
		w := NewWorkflow[[]*schema.Document, []string]()

		w.AddIndexerNode("indexer", indexer.NewMockIndexer(ctrl)).AddInput(NewMapping(START).FromField("F1"))
		w.AddEnd(NewMapping("indexer"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "type[[]*schema.Document] is not a struct")
	})

	t.Run("downstream not map, mapping has ToMapKey", func(t *testing.T) {
		w := NewWorkflow[string, []*schema.Document]()

		w.AddRetrieverNode("retriever", retriever.NewMockRetriever(ctrl)).AddInput(NewMapping(START).ToMapKey("key1"))
		w.AddEnd(NewMapping("retriever"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "type[string] is not a map")
	})

	t.Run("downstream not struct/struct ptr, mapping has ToField", func(t *testing.T) {
		w := NewWorkflow[[]string, [][]float64]()
		w.AddEmbeddingNode("embedder", embedding.NewMockEmbedder(ctrl)).AddInput(NewMapping(START).ToField("F1"))
		w.AddEnd(NewMapping("embedder"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "type[[]string] is not a struct")
	})

	t.Run("upstream not map[string]T, mapping has FromField", func(t *testing.T) {
		w := NewWorkflow[map[int]int, any]()
		w.AddLoaderNode("loader", document.NewMockLoader(ctrl)).AddInput(NewMapping(START).FromMapKey("key1"))
		w.AddEnd(NewMapping("loader"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "type[map[int]int] is not a map with string key")
	})

	t.Run("downstream not map[string]T, mapping has ToField", func(t *testing.T) {
		w := NewWorkflow[any, map[int]int]()
		w.AddDocumentTransformerNode("transformer", document.NewMockTransformer(ctrl)).AddInput(NewMapping(START))
		w.AddEnd(NewMapping("transformer").ToMapKey("key1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "type[map[int]int] is not a map with string key")
	})

	t.Run("map to non existing field in upstream", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("tools_node", &ToolsNode{}).AddInput(NewMapping(START).FromField("non_exist"))
		w.AddEnd(NewMapping("tools_node"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "type[schema.Message] has no field[non_exist]")
	})

	t.Run("map to not exported field in downstream", func(t *testing.T) {
		w := NewWorkflow[string, *Mapping]()
		w.AddLambdaNode("1", InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
			return input, nil
		})).AddInput(NewMapping(START))
		w.AddEnd(NewMapping("1").ToField("toField"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "type[compose.Mapping] has an unexported field[toField]")
	})

	t.Run("duplicate node key", func(t *testing.T) {
		w := NewWorkflow[[]*schema.Message, []*schema.Message]()
		w.AddChatModelNode("1", model.NewMockChatModel(ctrl)).AddInput(NewMapping(START))
		w.AddToolsNode("1", &ToolsNode{}).AddInput(NewMapping("1"))
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "node '1' already present")
	})

	t.Run("from non-existing node", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(NewMapping(START))
		w.AddEnd(NewMapping("2"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "edge start node '2' needs to be added to graph first")
	})

	t.Run("multiple mappings have an empty mapping", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(NewMapping(START), NewMapping(START).FromField("Content").ToField("Content"))
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have an empty mapping")
	})

	t.Run("multiple mappings have mapping to entire output ", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromField("Role"),
			NewMapping(START).FromField("Content"),
		)
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have a mapping to entire output")
	})

	t.Run("multiple mappings have mapping from entire input ", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).ToField("Role"),
			NewMapping(START).ToField("Content"),
		)
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have a mapping from entire input")
	})

	t.Run("multiple mappings set both FromMapKey and FromField", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromMapKey("Role").ToField("Role"),
			NewMapping(START).FromField("Content").ToField("Content"),
		)
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have both FromField and FromMapKey")

		w = NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromField("Content").ToField("Content"),
			NewMapping(START).FromMapKey("Role").ToField("Role"),
		)
		w.AddEnd(NewMapping("1"))
		_, err = w.Compile(ctx)
		assert.ErrorContains(t, err, "have both FromField and FromMapKey")
	})

	t.Run("multiple mappings have duplicate FromMapKey", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromMapKey("Role").ToField("Role"),
			NewMapping(START).FromMapKey("Role").ToField("Role"),
		)
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have the same FromMapKey")
	})

	t.Run("multiple mappings have duplicate FromField", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromField("Content").ToField("Content"),
			NewMapping(START).FromField("Content").ToField("Content"),
		)
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have the same FromField")
	})

	t.Run("multiple mappings set both ToMapKey and ToField", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromField("Role").ToField("Role"),
			NewMapping(START).FromField("Content").ToMapKey("Content"),
		)
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have both ToField and ToMapKey")

		w = NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromField("Content").ToMapKey("Content"),
			NewMapping(START).FromField("Role").ToField("Role"),
		)
		w.AddEnd(NewMapping("1"))
		_, err = w.Compile(ctx)
		assert.ErrorContains(t, err, "have both ToField and ToMapKey")
	})

	t.Run("multiple mappings have duplicate ToMapKey", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromMapKey("Role").ToMapKey("Role"),
			NewMapping(START).FromMapKey("Content").ToMapKey("Role"),
		)
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have the same ToMapKey")
	})

	t.Run("multiple mappings have duplicate ToField", func(t *testing.T) {
		w := NewWorkflow[*schema.Message, []*schema.Message]()
		w.AddToolsNode("1", &ToolsNode{}).AddInput(
			NewMapping(START).FromField("Content").ToField("Content"),
			NewMapping(START).FromField("Role").ToField("Content"),
		)
		w.AddEnd(NewMapping("1"))
		_, err := w.Compile(ctx)
		assert.ErrorContains(t, err, "have the same ToField")
	})
}
