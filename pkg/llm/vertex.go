package llm

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/googleai/vertex"
)

func initVertexClient(ctx context.Context, config Config) (llms.Model, error) {
	if config.CloudLocation == "" || config.CloudProject == "" {
		return nil, fmt.Errorf("Cloud project ID and location are required")
	}
	opts := []googleai.Option{
		googleai.WithDefaultModel(config.Model),
		googleai.WithCloudProject(config.CloudProject),
		googleai.WithCloudLocation(config.CloudLocation),
	}
	m, err := vertex.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return m, nil
}
