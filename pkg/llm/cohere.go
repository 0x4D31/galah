package llm

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/cohere"
)

func initCohereClient(config Config) (llms.Model, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	opts := []cohere.Option{
		cohere.WithModel(config.Model),
		cohere.WithToken(config.APIKey),
	}
	m, err := cohere.New(opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}
