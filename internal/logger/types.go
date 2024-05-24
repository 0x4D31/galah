package logger

import (
	"github.com/0x4d31/galah/pkg/enrich"
	"github.com/0x4d31/galah/pkg/llm"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	EnrichCache *enrich.Default
	EventLogger *logrus.Logger
	LLMConfig   llm.Config
	Logger      *logrus.Logger
}

type HTTPRequest struct {
	Body                string `json:"body"`
	BodySha256          string `json:"bodySha256"`
	Headers             string `json:"headers"`
	HeadersSorted       string `json:"headersSorted"`
	HeadersSortedSha256 string `json:"headersSortedSha256"`
	Method              string `json:"method"`
	ProtocolVersion     string `json:"protocolVersion"`
	Request             string `json:"request"`
	UserAgent           string `json:"userAgent"`
}

type LLM struct {
	Model       string  `json:"model"`
	Provider    string  `json:"provider"`
	Temperature float64 `json:"temperature"`
}
