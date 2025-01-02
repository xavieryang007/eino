package workflow

import (
	"github.com/cloudwego/eino/compose"
)

type CallOption compose.Option
type CompileOption compose.GraphCompileOption
type AddNodeOption compose.GraphAddNodeOpt
type NewWorkflowOption compose.NewGraphOption

func WithMaxRunSteps(maxRunSteps int) CompileOption {
	return CompileOption(compose.WithMaxRunSteps(maxRunSteps))
}

func WithWorkflowName(workflowName string) CompileOption {
	return CompileOption(compose.WithGraphName(workflowName))
}

func WithNodeName(n string) AddNodeOption {
	return AddNodeOption(compose.WithNodeName(n))
}

func WithCompileOptions(opts ...CompileOption) AddNodeOption {
	gCompileOpts := make([]compose.GraphCompileOption, 0, len(opts))
	for _, opt := range opts {
		gCompileOpts = append(gCompileOpts, compose.GraphCompileOption(opt))
	}
	return AddNodeOption(compose.WithGraphCompileOptions(gCompileOpts...))
}

func WithStatePreHandler[I, S any](pre compose.StatePreHandler[I, S]) AddNodeOption {
	return AddNodeOption(compose.WithStatePreHandler(pre))
}

func WithStatePostHandler[O, S any](post compose.StatePostHandler[O, S]) AddNodeOption {
	return AddNodeOption(compose.WithStatePostHandler(post))
}

func WithStreamStatePreHandler[I, S any](pre compose.StreamStatePreHandler[I, S]) AddNodeOption {
	return AddNodeOption(compose.WithStreamStatePreHandler(pre))
}

func WithStreamStatePostHandler[O, S any](post compose.StreamStatePostHandler[O, S]) AddNodeOption {
	return AddNodeOption(compose.WithStreamStatePostHandler(post))
}

func WithGenLocalState[S any](gls compose.GenLocalState[S]) NewWorkflowOption {
	return NewWorkflowOption(compose.WithGenLocalState(gls))
}
