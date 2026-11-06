package llm

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

// defaultAnthropicMaxTokens is the output limit sent to the Anthropic Messages
// API. langchaingo sends 256 when no limit is given, which cuts most generated
// HTML pages short and leaves Galah with invalid JSON (a 500 for the visitor).
// LLM_MAX_TOKENS overrides it.
const defaultAnthropicMaxTokens = 4096

// anthropicModel adds an output token limit to every call.
type anthropicModel struct {
	*anthropic.LLM
	maxTokens int
}

func (m *anthropicModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	opts := append([]llms.CallOption{llms.WithMaxTokens(m.maxTokens)}, options...)
	resp, err := m.LLM.GenerateContent(ctx, messages, opts...)
	if err != nil || resp == nil {
		return resp, err
	}
	// langchaingo turns every content block into a choice and Galah reads the
	// first one. Models that think before answering return a thinking block
	// first, whose text is empty, so keep only the blocks that carry text.
	text := make([]*llms.ContentChoice, 0, len(resp.Choices))
	for _, c := range resp.Choices {
		if c != nil && c.Content != "" {
			text = append(text, c)
		}
	}
	if len(text) > 0 {
		resp.Choices = text
	}
	return resp, nil
}

func anthropicMaxTokens() (int, error) {
	v := os.Getenv("LLM_MAX_TOKENS")
	if v == "" {
		return defaultAnthropicMaxTokens, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("LLM_MAX_TOKENS must be a positive number, got %q", v)
	}
	return n, nil
}

func initAnthropicClient(config Config) (llms.Model, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	maxTokens, err := anthropicMaxTokens()
	if err != nil {
		return nil, err
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
	return &anthropicModel{LLM: m, maxTokens: maxTokens}, nil
}
