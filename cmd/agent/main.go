package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/M1ngdaXie/mini-e2b/gen/spec/process"
	"github.com/M1ngdaXie/mini-e2b/gen/spec/process/processconnect"
)

const (
	Url     = "https://api.deepseek.com/chat/completions"
	ExecUrl = "http://10.200.0.2:43838/exec"
)

var apiKey = os.Getenv("DEEPSEEK_API_KEY")

var client = &http.Client{
	Timeout: 60 * time.Second,
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ChatReq struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Tools    []Tool    `json:"tools,omitempty"`
}
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

func main() {
	var messages []Message
	messages = append(messages, Message{
		Role:    "system",
		Content: "You are smart Network Engineer",
	})
	firstMessage := Message{
		Role:    "user",
		Content: "现在一个很小的服务器里面，没有网络",
	}
	messages = append(messages, firstMessage)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("-> ")
		scanner.Scan()
		line := scanner.Text()
		userMessage := Message{
			Role:    "user",
			Content: line,
		}
		messages = append(messages, userMessage)
		for {
			req := &ChatReq{
				Model:    "deepseek-flash",
				Messages: messages,
				Stream:   false,
				Tools: []Tool{
					{
						Type: "function",
						Function: ToolFunction{
							Name:        "run_command",
							Description: "在一台 Linux里执行 shell 命令，返回 stdout、stderr 和退出码",
							Parameters: json.RawMessage(`{
                    "type": "object",
                    "properties": {
                        "command": { "type": "string", "description": "要执行的 shell 命令" }
                    },
                    "required": ["command"]
                }`),
						},
					},
				},
			}
			resp, err := ChatRequest(req)
			if err != nil {
				log.Printf("Error sending request for agent : %v", err)
				return
			}
			msg := resp.Choices[0].Message
			messages = append(messages, msg)
			if resp.Choices[0].FinishReason == "tool_calls" {
				for _, tc := range msg.ToolCalls {
					fmt.Println("想调用的工具:", tc.Function.Name)       // get_weather
					fmt.Println("原始参数字符串:", tc.Function.Arguments) // {"location": "Hangzhou", ...}
					output, err := runCommandsInFc(tc.Function.Arguments)
					if err != nil {
						log.Printf("Error agent running command: %v", err)
					}

					log.Printf("Deepseek 看到的结果：%s", output)
					toolMessage := Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    output,
					}
					messages = append(messages, toolMessage)
				}
			} else {
				fmt.Println(msg.Content)
				break
			}
		}
	}
}

func ChatRequest(req *ChatReq) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("marshal body failed")
		return nil, err
	}
	request, err := http.NewRequest("POST", Url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepseek returned status %d, body: %s", resp.StatusCode, body)
	}
	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("failed to decode response: %v", err)
		return nil, err
	}

	if len(result.Choices) == 0 {
		log.Printf("deepseek returned empty choices")
		return nil, err
	}

	return &result, nil
}
func runCommandsInFc(command string) (string, error) {
	var rawCmd struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(strings.NewReader(command)).Decode(&rawCmd); err != nil {
		log.Printf("Error decoding Raw command")
		return "", err
	}
	client := processconnect.NewProcessClient(http.DefaultClient, "http://10.200.0.2:43838")
	stream, err := client.Start(context.Background(), connect.NewRequest(&process.CommandRequest{Command: rawCmd.Command}))
	if err != nil {
		log.Printf("Error Starting streams command %v", err)
		return "", err
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
		return "", err
	}
	defer stream.Close()
	type CommandResponse struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exitcode"`
	}
	response := &CommandResponse{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
	n := 1500
	if len(response.Stdout) > 4000 {
		response.Stdout = fmt.Sprintf("%s 输出过长，省略了%d字符 %s", response.Stdout[:n], len(response.Stdout)-2*n, response.Stdout[len(response.Stdout)-n:])
	}
	out, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error Encoding reponse for agent %v", err)
		return "", err
	}
	return string(out), nil
}
