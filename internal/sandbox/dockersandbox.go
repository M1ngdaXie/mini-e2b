package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

type DockerSandbox struct {
	ID string
}

func (d *DockerSandbox) Exec(ctx context.Context, cmd string) (ExecResult, error) {
	dcmd := exec.CommandContext(ctx, "docker", "exec", d.ID, "sh", "-c", cmd)
	var stdout, stderr bytes.Buffer
	dcmd.Stdout = &stdout
	dcmd.Stderr = &stderr
	err := dcmd.Run()
	var exitErr *exec.ExitError
	if err != nil {
		if errors.As(err, &exitErr) {
			return ExecResult{Stdout: stdout.String(), StdErr: stderr.String(), ExitCode: exitErr.ExitCode()}, nil
		}
		return ExecResult{}, fmt.Errorf("Error running command: %w", err)
	}

	return ExecResult{Stdout: stdout.String(), StdErr: stderr.String(), ExitCode: 0}, nil
}
