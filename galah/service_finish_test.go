package galah

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0x4d31/galah/internal/cache"
	"github.com/0x4d31/galah/internal/config"
	"github.com/tmc/langchaingo/llms"
)

const okJSON = `{"headers":{"Content-Type":"text/plain"},"body":"hi"}`

func newFinishService(t *testing.T) *Service {
	t.Helper()
	cfg := &config.Config{SystemPrompt: "sys", UserPrompt: "prompt: %q"}
	svc, err := NewServiceFromConfig(context.Background(), cfg, nil, Options{
		LLMProvider: "openai", LLMModel: "gpt-3.5-turbo", LLMAPIKey: "dummy",
		EventLogFile: filepath.Join(t.TempDir(), "eventlog.json"), CacheDBFile: ":memory:", CacheDuration: 1,
		GenerationTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewServiceFromConfig error: %v", err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

func cached(t *testing.T, svc *Service, key string) string {
	t.Helper()
	var v string
	// newest row, as cache.CheckCache reads it (the table has no unique key: a refresh adds a row)
	if err := svc.Cache.QueryRow("SELECT response FROM cache WHERE key = ? ORDER BY cachedAt DESC LIMIT 1", key).Scan(&v); err != nil {
		return ""
	}
	return v
}

// A client that hangs up mid-generation must not cancel it: the response is still produced and cached.
func TestGenerationSurvivesClientHangup(t *testing.T) {
	svc := newFinishService(t)
	started := make(chan struct{})
	svc.Model = &MockModel{GenerateContentFunc: func(ctx context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
		close(started)
		select {
		case <-time.After(200 * time.Millisecond): // the "model" is slow
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: okJSON}}}, nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("POST", "http://example.com/api/chat", nil).WithContext(ctx)
	req.RemoteAddr = "198.51.100.1:1234"
	go func() { <-started; cancel() }() // the scanner gives up while the model is still working

	resp, err := svc.GenerateHTTPResponse(req, "8080")
	if err != nil {
		t.Fatalf("generation was cancelled with the client: %v", err)
	}
	if string(resp) != okJSON {
		t.Fatalf("got %s", resp)
	}
	if got := cached(t, svc, cache.GetCacheKey(req, "8080")); got != okJSON {
		t.Fatalf("response not cached after the client left: %q", got)
	}
}

// The generation still has a deadline of its own.
func TestGenerationTimeout(t *testing.T) {
	svc := newFinishService(t)
	svc.GenerationTimeout = 50 * time.Millisecond
	svc.Model = &MockModel{GenerateContentFunc: func(ctx context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	req := httptest.NewRequest("GET", "http://example.com/slow", nil)
	req.RemoteAddr = "198.51.100.1:1234"
	// galah's llm package wraps errors as text (no %w), so check the message
	if _, err := svc.GenerateHTTPResponse(req, "8080"); err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("want a deadline error, got %v", err)
	}
}

// Concurrent requests for the same port+URL share one generation.
func TestConcurrentRequestsShareOneGeneration(t *testing.T) {
	svc := newFinishService(t)
	var calls int32
	release := make(chan struct{})
	svc.Model = &MockModel{GenerateContentFunc: func(ctx context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
		atomic.AddInt32(&calls, 1)
		<-release
		return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: okJSON}}}, nil
	}}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "http://example.com/api/chat", nil)
			req.RemoteAddr = "198.51.100.1:1234"
			if resp, err := svc.GenerateHTTPResponse(req, "8080"); err != nil || string(resp) != okJSON {
				t.Errorf("got %q, %v", resp, err)
			}
		}()
	}
	time.Sleep(100 * time.Millisecond) // let all five arrive while the first is generating
	close(release)
	wg.Wait()
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Fatalf("model called %d times, want 1", n)
	}
}

// An expired entry comes back with ErrCacheExpired (for serve-stale), and RefreshInBackground replaces it.
func TestExpiredResponseAndBackgroundRefresh(t *testing.T) {
	svc := newFinishService(t)
	req := httptest.NewRequest("GET", "http://example.com/old", nil)
	req.RemoteAddr = "198.51.100.1:1234"
	key := cache.GetCacheKey(req, "8080")
	if _, err := svc.Cache.Exec("INSERT INTO cache (cachedAt, key, response) VALUES (?, ?, ?)", time.Now().Add(-48*time.Hour), key, []byte("old")); err != nil {
		t.Fatal(err)
	}
	resp, err := cache.CheckCache(svc.Cache, req, "8080", 1)
	if !errors.Is(err, cache.ErrCacheExpired) || string(resp) != "old" {
		t.Fatalf("want the stale response with ErrCacheExpired, got %q, %v", resp, err)
	}
	done := make(chan struct{})
	svc.Model = &MockModel{GenerateContentFunc: func(ctx context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
		defer close(done)
		return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: okJSON}}}, nil
	}}
	svc.RefreshInBackground(req, "8080")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("background refresh never ran")
	}
	for i := 0; i < 50 && cached(t, svc, key) != okJSON; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if got := cached(t, svc, key); got != okJSON {
		t.Fatalf("cache not refreshed: %q", got)
	}
}

// With MaxGenerations set, a burst of unique URLs never runs more than that many generations at once; the rest fail
// fast with ErrGenerationBusy instead of queueing on the model.
func TestMaxGenerationsCap(t *testing.T) {
	svc := newFinishService(t)
	svc.slots = newSlots(2)
	var running, peak int32
	release := make(chan struct{})
	svc.Model = &MockModel{GenerateContentFunc: func(ctx context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
		n := atomic.AddInt32(&running, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
				break
			}
		}
		<-release
		atomic.AddInt32(&running, -1)
		return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: okJSON}}}, nil
	}}
	var wg sync.WaitGroup
	var busy, ok int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", fmt.Sprintf("http://example.com/unique-%d", i), nil)
			req.RemoteAddr = "198.51.100.1:1234"
			_, err := svc.GenerateHTTPResponse(req, "8080")
			switch {
			case errors.Is(err, ErrGenerationBusy):
				atomic.AddInt32(&busy, 1)
			case err == nil:
				atomic.AddInt32(&ok, 1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	time.Sleep(150 * time.Millisecond)
	close(release)
	wg.Wait()
	if peak > 2 {
		t.Fatalf("peak concurrent generations %d, want <= 2", peak)
	}
	if ok != 2 || busy != 6 {
		t.Fatalf("ok=%d busy=%d, want 2 and 6", ok, busy)
	}
}

// The cap is on by default: 0 means DefaultMaxGenerations, only a negative value turns it off.
func TestMaxGenerationsDefault(t *testing.T) {
	if c := cap(newSlots(0)); c != DefaultMaxGenerations || DefaultMaxGenerations != 4 {
		t.Fatalf("default cap %d, want %d (= 4)", c, DefaultMaxGenerations)
	}
	if s := newSlots(-1); s != nil {
		t.Fatal("-1 must mean no cap")
	}
	if c := cap(newSlots(2)); c != 2 {
		t.Fatalf("explicit cap %d, want 2", c)
	}
	if c := cap(newFinishService(t).slots); c != DefaultMaxGenerations {
		t.Fatalf("a service built without MaxGenerations has cap %d, want the default", c)
	}
}
