package llm

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func initOpenAIClient(config Config) (llms.Model, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	opts := []openai.Option{
		openai.WithModel(config.Model),
		openai.WithToken(config.APIKey),
		openai.WithBaseURL(config.ServerURL),
	}
	m, err := openai.New(opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}
