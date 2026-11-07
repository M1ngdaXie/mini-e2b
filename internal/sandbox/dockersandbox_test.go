package sandbox

import (
	"context"
	"testing"
	"time"
)

func TestDockerExec(t *testing.T) {
	docker := &DockerSandbox{
		ID: "2e4729f5733f45923cbfaedeaa7f48c5dc2266024a4fa7d8f8f638a2245fd519",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := docker.Exec(ctx, "ls /nope")
	if err != nil {
		t.Fatalf("Something wrong with command test %v", err)
	}
	t.Logf("%+v", res)
}
