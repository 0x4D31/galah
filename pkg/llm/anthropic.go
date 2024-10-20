package llm

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

func initAnthropicClient(config Config) (llms.Model, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	opts := []anthropic.Option{
		anthropic.WithModel(config.Model),
		anthropic.WithToken(config.APIKey),
		anthropic.WithBaseURL(config.ServerURL),
	}
	m, err := anthropic.New(opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}
