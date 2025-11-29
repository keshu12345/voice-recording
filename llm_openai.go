package main

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAILLM struct {
	client *openai.Client
}

func NewOpenAILLM(apiKey string) *OpenAILLM {
	return &OpenAILLM{
		client: openai.NewClient(apiKey),
	}
}

func (l *OpenAILLM) Generate(ctx context.Context, userInput string) (string, error) {
	resp, err := l.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: userInput},
			},
		},
	)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Message.Content, nil
}
