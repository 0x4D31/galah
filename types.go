package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/0x4d31/galah/enrich"
	"github.com/tmc/langchaingo/llms"
)

type Event struct {
	Timestamp    time.Time    `json:"timestamp"`
	SrcIP        string       `json:"srcIP"`
	SrcHost      string       `json:"srcHost"`
	Tags         []string     `json:"tags"`
	SrcPort      string       `json:"srcPort"`
	SensorName   string       `json:"sensorName"`
	Port         string       `json:"port"`
	HTTPRequest  HTTPRequest  `json:"httpRequest"`
	HTTPResponse HTTPResponse `json:"httpResponse"`
	// TODO: Sessionize the incoming requests based on the sessionTTL and source IP.
	// SessionID    string       `json:"sessionID"`
}

type HTTPRequest struct {
	Method              string `json:"method"`
	ProtocolVersion     string `json:"protocolVersion"`
	Request             string `json:"request"`
	UserAgent           string `json:"userAgent"`
	Headers             string `json:"headers"`
	HeadersSorted       string `json:"headersSorted"`
	HeadersSortedSha256 string `json:"headersSortedSha256"`
	Body                string `json:"body"`
	BodySha256          string `json:"bodySha256"`
}

type HTTPResponse struct {
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type App struct {
	Model       llms.Model
	Config      *Config
	DB          *sql.DB
	OutputFile  string
	Verbose     bool
	Servers     map[uint16]*http.Server
	Hostname    string
	EnrichCache *enrich.Default
}
