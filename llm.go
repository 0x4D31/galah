package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/googleai/vertex"
	"github.com/tmc/langchaingo/llms/openai"
)

type LLMConfig struct {
	APIKey        string
	Provider      string
	Model         string
	CloudProject  string
	CloudLocation string
}

var supportsJSONMode = map[string]bool{
	"gpt-4-turbo":            true,
	"gpt-4-turbo-2024-04-09": true,
	"gpt-4-1106-preview":     true,
	"gpt-3.5-turbo-1106":     true,
}

var supportsSystemPrompt = map[string]bool{
	"openai": true,
}

const systemPrompt = `Return JSON output and format your output as follows: ` +
	`{"Headers": {"headerName1": "headerValue1", "headerName2": "headerValue2"}, "Body": "httpBody"}`

// InitializeLLMClient initializes the LLM client based on the configured provider and model name.
func (app *App) initializeLLMClient(ctx context.Context) (llms.Model, error) {
	switch app.LLMConfig.Provider {
	case "openai":
		// TODO: Set temperature.
		opts := []openai.Option{
			openai.WithModel(app.LLMConfig.Model),
			openai.WithToken(app.LLMConfig.APIKey),
		}
		if supportsJSONMode[app.LLMConfig.Model] {
			opts = append(opts, openai.WithResponseFormat(openai.ResponseFormatJSON))
		}
		m, err := openai.New(opts...)
		if err != nil {
			return nil, err
		}
		return m, nil
	case "gcp-vertex":
		opts := []googleai.Option{
			googleai.WithDefaultModel(app.LLMConfig.Model),
			googleai.WithCloudProject(app.LLMConfig.CloudProject),
			googleai.WithCloudLocation(app.LLMConfig.CloudLocation),
		}
		m, err := vertex.New(ctx, opts...)
		if err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, errors.New("unsupported llm provider")
	}
}

func (app *App) generateLLMResponse(r *http.Request) (string, error) {
	ctx := r.Context()
	messages, err := app.createMessageContent(r)
	if err != nil {
		return "", err
	}

	response, err := app.Client.GenerateContent(ctx, messages)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("no valid response from the model")
	}
	return cleanResponse(response.Choices[0].Content), nil
}

func (app *App) createMessageContent(r *http.Request) ([]llms.MessageContent, error) {
	httpReq, err := httputil.DumpRequest(r, true)
	if err != nil {
		return nil, err
	}
	userPrompt := fmt.Sprintf(app.Config.PromptTemplate, httpReq)

	if supportsSystemPrompt[app.LLMConfig.Provider] {
		return []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
		}, nil
	}

	userPrompt += "\n" + systemPrompt
	return []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}, nil
}

func cleanResponse(input string) string {
	// Remove markdown code block backticks and json specifier.
	re := regexp.MustCompile("^```(?:json)?|```$")
	cleaned := re.ReplaceAllString(input, "")

	return strings.TrimSpace(cleaned)
}
