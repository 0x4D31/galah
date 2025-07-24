package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/0x4d31/galah/galah"
	"github.com/0x4d31/galah/internal/cache"
	"github.com/0x4d31/galah/internal/config"
	el "github.com/0x4d31/galah/internal/logger"
	"github.com/0x4d31/galah/pkg/llm"
	"github.com/0x4d31/galah/pkg/suricata"
	cblog "github.com/charmbracelet/log"
	"github.com/tmc/langchaingo/llms"
	"golang.org/x/sync/errgroup"
)

var ignoreHeaders = map[string]bool{
	// Standard headers to ignore
	"content-length": true,
	"content-type":   true,
	"date":           true,
	"expires":        true,
	"last-modified":  true,
	// OpenAI made-up headers to ignore
	"http":     true,
	"http/1.0": true,
	"http/1.1": true,
	"http/1.2": true,
	"http/2.0": true,
}

// safeMatch wraps Suricata.Match to recover from any panics and return no matches on error.
func safeMatch(rs *suricata.RuleSet, req *http.Request, body string) (out []suricata.Rule) {
	defer func() {
		if rec := recover(); rec != nil {
			// log the panic and continue
			cblog.WithPrefix("GALAH").Errorf("panic in Suricata.Match: %v", rec)
		}
	}()
	return rs.Match(req, body)
}

// Server holds the configuration and components for running HTTP/TLS servers.
type Server struct {
	// Service handles response generation and related components.
	Service       *galah.Service
	Cache         *sql.DB
	CacheDuration int
	Interface     string
	Config        *config.Config
	Rules         []config.Rule
	EventLogger   *el.Logger
	LLMConfig     llm.Config
	Logger        *cblog.Logger
	Model         llms.Model
	Suricata      *suricata.RuleSet
	Servers       map[uint16]*http.Server
}

// StartServers starts all servers defined in the configuration.
func (s *Server) StartServers() error {
	var g errgroup.Group
	mu := sync.Mutex{}

	for _, pc := range s.Config.Ports {
		pc := pc // Capture the loop variable
		g.Go(func() error {
			return s.startServer(pc, &mu)
		})
	}

	return g.Wait()
}

func (s *Server) startServer(pc config.PortConfig, mu *sync.Mutex) error {
	server := s.SetupServer(pc)

	var err error
	switch pc.Protocol {
	case "TLS":
		err = s.StartTLSServer(server, pc)
	case "HTTP":
		err = s.StartHTTPServer(server, pc)
	default:
		err = fmt.Errorf("unknown protocol for port %d", pc.Port)
	}
	if err != nil {
		s.Logger.WithPrefix("GALAH").Errorf("error starting server on port %d: %s", pc.Port, err)
		return err
	}

	mu.Lock()
	s.Servers[pc.Port] = server
	mu.Unlock()

	return nil
}

// SetupServer configures the server with the provided settings.
func (s *Server) SetupServer(pc config.PortConfig) *http.Server {
	var ip string
	var err error

	if s.Interface != "" {
		ip, err = getInterfaceIP(s.Interface)
		if err != nil {
			s.Logger.WithPrefix("GALAH").Error(err)
		}
	}
	serverAddr := net.JoinHostPort(ip, fmt.Sprintf("%d", pc.Port))

	return &http.Server{
		Addr: serverAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.handleRequest(w, r, serverAddr, s.Rules)
		}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// StartTLSServer starts the configured TLS server.
func (s *Server) StartTLSServer(server *http.Server, pc config.PortConfig) error {
	if pc.TLSProfile == "" {
		return fmt.Errorf("TLS profile is not configured for port %d", pc.Port)
	}

	tlsConfig, ok := s.Config.Profiles[pc.TLSProfile]
	if !ok || tlsConfig.Certificate == "" || tlsConfig.Key == "" {
		return fmt.Errorf("TLS profile is incomplete for port %d", pc.Port)
	}

	s.Logger.WithPrefix("GALAH").Infof("starting HTTPS server on %s with TLS profile: %s", server.Addr, pc.TLSProfile)
	return server.ListenAndServeTLS(tlsConfig.Certificate, tlsConfig.Key)
}

// StartHTTPServer starts the configured HTTP server.
func (s *Server) StartHTTPServer(server *http.Server, pc config.PortConfig) error {
	s.Logger.WithPrefix("GALAH").Infof("starting HTTP server on %s", server.Addr)
	return server.ListenAndServe()
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request, serverAddr string, rules []config.Rule) {
	var response []byte
	var respSource string
	var err error

	// Read and capture request body for Suricata matching and LLM
	var bodyBytes []byte
	// Only read body if Suricata matching is enabled (to reduce overhead)
	if s.Suricata != nil && r.Body != nil {
		const maxBodySize = 1 << 20 // 1 MiB cap
		limited := io.LimitReader(r.Body, maxBodySize)
		bodyBytes, err = io.ReadAll(limited)
		if err != nil {
			s.Logger.WithPrefix("GALAH").Errorf("error reading request body: %s", err)
		}
		// Restore Body for downstream handlers (and LLM)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	reqBodyStr := string(bodyBytes)

	port := s.extractPort(serverAddr)
	s.Logger.WithPrefix("GALAH").Infof("port %s received a request for %q, from source %s", port, r.URL.String(), r.RemoteAddr)

	// Check for applicable rules before generating the response
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if matched, err := regexp.MatchString(rule.HTTPRequestRegex, r.RequestURI); matched {
			if rule.Response.Type == "static" {
				resp, err := os.ReadFile(rule.Response.Template)
				if err != nil {
					s.Logger.WithPrefix("GALAH").Error(err)
				} else {
					response = resp
					respSource = "static"
					break
				}
			}
		} else if err != nil {
			s.Logger.WithPrefix("GALAH").Error(err)
		}
	}

	// Check if the response is already cached
	if response == nil {
		response, err = cache.CheckCache(s.Cache, r, port, s.CacheDuration)
		if err != nil {
			if errors.Is(err, cache.ErrCacheExpired) || errors.Is(err, cache.ErrCacheMiss) {
				s.Logger.WithPrefix("GALAH").Infof("cache check for %q: %s", r.URL.String(), err)
			} else {
				s.Logger.WithPrefix("GALAH").Error(err)
			}
		} else {
			respSource = "cache"
		}
	}

	// Generate response using the service's LLM handler
	if response == nil {
		if s.Service == nil {
			s.Logger.WithPrefix("GALAH").Error("service is not configured")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		response, err = s.Service.GenerateHTTPResponse(r, port)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		respSource = "llm"
	}

	var respData llm.JSONResponse
	if err := json.Unmarshal(response, &respData); err != nil {
		s.Logger.WithPrefix("GALAH").Errorf("error unmarshalling the JSON-encoded data: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	s.sendResponse(w, respData)
	s.Logger.WithPrefix("GALAH").Infof("sent the response to %s (source: %s)", r.RemoteAddr, respSource)

	// Asynchronously perform Suricata matching and event logging to avoid blocking the handler
	if s.Suricata != nil {
		go func(req *http.Request, body string, resp llm.JSONResponse, port, src string) {
			matches := safeMatch(s.Suricata, req, body)
			for _, m := range matches {
				s.Logger.WithPrefix("GALAH").Infof("Suricata SID=%s – %q", m.SID, m.Msg)
			}
			s.EventLogger.LogEvent(req, resp, port, src, matches)
		}(r, reqBodyStr, respData, port, respSource)
	} else {
		// No Suricata: just log the event immediately
		s.EventLogger.LogEvent(r, respData, port, respSource, nil)
	}
}

func (s *Server) extractPort(serverAddr string) string {
	_, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		port = ""
	}
	return port
}

func (s *Server) sendResponse(w http.ResponseWriter, response llm.JSONResponse) {
	for key, value := range response.Headers {
		if !isExcludedHeader(key) {
			w.Header().Set(key, value)
		}
	}

	if _, err := w.Write([]byte(response.Body)); err != nil {
		s.Logger.WithPrefix("GALAH").Errorf("error writing response: %s", err)
	}
}

func isExcludedHeader(headerKey string) bool {
	return ignoreHeaders[strings.ToLower(headerKey)]
}

// ListenForShutdownSignals handles graceful shutdown on receiving signals.
func (s *Server) ListenForShutdownSignals() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		s.Logger.WithPrefix("GALAH").Infof("received shutdown signal. shutting down servers...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for _, server := range s.Servers {
			if err := server.Shutdown(ctx); err != nil {
				s.Logger.WithPrefix("GALAH").Errorf("error shutting down server: %s", err)
			}
		}

		s.Logger.WithPrefix("GALAH").Info("all servers shut down gracefully.")
		os.Exit(0)
	}()
}

// getInterfaceIP retrieves the IPv4 address of the specified network interface.
func getInterfaceIP(ifaceName string) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		switch v := address.(type) {
		case *net.IPNet:
			ip := v.IP
			if ip.To4() != nil && !ip.IsLoopback() {
				return ip.String(), nil
			}
		case *net.IPAddr:
			ip := v.IP
			if ip.To4() != nil && !ip.IsLoopback() {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no non-loopback addresses found for interface: %s", ifaceName)
}
