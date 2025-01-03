package compose

import (
	"context"
	"testing"
)

func TestWorkflow(t *testing.T) {
	type structA struct {
		field1 string
		field2 int
		field3 []any
	}

	type structB struct {
		field1 string
		field2 int
	}

	type structE struct {
		field1 string
		field2 string
		field3 []any
	}

	w := NewWorkflow[*structA, string]()

	w.
		AddLambdaNode(
			"B",
			InvokableLambda(func(context.Context, string) (*structB, error) {
				return &structB{field1: "1", field2: 2}, nil
			}),
			WithWorkflowNodeName("node B")).
		AddInput(&Mapping{
			From:      "START",
			FromField: "field1",
		})

	w.
		AddLambdaNode(
			"C",
			InvokableLambda(func(context.Context, int) (map[string]string, error) {
				return map[string]string{"key2": "value2"}, nil
			})).
		AddInput(&Mapping{
			From:      "START",
			FromField: "field2",
		})

	w.
		AddLambdaNode(
			"D",
			InvokableLambda(func(context.Context, []any) ([]any, error) {
				return make([]any, 0), nil
			})).
		AddInput(&Mapping{
			From:      "START",
			FromField: "field3",
		})

	w.
		AddLambdaNode(
			"E",
			InvokableLambda(func(context.Context, *structE) (string, error) {
				return "", nil
			})).
		AddInput(&Mapping{
			From:      "B",
			FromField: "field1",
			ToField:   "field2",
		}, &Mapping{
			From:       "C",
			FromMapKey: "key2",
			ToField:    "field3",
		}, &Mapping{
			From:    "D",
			ToField: "field3",
		})

	w.AddEnd([]*Mapping{{
		From: "E",
	}})
}
