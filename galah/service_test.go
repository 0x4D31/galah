package galah

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0x4d31/galah/internal/cache"
	"github.com/0x4d31/galah/internal/config"
	"github.com/0x4d31/galah/pkg/llm"
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

func TestNewServiceFromConfigWithDefaults(t *testing.T) {
	wd, _ := os.Getwd()
	os.Chdir("..")
	t.Cleanup(func() { os.Chdir(wd) })

	svc, err := NewServiceFromConfig(context.Background(), &config.Config{}, nil, Options{
		LLMProvider: "openai",
		LLMModel:    "gpt-3.5-turbo-1106",
		LLMAPIKey:   "dummy",
	})
	if err != nil {
		t.Fatalf("NewServiceFromConfig returned error: %v", err)
	}
	if svc == nil {
		t.Fatal("NewServiceFromConfig returned nil service")
	}

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

func TestServiceClose(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDB := filepath.Join(tmpDir, "cache.db")
	tmpLog := filepath.Join(tmpDir, "eventlog.json")

	svc, err := NewServiceFromConfig(context.Background(), &config.Config{}, nil, Options{
		LLMProvider:  "openai",
		LLMModel:     "gpt-3.5-turbo-1106",
		LLMAPIKey:    "dummy",
		CacheDBFile:  tmpDB,
		EventLogFile: tmpLog,
	})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	if err := svc.Close(); err != nil {
		t.Fatalf("Service.Close error: %v", err)
	}

	if err := svc.Cache.Ping(); err == nil {
		t.Error("expected closed database error")
	}

	if _, err := svc.EventLogger.EventFile.Write([]byte("test")); err == nil {
		t.Error("expected write to closed file to fail")
	}
}

func TestServiceCheckCache(t *testing.T) {
	cfg := &config.Config{}
	tmpLog := filepath.Join(t.TempDir(), "eventlog.json")
	svc, err := NewServiceFromConfig(context.Background(), cfg, nil, Options{
		LLMProvider:   "openai",
		LLMModel:      "gpt-3.5-turbo-1106",
		LLMAPIKey:     "dummy",
		EventLogFile:  tmpLog,
		CacheDBFile:   ":memory:",
		CacheDuration: 1,
	})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	defer svc.Close()

	req := httptest.NewRequest("GET", "http://example.com/cached", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	key := cache.GetCacheKey(req, "8080")
	if err := cache.StoreResponse(svc.Cache, key, []byte("cached")); err != nil {
		t.Fatalf("StoreResponse error: %v", err)
	}

	data, err := svc.CheckCache(req, "8080")
	if err != nil {
		t.Fatalf("CheckCache error: %v", err)
	}
	if string(data) != "cached" {
		t.Fatalf("expected cached response, got %s", string(data))
	}
}

func TestServiceLogEvent(t *testing.T) {
	cfg := &config.Config{}
	tmpDir := t.TempDir()
	tmpLog := filepath.Join(tmpDir, "eventlog.json")

	svc, err := NewServiceFromConfig(context.Background(), cfg, nil, Options{
		LLMProvider:  "openai",
		LLMModel:     "gpt-3.5-turbo-1106",
		LLMAPIKey:    "dummy",
		EventLogFile: tmpLog,
		CacheDBFile:  ":memory:",
	})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	defer svc.Close()

	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp := llm.JSONResponse{Headers: map[string]string{"Content-Type": "text/plain"}, Body: "hi"}

	svc.LogEvent(req, resp, "8080", "llm", nil)

	contents, err := os.ReadFile(tmpLog)
	if err != nil {
		t.Fatalf("failed reading log file: %v", err)
	}
	if !strings.Contains(string(contents), "successfulResponse") {
		t.Fatalf("log file missing entry: %s", string(contents))
	}
}
