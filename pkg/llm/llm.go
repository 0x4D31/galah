package llm

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
)

type Config struct {
	APIKey        string
	CloudLocation string
	CloudProject  string
	Model         string
	Provider      string
	Temperature   float64
}

type JSONResponse struct {
	Headers map[string]string `json:"headers" validate:"required"`
	Body    string            `json:"body" validate:"required"`
}

var supportsSystemPrompt = map[string]bool{
	"openai": true,
}

const systemPrompt = `Return JSON output and format your output as follows: ` +
	`{"Headers": {"headerName1": "headerValue1", "headerName2": "headerValue2"}, "Body": "httpBody"}`

// New initializes the LLM client based on the configured provider and model name.
func New(ctx context.Context, config Config) (llms.Model, error) {
	switch config.Provider {
	case "openai":
		return initOpenAIClient(config)
	case "googleai":
		return initGoogleAIClient(ctx, config)
	case "gcp-vertex":
		return initVertexClient(ctx, config)
	case "anthropic":
		return initAnthropicClient(config)
	default:
		return nil, errors.New("unsupported llm provider")
	}
}

func GenerateLLMResponse(ctx context.Context, model llms.Model, temperature float64, messages []llms.MessageContent) (string, error) {
	response, err := model.GenerateContent(
		ctx,
		messages,
		llms.WithJSONMode(),
		llms.WithTemperature(temperature),
	)
	if err != nil {
		return "", fmt.Errorf("contentGenerationError: %s", err)
	}
	if response == nil {
		return "", errors.New("emptyLLMResponse: response is nil")
	}
	if len(response.Choices) == 0 {
		return "", errors.New("emptyLLMResponse: no choices available")
	}
	content := response.Choices[0].Content
	if content == "" {
		return "", errors.New("emptyLLMResponse: content of first choice is empty")
	}
	resp := cleanResponse(content)
	if err := ValidateJSON(resp); err != nil {
		return resp, fmt.Errorf("invalidJSONResponse: %s", err)
	}

	return resp, nil
}

func CreateMessageContent(r *http.Request, promptTmpl string, provider string) ([]llms.MessageContent, error) {
	httpReq, err := httputil.DumpRequest(r, true)
	if err != nil {
		return nil, err
	}
	userPrompt := fmt.Sprintf(promptTmpl, httpReq)

	if supportsSystemPrompt[provider] {
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

func ValidateJSON(jsonStr string) error {
	jsonBytes := []byte(jsonStr)
	// Check if the JSON format is correct
	if !json.Valid(jsonBytes) {
		return fmt.Errorf("input is not valid JSON")
	}
	// Try to unmarshal the JSON into the struct
	var resp JSONResponse
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
