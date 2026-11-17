package server

import (
	"net/http/httptest"
	"testing"

	"github.com/0x4d31/galah/pkg/llm"
)

func TestSendResponseDropsFramingHeaders(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	body := "<!DOCTYPE html><html><head><title>Team</title></head><body>hi</body></html>"
	s.sendResponse(rec, llm.JSONResponse{
		Headers: map[string]string{
			"Content-Encoding":  "gzip",
			"Transfer-Encoding": "chunked",
			"Server":            "Apache/2.4.38",
		},
		Body: body,
	})
	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want it dropped", got)
	}
	if got := rec.Header().Get("Transfer-Encoding"); got != "" {
		t.Errorf("Transfer-Encoding = %q, want it dropped", got)
	}
	if got := rec.Header().Get("Server"); got != "Apache/2.4.38" {
		t.Errorf("Server = %q, want it kept", got)
	}
	if rec.Body.String() != body {
		t.Errorf("body = %q, want %q", rec.Body.String(), body)
	}
}
