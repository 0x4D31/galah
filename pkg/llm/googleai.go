package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

func initGoogleAIClient(ctx context.Context, config Config) (llms.Model, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	opts := []googleai.Option{
		googleai.WithDefaultModel(config.Model),
		googleai.WithAPIKey(config.APIKey),
	}
	m, err := googleai.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}
