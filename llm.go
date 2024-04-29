package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var supportsJSONMode = map[string]bool{
	"gpt-4-turbo":            true,
	"gpt-4-turbo-2024-04-09": true,
	"gpt-4-1106-preview":     true,
	"gpt-3.5-turbo-1106":     true,
}

var exampleOutput = `{"Headers": {"headerName1": "headerValue1", "headerName2": "headerValue2"}, "Body": "httpBody"}`

// InitializeLLMClient initializes the correct LLM client based on the configured model name.
func InitializeLLMClient(llm LLMConfig) (llms.Model, error) {
	switch llm.Provider {
	case "openai":
		// TODO: Set temperature.
		opts := []openai.Option{
			openai.WithModel(llm.Model),
			openai.WithToken(llm.APIKey),
		}
		if supportsJSONMode[llm.Model] {
			opts = append(opts, openai.WithResponseFormat(openai.ResponseFormatJSON))
		}
		return openai.New(opts...)
	default:
		return nil, errors.New("unsupported llm provider")
	}
}

func (app *App) GenerateLLMResponse(r *http.Request) (string, error) {
	ctx := context.Background()
	systemPrompt := "Return JSON output and format your output as follows: " + exampleOutput

	httpReq, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Fatal(err)
	}
	userPrompt := fmt.Sprintf(app.Config.PromptTemplate, httpReq)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}
	response, err := app.Model.GenerateContent(ctx, messages)

	if len(response.Choices) > 0 {
		return cleanResponse(response.Choices[0].Content), nil
	}

	return "", errors.New("no valid response from the model")
}

func cleanResponse(input string) string {
	// Remove markdown code block backticks and json specifier.
	re := regexp.MustCompile("^```(?:json)?|```$")
	cleaned := re.ReplaceAllString(input, "")

	return strings.TrimSpace(cleaned)
}
