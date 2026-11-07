package sandbox

import (
	"context"
)

type Sandbox interface {
	Start(ctx context.Context) (StartResult, error)
	Stop() error
	Exec(ctx context.Context, cmd string) (ExecResult, error)
}

type StartResult struct {
	ID string
}

type ExecResult struct {
	Stdout   string
	StdErr   string
	ExitCode int
}
