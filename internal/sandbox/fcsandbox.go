package sandbox

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/M1ngdaXie/mini-e2b/gen/spec/process"
	"github.com/M1ngdaXie/mini-e2b/gen/spec/process/processconnect"
	"github.com/M1ngdaXie/mini-e2b/internal/fc"
)

type FCSandbox struct {
	vm *fc.VM
}

func (f *FCSandbox) Exec(ctx context.Context, cmd string) (ExecResult, error) {
	execUrl := fmt.Sprintf("http://%s:43838", f.vm.IP)
	client := processconnect.NewProcessClient(http.DefaultClient, execUrl)
	stream, err := client.Start(ctx, connect.NewRequest(&process.CommandRequest{Command: cmd}))
	if err != nil {
		log.Printf("Error Starting streams command %v", err)
		return ExecResult{}, err
	}
	var stdout, stderr strings.Builder
	exitCode := -1
	for stream.Receive() {
		switch v := stream.Msg().Output.(type) {
		case *process.CommandResponse_Exitcode:
			exitCode = int(v.Exitcode)
		case *process.CommandResponse_Stderr:
			stderr.WriteString(string(v.Stderr))
		case *process.CommandResponse_Stdout:
			stdout.WriteString(string(v.Stdout))
			stdout.WriteString("\n")
		}
	}
	if err := stream.Err(); err != nil {
		return ExecResult{}, err
	}
	defer stream.Close()

	response := ExecResult{
		Stdout:   stdout.String(),
		StdErr:   stderr.String(),
		ExitCode: exitCode,
	}
	return response, nil
}
