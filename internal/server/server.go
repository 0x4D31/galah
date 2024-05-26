package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/0x4d31/galah/internal/cache"
	"github.com/0x4d31/galah/internal/config"
	"github.com/0x4d31/galah/internal/logger"
	"github.com/0x4d31/galah/pkg/llm"
	"github.com/google/gopacket/pcap"
	"github.com/sirupsen/logrus"
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

type Server struct {
	Cache         *sql.DB
	CacheDuration int
	Interface     string
	Config        *config.Config
	EventLogger   *logger.Logger
	LLMConfig     llm.Config
	Logger        *logrus.Logger
	Model         llms.Model
	Servers       map[uint16]*http.Server
}

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
		s.Logger.Errorf("error starting server on port %d: %s", pc.Port, err)
		return err
	}

	mu.Lock()
	s.Servers[pc.Port] = server
	mu.Unlock()

	return nil
}

func (s *Server) SetupServer(pc config.PortConfig) *http.Server {
	var ip string
	var err error

	if s.Interface != "" {
		ip, err = getInterfaceIP(s.Interface)
		if err != nil {
			s.Logger.Errorln(err)
		}
	}
	serverAddr := net.JoinHostPort(ip, fmt.Sprintf("%d", pc.Port))

	return &http.Server{
		Addr: serverAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.handleRequest(w, r, serverAddr)
		}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func (s *Server) StartTLSServer(server *http.Server, pc config.PortConfig) error {
	if pc.TLSProfile == "" {
		return fmt.Errorf("TLS profile is not configured for port %d", pc.Port)
	}

	tlsConfig, ok := s.Config.Profiles[pc.TLSProfile]
	if !ok || tlsConfig.Certificate == "" || tlsConfig.Key == "" {
		return fmt.Errorf("TLS profile is incomplete for port %d", pc.Port)
	}

	s.Logger.Infof("starting HTTPS server on port %d with TLS profile: %s", pc.Port, pc.TLSProfile)
	return server.ListenAndServeTLS(tlsConfig.Certificate, tlsConfig.Key)
}

func (s *Server) StartHTTPServer(server *http.Server, pc config.PortConfig) error {
	s.Logger.Infof("starting HTTP server on port %d", pc.Port)
	return server.ListenAndServe()
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request, serverAddr string) {
	port := s.extractPort(serverAddr)
	s.Logger.Infof("port %s received a request for %q, from source %s", port, r.URL.String(), r.RemoteAddr)

	response, err := cache.CheckCache(s.Cache, r, port, s.CacheDuration)
	if err != nil {
		if errors.Is(err, cache.ErrCacheExpired) || errors.Is(err, cache.ErrCacheMiss) {
			s.Logger.Infof("Cache check for %q: %s", r.URL.String(), err)
		} else {
			s.Logger.Error(err)
		}
	}

	if response == nil {
		response, err = s.generateResponse(r, port)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	var respData llm.JSONResponse
	if err := json.Unmarshal(response, &respData); err != nil {
		s.Logger.Errorf("error unmarshalling the json-encoded data: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	s.sendResponse(w, respData)
	s.Logger.Infof("sent the generated response to %s", r.RemoteAddr)
	s.EventLogger.LogEvent(r, respData, port)
}

func (s *Server) extractPort(serverAddr string) string {
	_, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		port = ""
	}
	return port
}

func (s *Server) generateResponse(r *http.Request, port string) ([]byte, error) {
	messages, err := llm.CreateMessageContent(r, s.Config, s.LLMConfig.Provider)
	if err != nil {
		s.Logger.Errorf("error creating llm message: %s", err)
		return nil, err
	}

	responseString, err := llm.GenerateLLMResponse(r.Context(), s.Model, s.LLMConfig.Temperature, messages)
	if err != nil {
		s.Logger.Errorf("error generating response: %s", err)
		s.EventLogger.LogError(r, responseString, port, err)
		return nil, err
	}
	response := []byte(responseString)

	s.Logger.Infof("generated HTTP response: %s", strings.ReplaceAll(responseString, "\n", " "))

	// Store the response if caching is enabled
	if s.CacheDuration != 0 {
		key := cache.GetCacheKey(r, port)
		if err := cache.StoreResponse(s.Cache, key, response); err != nil {
			s.Logger.Errorf("error storing response in cache: %s", err)
		}
	}

	return response, nil
}

func (s *Server) sendResponse(w http.ResponseWriter, response llm.JSONResponse) {
	for key, value := range response.Headers {
		if !isExcludedHeader(key) {
			w.Header().Set(key, value)
		}
	}

	if _, err := w.Write([]byte(response.Body)); err != nil {
		s.Logger.Errorf("error writing response: %s", err)
	}
}

func isExcludedHeader(headerKey string) bool {
	return ignoreHeaders[strings.ToLower(headerKey)]
}

func (s *Server) ListenForShutdownSignals() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		s.Logger.Infof("received shutdown signal. shutting down servers...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for _, server := range s.Servers {
			if err := server.Shutdown(ctx); err != nil {
				s.Logger.Errorf("error shutting down server: %s", err)
			}
		}

		s.Logger.Infoln("all servers shut down gracefully.")
		os.Exit(0)
	}()
}

// getInterfaceIP retrieves the IPv4 address of the specified network interface.
func getInterfaceIP(ifaceName string) (string, error) {
	ifs, err := pcap.FindAllDevs()
	if err != nil {
		return "", err
	}

	for _, iface := range ifs {
		if iface.Name == ifaceName {
			for _, address := range iface.Addresses {
				ip := address.IP
				if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
					return ip.String(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no non-loopback addresses found for interface: %s", ifaceName)
}
