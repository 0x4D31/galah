package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	"github.com/go-playground/validator"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/googleai/vertex"
	"github.com/tmc/langchaingo/llms/openai"
)

type LLMConfig struct {
	Provider      string
	Model         string
	Temperature   float64
	APIKey        string
	CloudProject  string
	CloudLocation string
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
		if app.LLMConfig.APIKey == "" {
			return nil, fmt.Errorf("api key is required")
		}
		opts := []openai.Option{
			openai.WithModel(app.LLMConfig.Model),
			openai.WithToken(app.LLMConfig.APIKey),
		}
		m, err := openai.New(opts...)
		if err != nil {
			return nil, err
		}
		return m, nil
	case "gcp-vertex":
		if app.LLMConfig.CloudLocation == "" || app.LLMConfig.CloudProject == "" {
			return nil, fmt.Errorf("cloud project id and location are required")
		}
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

	response, err := app.Client.GenerateContent(
		ctx,
		messages,
		llms.WithJSONMode(),
		llms.WithTemperature(app.LLMConfig.Temperature),
	)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("empty response from the model")
	}
	resp := cleanResponse(response.Choices[0].Content)
	if err := validateJSON(resp); err != nil {
		logger.Errorf("invalid response: %s", err)
		logger.Debugf("invalid generated response: %s", resp)
		return "", err
	}

	return resp, nil
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

func validateJSON(jsonStr string) error {
	jsonBytes := []byte(jsonStr)
	// Check if the JSON format is correct
	if !json.Valid(jsonBytes) {
		return fmt.Errorf("input is not valid JSON")
	}
	// Try to unmarshal the JSON into the struct
	var resp HTTPResponse
	if err := json.Unmarshal(jsonBytes, &resp); err != nil {
		return fmt.Errorf("error unmarshalling JSON: %s", err)
	}
	// Validate the struct using the `validator` package
	validate := validator.New()
	if err := validate.Struct(resp); err != nil {
		return fmt.Errorf("validation error: %s", err)
	}

	return nil
}
