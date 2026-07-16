package main

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	client := openai.NewClient("test-key")
	model := "gpt-4o"
	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: "hello"},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
