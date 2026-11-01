package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	ctx := context.Background()
	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("DEEPSEEK_API_KEY")),
		option.WithBaseURL("https://api.deepseek.com"),
	)
	question := "Tell me about TUN mode"
	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(question),
		},
		Model: "deepseek-flash",
	})

	defer func() { _ = stream.Close() }()

	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) > 0 {
			fmt.Print(event.Choices[0].Delta.Content)
		}
	}

	if stream.Err() != nil {
		panic(stream.Err())
	}
}
