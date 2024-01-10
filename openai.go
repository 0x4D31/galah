package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func GenerateOpenAIResponse(cfg *Config, r *http.Request) (string, error) {
	// Create a prompt based on the HTTP request
	httpReq, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Fatal(err)
	}
	prompt := fmt.Sprintf(cfg.PromptTemplate, httpReq)

	// Set up API call options
	client := openai.NewClient(cfg.APIKey)
	options := openai.ChatCompletionRequest{
		Model: cfg.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Call the API
	ctx := context.Background()
	response, err := client.CreateChatCompletion(ctx, options)
	if err != nil {
		log.Printf("Error generating OpenAI response: %v", err)
		return "", err
	}

	if len(response.Choices) > 0 {
		return strings.TrimSpace(response.Choices[0].Message.Content), nil
	}

	return "", errors.New("no valid response from OpenAI")
}
