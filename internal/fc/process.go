package fc

import (
	"net/http"
	"os/exec"
)

type VM struct {
	cmd      *exec.Cmd
	sockPath string
	client   *http.Client
	errCh    chan error
}
