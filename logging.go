package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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
	LLM          LLM          `json:"LLM"`
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

type LLM struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

var logger = logrus.StandardLogger()

func setLogLevel(level string) {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		logger.Fatalf("error parsing the log level: %s", err)
	}
	logger.SetLevel(l)
}

func (app *App) writeLog(event Event) {
	f, err := os.OpenFile(app.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Errorf("error opening log file: %s", err)
		return
	}
	defer f.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("error marshaling event to JSON: %s", err)
		return
	}

	if _, err = f.Write(append(eventJSON, '\n')); err != nil {
		logger.Errorf("error writing to log file: %s", err)
		return
	}
}

func (app *App) makeEvent(req *http.Request, resp HTTPResponse, port string) Event {
	var tags []string

	srcIP, srcPort, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		srcIP = req.RemoteAddr
		srcPort = ""
	}

	e := app.EnrichCache
	srcIPInfo, err := e.Process(srcIP)
	if err != nil {
		logger.Errorf("error getting enrichment info for %q: %s", srcIP, err)
	}
	if s := srcIPInfo.KnownScanner; s != "" {
		tags = append(tags, s)
	}

	httpRequest := extractHTTPRequestInfo(req)
	return Event{
		Timestamp:    time.Now(),
		SrcIP:        srcIP,
		SrcHost:      srcIPInfo.Host,
		SrcPort:      srcPort,
		Tags:         tags,
		SensorName:   app.Hostname,
		Port:         port,
		HTTPRequest:  httpRequest,
		HTTPResponse: resp,
		LLM: LLM{
			Provider: app.LLMConfig.Provider,
			Model:    app.LLMConfig.Model,
		},
	}
}

func extractHTTPRequestInfo(r *http.Request) HTTPRequest {
	httpRequest := HTTPRequest{}
	httpRequest.Method = r.Method
	httpRequest.ProtocolVersion = r.Proto
	httpRequest.Request = r.RequestURI
	httpRequest.UserAgent = r.UserAgent()
	httpRequest.Headers = extractHeaderValues(r.Header)
	headerKeys := extractHeaderKeys(r.Header)
	sort.Strings(headerKeys)
	httpRequest.HeadersSorted = strings.Join(headerKeys, ",")
	httpRequest.HeadersSortedSha256 = calculateHeadersSortedSha256(headerKeys)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Errorf("error reading request body: %s", err)
	}
	httpRequest.Body = string(bodyBytes)
	httpRequest.BodySha256 = func(data []byte) string {
		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:])
	}(bodyBytes)

	return httpRequest
}

func extractHeaderKeys(headers http.Header) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	return keys
}

func extractHeaderValues(headers http.Header) string {
	values := make([]string, 0, len(headers))
	for key, value := range headers {
		values = append(values, fmt.Sprintf("%s: %v", key, value))
	}
	return strings.Join(values, ", ")
}

func calculateHeadersSortedSha256(headerKeys []string) string {
	hash := sha256.Sum256([]byte(strings.Join(headerKeys, ",")))
	return hex.EncodeToString(hash[:])
}
