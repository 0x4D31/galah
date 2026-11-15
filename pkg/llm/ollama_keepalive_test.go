package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// TestOllamaKeepAlive checks that --keep-alive reaches Ollama's /api/chat body
// and that it is left out (server default) when empty.
func TestOllamaKeepAlive(t *testing.T) {
	for _, tc := range []struct{ set, want string }{{"1h", "1h"}, {"", ""}} {
		var got map[string]any
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &got)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"model":"m","message":{"role":"assistant","content":"{}"},"done":true}`+"\n")
		}))
		m, err := initOllamaClient(Config{ServerURL: ts.URL, Model: "m", KeepAlive: tc.set})
		if err != nil {
			t.Fatal(err)
		}
		_, err = m.GenerateContent(context.Background(), []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "hi")})
		ts.Close()
		if err != nil {
			t.Fatalf("GenerateContent: %v", err)
		}
		ka, _ := got["keep_alive"].(string)
		if ka != tc.want {
			t.Errorf("keep_alive = %q (body %v), want %q", ka, got, tc.want)
		}
	}
}
