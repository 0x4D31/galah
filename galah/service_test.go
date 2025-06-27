package galah

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0x4d31/galah/internal/cache"
	"github.com/0x4d31/galah/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tmc/langchaingo/llms"
)

func TestNewServiceWithDefaults(t *testing.T) {
	// Ensure working directory is project root so default paths exist.
	wd, _ := os.Getwd()
	os.Chdir("..")
	t.Cleanup(func() { os.Chdir(wd) })

	svc, err := NewService(context.Background(), Options{
		LLMProvider: "openai",
		LLMModel:    "gpt-3.5-turbo-1106",
		LLMAPIKey:   "dummy",
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	if svc == nil {
		t.Fatal("NewService returned nil service")
	}
	// Cleanup files created by defaults
	t.Cleanup(func() {
		svc.Cache.Close()
		os.Remove(DefaultCacheDBFile)
		os.Remove(DefaultEventLogFile)
	})
}

type MockModel struct {
	GenerateContentFunc func(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error)
}

func (m *MockModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, messages, opts...)
	}
	return nil, nil
}

func (m *MockModel) Call(ctx context.Context, method string, opts ...llms.CallOption) (string, error) {
	return "", nil
}

func TestGenerateHTTPResponse(t *testing.T) {
	cfg := &config.Config{SystemPrompt: "sys", UserPrompt: "prompt: %q"}
	tmpLog := filepath.Join(t.TempDir(), "eventlog.json")
	svc, err := NewServiceFromConfig(context.Background(), cfg, nil, Options{
		LLMProvider:  "openai",
		LLMModel:     "gpt-3.5-turbo",
		LLMAPIKey:    "dummy",
		EventLogFile: tmpLog,
		CacheDBFile:  ":memory:",
	})
	if err != nil {
		t.Fatalf("NewServiceFromConfig error: %v", err)
	}
	svc.Model = &MockModel{
		GenerateContentFunc: func(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
			return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: `{"headers":{"Content-Type":"text/plain"},"body":"hi"}`}}}, nil
		},
	}
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	resp, err := svc.GenerateHTTPResponse(req, "8080")
	if err != nil {
		t.Fatalf("GenerateHTTPResponse error: %v", err)
	}
	want := `{"headers":{"Content-Type":"text/plain"},"body":"hi"}`
	if string(resp) != want {
		t.Fatalf("expected %s, got %s", want, string(resp))
	}
}

func TestGenerateHTTPResponseCaches(t *testing.T) {
	cfg := &config.Config{SystemPrompt: "sys", UserPrompt: "prompt: %q"}
	tmpLog := filepath.Join(t.TempDir(), "eventlog.json")
	svc, err := NewServiceFromConfig(context.Background(), cfg, nil, Options{
		LLMProvider:   "openai",
		LLMModel:      "gpt-3.5-turbo",
		LLMAPIKey:     "dummy",
		EventLogFile:  tmpLog,
		CacheDBFile:   ":memory:",
		CacheDuration: 1,
	})
	if err != nil {
		t.Fatalf("NewServiceFromConfig error: %v", err)
	}
	svc.Model = &MockModel{
		GenerateContentFunc: func(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
			return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: `{"headers":{"Content-Type":"text/plain"},"body":"hi"}`}}}, nil
		},
	}
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	resp, err := svc.GenerateHTTPResponse(req, "8080")
	if err != nil {
		t.Fatalf("GenerateHTTPResponse error: %v", err)
	}

	var cached string
	row := svc.Cache.QueryRow("SELECT response FROM cache WHERE key = ?", cache.GetCacheKey(req, "8080"))
	if err := row.Scan(&cached); err != nil {
		t.Fatalf("failed to read cache: %v", err)
	}
	if cached != string(resp) {
		t.Fatalf("cached response mismatch: got %s", cached)
	}
}
