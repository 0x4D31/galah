package logger

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/0x4d31/galah/pkg/enrich"
	"github.com/0x4d31/galah/pkg/llm"
	"github.com/0x4d31/galah/pkg/suricata"
	cblog "github.com/charmbracelet/log"
	"github.com/google/uuid"
)

const (
	errorInvalidJSONResponse = "invalidJSONResponse"
	errorEmptyLLMResponse    = "emptyLLMResponse"
	errorContentGeneration   = "contentGenerationError"
)

// New creates a new Logger instance with the specified configuration.
func New(eventLogFile string, modelConfig llm.Config, eCache *enrich.Enricher, sessionizer *Sessionizer, l *cblog.Logger) (*Logger, error) {
	eventLogger := cblog.NewWithOptions(nil, cblog.Options{Formatter: cblog.JSONFormatter, TimeFormat: time.RFC3339Nano})
	evFile, err := os.OpenFile(eventLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	eventLogger.Out = evFile

	return &Logger{
		EnrichCache: eCache,
		Sessionizer: sessionizer,
		EventLogger: eventLogger,
		EventFile:   evFile,
		LLMConfig:   modelConfig,
		Logger:      l,
	}, nil
}

// LogError logs a failedResponse event.
func (l *Logger) LogError(r *http.Request, resp, port string, err error) {
	fields := l.commonFields(r, port)
	fields["error"] = errorFields(err, resp)
	fields["responseMetadata"] = ResponseMetadata{
		Info: LLMInfo{
			Provider:    l.LLMConfig.Provider,
			Model:       l.LLMConfig.Model,
			Temperature: l.LLMConfig.Temperature,
		},
	}

	l.EventLogger.Error("failedResponse: returned 500 internal server error", mapToArgs(fields)...)
}

// LogEvent logs a successfulResponse event, including any matched Suricata rules.
func (l *Logger) LogEvent(r *http.Request, resp llm.JSONResponse, port, respSource string, suricataMatches []suricata.Rule) {
	fields := l.commonFields(r, port)
	fields["httpResponse"] = resp

	if respSource == "llm" {
		fields["responseMetadata"] = ResponseMetadata{
			GenerationSource: respSource,
			Info: LLMInfo{
				Provider:    l.LLMConfig.Provider,
				Model:       l.LLMConfig.Model,
				Temperature: l.LLMConfig.Temperature,
			},
		}
	} else {
		fields["responseMetadata"] = ResponseMetadata{GenerationSource: respSource}
	}

	// Include Suricata match info if available
	if len(suricataMatches) > 0 {
		var matches []map[string]string
		for _, m := range suricataMatches {
			matches = append(matches, map[string]string{"sid": m.SID, "msg": m.Msg})
		}
		fields["suricataMatches"] = matches
	}

	l.EventLogger.Info("successfulResponse", mapToArgs(fields)...)
}

func (l *Logger) commonFields(r *http.Request, port string) map[string]any {
	now := time.Now()

	srcIP, srcPort, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		srcIP = r.RemoteAddr
		srcPort = ""
	}

	var tags []string
	var host string
	srcIPInfo, err := l.EnrichCache.Process(srcIP)
	if err != nil {
		l.Logger.Errorf("error getting enrichment info for %q: %s", srcIP, err)
	} else if srcIPInfo != nil {
		if s := srcIPInfo.KnownScanner; s != "" {
			tags = append(tags, s)
		}
		if h := srcIPInfo.Host; h != "" {
			host = h
		}
	}

	sensorName, err := getHostname()
	if err != nil {
		sensorName = uuid.NewString()
	}

	headerKeys := headerKeys(r.Header)
	sort.Strings(headerKeys)
	bodyBytes, _ := io.ReadAll(r.Body)

	sessionID, err := l.Sessionizer.Process(srcIP, now)
	if err != nil {
		l.Logger.Errorf("error generating session ID for %q: %s", srcIP, err)
	}

	return map[string]any{
		"eventTime":  now,
		"srcIP":      srcIP,
		"srcHost":    host,
		"srcPort":    srcPort,
		"tags":       tags,
		"sensorName": sensorName,
		"port":       port,
		"httpRequest": HTTPRequest{
			SessionID:           sessionID,
			Method:              r.Method,
			ProtocolVersion:     r.Proto,
			Request:             r.RequestURI,
			UserAgent:           r.UserAgent(),
			Headers:             convertMap(r.Header),
			HeadersSorted:       strings.Join(headerKeys, ","),
			HeadersSortedSha256: headersSortedSha256(headerKeys),
			Body:                string(bodyBytes),
			BodySha256: func(data []byte) string {
				hash := sha256.Sum256(data)
				return hex.EncodeToString(hash[:])
			}(bodyBytes),
		},
	}
}

func errorFields(err error, resp string) map[string]any {
	errMsg := err.Error()
	var errorType string

	switch {
	case strings.Contains(errMsg, errorInvalidJSONResponse):
		errorType = errorInvalidJSONResponse
		errMsg = strings.ReplaceAll(errMsg, errorInvalidJSONResponse+": ", "")
	case strings.Contains(errMsg, errorEmptyLLMResponse):
		errorType = errorEmptyLLMResponse
		errMsg = strings.ReplaceAll(errMsg, errorEmptyLLMResponse+": ", "")
	default:
		errorType = errorContentGeneration
		errMsg = strings.ReplaceAll(errMsg, errorContentGeneration+": ", "")
	}

	return map[string]any{
		"type":            errorType,
		"msg":             errMsg,
		"invalidResponse": resp,
	}
}

func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %s", err)
	}
	return hostname, nil
}

func headerKeys(headers http.Header) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	return keys
}

func headersSortedSha256(headerKeys []string) string {
	hash := sha256.Sum256([]byte(strings.Join(headerKeys, ",")))
	return hex.EncodeToString(hash[:])
}

func convertMap(input map[string][]string) map[string]string {
	result := make(map[string]string)

	for key, values := range input {
		result[key] = strings.Join(values, ", ")
	}

	return result
}

func mapToArgs(m map[string]any) []any {
	args := make([]any, 0, len(m)*2)
	for k, v := range m {
		args = append(args, k, v)
	}
	return args
}
