package llm

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func initOllamaClient(config Config) (llms.Model, error) {
	if config.ServerURL == "" {
		return nil, fmt.Errorf("Server URL is required")
	}
	opts := []ollama.Option{
		ollama.WithServerURL(config.ServerURL),
		ollama.WithModel(config.Model),
	}
	m, err := ollama.New(opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}
