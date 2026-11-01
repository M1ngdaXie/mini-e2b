package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type RunResponse struct {
	Name string
	ID   string
}

func main() {
	name := "myapline"
	resp, err := StartWithImage("docker.io/library/alpine", time.Second, name)
	if err != nil {
		log.Printf("Error : %v", err)
		return
	}
	time.Sleep(1 * time.Second)
	fmt.Printf("Name is : %s, Id is %s ", resp.Name, resp.ID)

	output, err := RunCommands("uname -a; head -2 /etc/os-release; id", name)
	time.Sleep(1 * time.Second)
	if err != nil {
		log.Printf("Error : %v", err)
		return
	}
	fmt.Println(output)
	time.Sleep(1 * time.Second)
	StopPod(name)
}
func StartWithImage(imageLink string, timeout time.Duration, name string) (*RunResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "podman", "run", "-d", "--name", name, imageLink, "sleep", "infinity")
	output, err := cmd.CombinedOutput()
	defer StopPod(name)
	if err != nil {
		log.Printf("Error Getting start command outputs %s: %v", output, err)
		return nil, err
	}
	resp := &RunResponse{
		Name: name,
		ID:   strings.TrimSpace(string(output)),
	}
	return resp, nil
}
func RunCommands(inputs string, name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "podman", "exec", name, "sh", "-c", inputs)

	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Error Getting start command outputs : %s: %v", output, err)
		return "", err
	}
	return string(output), nil
}

func StopPod(name string) {
	cmd := exec.Command("podman", "rm", "-f", name)
	err := cmd.Run()
	if err != nil {
		log.Printf("Error stoping pod : %s : %v", name, err)
		return
	}
	fmt.Printf("Sucessfully stopped %s", name)
}
