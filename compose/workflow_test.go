package compose

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflow(t *testing.T) {
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

	w := NewWorkflow[*structA, string]()

	w.
		AddLambdaNode(
			"B",
			InvokableLambda(func(context.Context, string) (*structB, error) {
				return &structB{Field1: "1", Field2: 2}, nil
			}),
			WithWorkflowNodeName("node B")).
		AddInput(&Mapping{
			From:      START,
			FromField: "Field1",
		})

	w.
		AddLambdaNode(
			"C",
			InvokableLambda(func(_ context.Context, in int) (map[string]string, error) {
				return map[string]string{"key2": fmt.Sprintf("%d", in)}, nil
			})).
		AddInput(&Mapping{
			From:      START,
			FromField: "Field2",
		})

	w.
		AddLambdaNode(
			"D",
			InvokableLambda(func(_ context.Context, in []any) ([]any, error) {
				return in, nil
			})).
		AddInput(&Mapping{
			From:      START,
			FromField: "Field3",
		})

	w.
		AddLambdaNode(
			"E",
			InvokableLambda(func(_ context.Context, in *structE) (string, error) {
				return in.Field1 + "_" + in.Field2 + "_" + fmt.Sprintf("%v", in.Field3[0]), nil
			})).
		AddInput(&Mapping{
			From:      "B",
			FromField: "Field1",
			ToField:   "Field1",
		}, &Mapping{
			From:       "C",
			FromMapKey: "key2",
			ToField:    "Field2",
		}, &Mapping{
			From:    "D",
			ToField: "Field3",
		})

	w.AddEnd([]*Mapping{{
		From: "E",
	}})

	ctx := context.Background()
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
	assert.Equal(t, "1_2_1", out)

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

		assert.Equal(t, "1_2_1", chunk)
	}
}
