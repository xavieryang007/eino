package compose

import (
	"context"
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
			InvokableLambda(func(context.Context, int) (map[string]string, error) {
				return map[string]string{"key2": "value2"}, nil
			})).
		AddInput(&Mapping{
			From:      START,
			FromField: "Field2",
		})

	w.
		AddLambdaNode(
			"D",
			InvokableLambda(func(context.Context, []any) ([]any, error) {
				return make([]any, 0), nil
			})).
		AddInput(&Mapping{
			From:      START,
			FromField: "Field3",
		})

	w.
		AddLambdaNode(
			"E",
			InvokableLambda(func(context.Context, *structE) (string, error) {
				return "", nil
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

	out, err := compiled.Invoke(ctx, &structA{
		Field1: "1",
		Field2: 2,
		Field3: []any{
			1, "good",
		},
	})
	assert.NoError(t, err)
	t.Logf(out)
}
