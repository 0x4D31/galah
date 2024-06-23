package logger

import (
	"github.com/0x4d31/galah/pkg/enrich"
	"github.com/0x4d31/galah/pkg/llm"
	"github.com/sirupsen/logrus"
)

// Logger contains the components for logging.
type Logger struct {
	EnrichCache *enrich.Enricher
	EventLogger *logrus.Logger
	LLMConfig   llm.Config
	Logger      *logrus.Logger
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
	UserAgent           string            `json:"userAgent"`
}

// LLM contains information about the large language model.
type LLM struct {
	Model       string  `json:"model"`
	Provider    string  `json:"provider"`
	Temperature float64 `json:"temperature"`
}
