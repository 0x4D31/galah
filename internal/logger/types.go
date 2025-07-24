package logger

import (
	"os"

	"github.com/0x4d31/galah/pkg/enrich"
	"github.com/0x4d31/galah/pkg/llm"
	cblog "github.com/charmbracelet/log"
)

// Logger contains the components for logging.
type Logger struct {
	EnrichCache *enrich.Enricher
	Sessionizer *Sessionizer
	EventLogger *cblog.Logger
	EventFile   *os.File
	LLMConfig   llm.Config
	Logger      *cblog.Logger
}

// HTTPRequest contains information about the HTTP request.
type HTTPRequest struct {
	Body                string            `json:"body"`
	BodySha256          string            `json:"bodySha256"`
	Headers             map[string]string `json:"headers"`
	HeadersSorted       string            `json:"headersSorted"`
	HeadersSortedSha256 string            `json:"headersSortedSha256"`
	Method              string            `json:"method"`
	ProtocolVersion     string            `json:"protocolVersion"`
	Request             string            `json:"request"`
	SessionID           string            `json:"sessionID"`
	UserAgent           string            `json:"userAgent"`
}

// ResponseMetadata holds metadata about the generated response
type ResponseMetadata struct {
	GenerationSource string  `json:"generationSource"`
	Info             LLMInfo `json:"info,omitempty"`
}

// LLMInfo holds information about the large language model
type LLMInfo struct {
	Model       string  `json:"model,omitempty"`
	Provider    string  `json:"provider,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}
