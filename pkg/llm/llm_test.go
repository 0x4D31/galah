package llm_test

import (
	"context"
	"testing"

	"github.com/0x4d31/galah/pkg/llm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

type MockModel struct {
	GenerateContentFunc func(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error)
	CallFunc            func(ctx context.Context, method string, opts ...llms.CallOption) (string, error)
}

var ErrorLogger *logrus.Logger

func (m *MockModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, messages, opts...)
	}
	return nil, nil
}

func (m *MockModel) Call(ctx context.Context, method string, opts ...llms.CallOption) (string, error) {
	if m.CallFunc != nil {
		return m.CallFunc(ctx, method, opts...)
	}
	return "", nil
}

func TestGenerateLLMResponse(t *testing.T) {
	tests := []struct {
		name                string
		mockGenerateContent func(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error)
		errorMessage        string
		wantError           bool
	}{
		{
			name: "emptyLLMResponseChoices",
			mockGenerateContent: func(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
				return &llms.ContentResponse{
					Choices: []*llms.ContentChoice{},
				}, nil
			},
			errorMessage: "emptyLLMResponse: no choices available",
			wantError:    true,
		},
		{
			name: "emptyLLMResponseContent",
			mockGenerateContent: func(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
				return &llms.ContentResponse{
					Choices: []*llms.ContentChoice{
						{Content: ""},
					},
				}, nil
			},
			errorMessage: "emptyLLMResponse: content of first choice is empty",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &MockModel{
				GenerateContentFunc: tt.mockGenerateContent,
			}
			messages := []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeHuman, "test message"),
			}

			_, err := llm.GenerateLLMResponse(context.Background(), model, 1.0, messages)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, tt.errorMessage, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "validJSON",
			input:     `{"headers": {"headerName1": "headerValue1"}, "body": "httpBody"}`,
			expectErr: false,
		},
		{
			name:      "invalidJSONFormat",
			input:     `{"headers": {"headerName1": "headerValue1", "body": "httpBody"`,
			expectErr: true,
			errMsg:    "input is not valid JSON",
		},
		{
			name:      "unmarshalError",
			input:     `{"headers": "headerValue1", "body": "httpBody"}`,
			expectErr: true,
			errMsg:    "error unmarshalling JSON",
		},
		{
			name:      "validationErrorMissingHeaders",
			input:     `{"body": "httpBody"}`,
			expectErr: true,
			errMsg:    "validation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := llm.ValidateJSON(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
