package galah

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/0x4d31/galah/internal/cache"
	"github.com/0x4d31/galah/internal/config"
	el "github.com/0x4d31/galah/internal/logger"
	"github.com/0x4d31/galah/pkg/enrich"
	"github.com/0x4d31/galah/pkg/llm"
	cblog "github.com/charmbracelet/log"
	"github.com/0x4d31/galah/pkg/suricata"
	"github.com/tmc/langchaingo/llms"
)

const (
	cacheSize  = 1_000_000
	lookupTTL  = 1 * time.Hour
	sessionTTL = 2 * time.Minute

	// Default file paths used when Options fields are left empty.
	// Leaving RulesConfigFile empty disables rule checking entirely.
	DefaultConfigFile      = "config/config.yaml"
	DefaultRulesConfigFile = ""
	DefaultCacheDBFile     = "cache.db"
	DefaultEventLogFile    = "event_log.json"
)

// Options defines the configuration for creating a Service.
//
// If ConfigFile, EventLogFile, or CacheDBFile are empty, NewService falls back
// to DefaultConfigFile, DefaultEventLogFile, and DefaultCacheDBFile
// respectively. Leaving RulesConfigFile empty disables the rule engine
// entirely.
type Options struct {
	LLMProvider      string
	LLMModel         string
	LLMServerURL     string
	LLMTemperature   float64
	LLMAPIKey        string
	LLMCloudProject  string
	LLMCloudLocation string
	ConfigFile       string
	RulesConfigFile  string
	EventLogFile     string
	CacheDBFile      string
	CacheDuration    int
	LogLevel         string
	Logger           *cblog.Logger
}

// Service encapsulates the components required to generate HTTP responses.
type Service struct {
	Cache         *sql.DB
	CacheDuration int
	Config        *config.Config
	Rules         []config.Rule
	EventLogger   *el.Logger
	LLMConfig     llm.Config
	Logger        *cblog.Logger
	Model         llms.Model
}

// NewService loads configuration and initializes the components required for response generation.
func NewService(ctx context.Context, opts Options) (*Service, error) {
	if opts.LogLevel != "" {
		level, err := cblog.ParseLevel(opts.LogLevel)
		if err == nil {
			cblog.SetLevel(level)
		}
	}
	cblog.SetPrefix("GALAH")
	cblog.SetTimeFormat("2006/01/02 15:04:05")

	logger := opts.Logger
	if logger == nil {
		logger = cblog.Default()
	}

	if opts.ConfigFile == "" {
		opts.ConfigFile = DefaultConfigFile
	}
	if opts.CacheDBFile == "" {
		opts.CacheDBFile = DefaultCacheDBFile
	}
	if opts.EventLogFile == "" {
		opts.EventLogFile = DefaultEventLogFile
	}

	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	var rules []config.Rule
	if opts.RulesConfigFile != "" {
		rulesCfg, err := config.LoadRules(opts.RulesConfigFile)
		if err != nil {
			return nil, fmt.Errorf("error loading rules config: %w", err)
		}
		rules = rulesCfg.Rules
	}

	return createService(ctx, cfg, rules, opts, logger)
}

// NewServiceFromConfig initializes a Service using the provided configuration
// and rule set. The ConfigFile and RulesConfigFile values from opts are ignored.
func NewServiceFromConfig(ctx context.Context, cfg *config.Config, rules []config.Rule, opts Options) (*Service, error) {
	if opts.LogLevel != "" {
		level, err := cblog.ParseLevel(opts.LogLevel)
		if err == nil {
			cblog.SetLevel(level)
		}
	}
	cblog.SetPrefix("GALAH")
	cblog.SetTimeFormat("2006/01/02 15:04:05")

	logger := opts.Logger
	if logger == nil {
		logger = cblog.Default()
	}

	if opts.ConfigFile == "" {
		opts.ConfigFile = DefaultConfigFile
	}
	if opts.CacheDBFile == "" {
		opts.CacheDBFile = DefaultCacheDBFile
	}
	if opts.EventLogFile == "" {
		opts.EventLogFile = DefaultEventLogFile
	}

	return createService(ctx, cfg, rules, opts, logger)
}

func createService(ctx context.Context, cfg *config.Config, rules []config.Rule, opts Options, logger *cblog.Logger) (*Service, error) {
	modelCfg := llm.Config{
		Provider:      opts.LLMProvider,
		Model:         opts.LLMModel,
		ServerURL:     opts.LLMServerURL,
		Temperature:   opts.LLMTemperature,
		APIKey:        opts.LLMAPIKey,
		CloudProject:  opts.LLMCloudProject,
		CloudLocation: opts.LLMCloudLocation,
	}

	model, err := llm.New(ctx, modelCfg)
	if err != nil {
		return nil, fmt.Errorf("error initializing the LLM client: %w", err)
	}

	cacheDB, err := cache.InitializeCache(opts.CacheDBFile)
	if err != nil {
		return nil, fmt.Errorf("error initializing the cache database: %w", err)
	}

	enrichCache := enrich.New(enrich.Config{CacheSize: cacheSize, CacheTTL: lookupTTL})
	sessionizer := el.NewSessionizer(el.Config{CacheSize: cacheSize, CacheTTL: sessionTTL})

	eventLogger, err := el.New(opts.EventLogFile, modelCfg, enrichCache, sessionizer, logger)
	if err != nil {
		return nil, err
	}

	return &Service{
		Cache:         cacheDB,
		CacheDuration: opts.CacheDuration,
		Config:        cfg,
		Rules:         rules,
		EventLogger:   eventLogger,
		LLMConfig:     modelCfg,
		Logger:        logger,
		Model:         model,
	}, nil
}

// GenerateHTTPResponse creates an HTTP response using the LLM.
func (s *Service) GenerateHTTPResponse(r *http.Request, port string) ([]byte, error) {
	messages, err := llm.CreateMessageContent(r, s.Config, s.LLMConfig.Provider)
	if err != nil {
		s.Logger.WithPrefix("GALAH").Errorf("error creating llm message: %s", err)
		return nil, err
	}

	respStr, err := llm.GenerateLLMResponse(r.Context(), s.Model, s.LLMConfig.Temperature, messages)
	if err != nil {
		s.Logger.WithPrefix("GALAH").Errorf("error generating response: %s", err)
		s.EventLogger.LogError(r, respStr, port, err)
		return nil, err
	}
	resp := []byte(respStr)

	s.Logger.WithPrefix("GALAH").Infof("generated HTTP response: %s", strings.ReplaceAll(respStr, "\n", " "))

	if s.CacheDuration != 0 {
		key := cache.GetCacheKey(r, port)
		if err := cache.StoreResponse(s.Cache, key, resp); err != nil {
			s.Logger.WithPrefix("GALAH").Errorf("error storing response in cache: %s", err)
		}
	}

	return resp, nil
}

// CheckCache verifies if a response for the given request and port exists in
// the service's cache. It returns the cached response bytes if found and valid.
func (s *Service) CheckCache(r *http.Request, port string) ([]byte, error) {
	return cache.CheckCache(s.Cache, r, port, s.CacheDuration)
}

// LogEvent writes a successfulResponse entry to the configured event log file.
// Suricata rule matches can optionally be provided.
func (s *Service) LogEvent(r *http.Request, resp llm.JSONResponse, port, respSource string, suricataMatches []suricata.Rule) {
	s.EventLogger.LogEvent(r, resp, port, respSource, suricataMatches)
}

// LogError writes a failedResponse entry to the configured event log file.
func (s *Service) LogError(r *http.Request, resp, port string, err error) {
	s.EventLogger.LogError(r, resp, port, err)
}

// Close releases resources held by the Service.
func (s *Service) Close() error {
	var errs []error
	if s.Cache != nil {
		if err := s.Cache.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.EventLogger != nil && s.EventLogger.EventFile != nil {
		if err := s.EventLogger.EventFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
