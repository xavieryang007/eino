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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/retriever"
	mockDocument "github.com/cloudwego/eino/internal/mock/components/document"
	mockEmbedding "github.com/cloudwego/eino/internal/mock/components/embedding"
	mockRetriever "github.com/cloudwego/eino/internal/mock/components/retriever"
	"github.com/cloudwego/eino/schema"
)

var optionSuccess = true
var idx int

func checkOption(opts ...model.Option) bool {
	if len(opts) != 2 {
		return false
	}
	o := model.GetCommonOptions(&model.Options{}, opts...)
	if o.TopP == nil || *o.TopP != 1.0 {
		return false
	}
	if o.Model == nil {
		return false
	}
	if idx == 0 {
		idx = 1
		if o.Model == nil || *o.Model != "123" {
			return false
		}
	} else {
		idx = 0
		if o.Model == nil || *o.Model != "456" {
			return false
		}
	}

	return true
}

type testModel struct{}

func (t *testModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (t *testModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if !checkOption(opts...) {
		optionSuccess = false
	}
	return &schema.Message{}, nil
}

func (t *testModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if !checkOption(opts...) {
		optionSuccess = false
	}
	sr, sw := schema.Pipe[*schema.Message](1)
	sw.Send(nil, nil)
	sw.Close()
	return sr, nil
}

func TestCallOption(t *testing.T) {
	g := NewGraph[[]*schema.Message, *schema.Message]()
	err := g.AddLambdaNode("1", InvokableLambdaWithOption(func(ctx context.Context, input []*schema.Message, opts ...string) (output []*schema.Message, err error) {
		if len(opts) != 1 || opts[0] != "1" {
			t.Fatalf("lambda option length isn't 1 or content isn't '1': %v", opts)
		}
		return input, nil
	}))
	assert.Nil(t, err)

	err = g.AddChatModelNode("2", &testModel{})
	assert.Nil(t, err)

	err = g.AddLambdaNode("-", InvokableLambda(func(ctx context.Context, input *schema.Message) (output []*schema.Message, err error) {
		return []*schema.Message{input}, nil
	}))
	assert.Nil(t, err)

	err = g.AddChatModelNode("3", &testModel{})
	if err != nil {
		t.Fatal(err)
	}
	err = g.AddEdge(START, "1")
	if err != nil {
		t.Fatal(err)
	}
	err = g.AddEdge("1", "2")
	assert.Nil(t, err)

	err = g.AddEdge("2", "-")
	assert.Nil(t, err)

	err = g.AddEdge("-", "3")
	assert.Nil(t, err)

	err = g.AddEdge("3", END)
	assert.Nil(t, err)

	ctx := context.Background()

	r, err := g.Compile(ctx)
	assert.Nil(t, err)

	sessionKey := struct{}{}
	startCnt := 0
	endCnt := 0
	opts := []Option{
		WithChatModelOption(
			model.WithModel("123"),
		).DesignateNode("2"),
		WithChatModelOption(
			model.WithModel("456"),
		).DesignateNode("3"),
		WithChatModelOption(
			model.WithTopP(1.0),
		),
		WithCallbacks(callbacks.NewHandlerBuilder().
			OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
				startCnt++
				return context.WithValue(ctx, sessionKey, "start")
			}).
			OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
				if ctx.Value(sessionKey).(string) == "start" {
					endCnt++
					return context.WithValue(ctx, sessionKey, "end")
				}
				return ctx
			}).Build()).DesignateNode("3"),
		WithLambdaOption("1").DesignateNode("1"),
	}

	_, err = r.Invoke(ctx, []*schema.Message{},
		opts...)
	if err != nil {
		t.Fatal(err)
	}
	if !optionSuccess {
		t.Fatal("invoke option fail")
	}
	if startCnt != 1 {
		t.Fatal("node callback fail")
	}
	if endCnt != 1 {
		t.Fatal("node callback fail")
	}
	_, err = r.Stream(ctx, []*schema.Message{},
		opts...)
	if err != nil {
		t.Fatal(err)
	}
	if !optionSuccess {
		t.Fatal("stream option fail")
	}

	srOfCollect, swOfCollect := schema.Pipe[[]*schema.Message](1)
	swOfCollect.Send([]*schema.Message{}, nil)
	swOfCollect.Close()
	_, err = r.Collect(ctx, srOfCollect, opts...)
	assert.Nil(t, err)

	if !optionSuccess {
		t.Fatal("collect option fail")
	}

	srOfTransform, swOfTransform := schema.Pipe[[]*schema.Message](1)
	swOfTransform.Send([]*schema.Message{}, nil)
	swOfTransform.Close()
	_, err = r.Transform(ctx, srOfTransform, opts...)
	assert.Nil(t, err)

	if !optionSuccess {
		t.Fatal("transform option fail")
	}
}

func TestCallOptionsOneByOne(t *testing.T) {
	ctx := context.Background()
	t.Run("common_option", func(t *testing.T) {
		type option struct {
			uid int64
		}

		opt := withComponentOption(&option{uid: 100})
		assert.Len(t, opt.options, 1)
		assert.IsType(t, &option{}, opt.options[0])
		assert.Equal(t, &option{uid: 100}, opt.options[0])
	})

	t.Run("embedding_option", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		inst := mockEmbedding.NewMockEmbedder(ctrl)
		var opt *embedding.Options
		inst.EXPECT().EmbedStrings(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
				opt = embedding.GetCommonOptions(&embedding.Options{}, opts...)
				return nil, nil
			}).Times(1)
		ch := NewChain[map[string]any, map[string]any]()
		ch.AppendEmbedding(inst, WithInputKey("input"), WithOutputKey("output"))
		r, err := ch.Compile(ctx)
		assert.NoError(t, err)
		outs, err := r.Invoke(ctx,
			map[string]any{"input": []string{}},
			WithEmbeddingOption(embedding.WithModel("123")),
		)
		assert.NoError(t, err)
		assert.Contains(t, outs, "output")

		assert.NotNil(t, opt.Model)
		assert.Equal(t, "123", *opt.Model)
	})

	t.Run("retriever_option", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		inst := mockRetriever.NewMockRetriever(ctrl)
		var opt *retriever.Options
		inst.EXPECT().Retrieve(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
				opt = retriever.GetCommonOptions(&retriever.Options{}, opts...)
				return nil, nil
			}).
			Times(1)
		ch := NewChain[map[string]any, map[string]any]()
		ch.AppendRetriever(inst, WithInputKey("input"), WithOutputKey("output"))
		r, err := ch.Compile(ctx)
		assert.NoError(t, err)
		outs, err := r.Invoke(ctx,
			map[string]any{"input": "hi"},
			WithRetrieverOption(retriever.WithIndex("123")),
		)
		assert.NoError(t, err)
		assert.Contains(t, outs, "output")

		assert.NotNil(t, opt.Index)
		assert.Equal(t, "123", *opt.Index)
	})

	t.Run("loader_option", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		inst := mockDocument.NewMockLoader(ctrl)
		type implOption struct {
			uid int64
		}

		type implOptFn func(o *implOption)

		withUID := func(uid int64) document.LoaderOption {
			return document.WrapLoaderImplSpecificOptFn[implOption](func(i *implOption) {
				i.uid = uid
			})
		}

		var opt *implOption

		inst.EXPECT().Load(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, src document.Source, opts ...document.LoaderOption) ([]*schema.Document, error) {
				opt = document.GetLoaderImplSpecificOptions[implOption](&implOption{uid: 1}, opts...)
				return nil, nil
			}).
			Times(1)
		ch := NewChain[map[string]any, map[string]any]()
		ch.AppendLoader(inst, WithInputKey("input"), WithOutputKey("output"))
		r, err := ch.Compile(ctx)
		assert.NoError(t, err)
		outs, err := r.Invoke(ctx,
			map[string]any{"input": document.Source{}},
			WithLoaderOption(withUID(123)),
		)
		assert.NoError(t, err)
		assert.Contains(t, outs, "output")

		assert.Equal(t, int64(123), opt.uid)
	})
}
