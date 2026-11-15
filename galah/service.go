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
	"github.com/0x4d31/galah/pkg/suricata"
	cblog "github.com/charmbracelet/log"
	"github.com/tmc/langchaingo/llms"
	"golang.org/x/sync/singleflight"
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
	// GenerationTimeout bounds one LLM generation. Generations are detached from the client's request, so a
	// scanner that hangs up doesn't cancel them: the response is still cached for the next visitor. 0 = 2m.
	GenerationTimeout time.Duration
	// MaxGenerations caps LLM generations running at once. Detached generations survive the client, so without a
	// cap a burst of unique URLs from clients that hang up would queue unbounded work (and cost) on the model.
	// 0 = DefaultMaxGenerations, negative = no cap.
	MaxGenerations  int
	ConfigFile      string
	RulesConfigFile string
	EventLogFile    string
	CacheDBFile     string
	CacheDuration   int
	LogLevel        string
	Logger          *cblog.Logger
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
	// GenerationTimeout bounds one detached generation (0 = DefaultGenerationTimeout).
	GenerationTimeout time.Duration

	flight singleflight.Group // one generation per cache key at a time
	slots  chan struct{}      // MaxGenerations; nil = no cap
}

// ErrGenerationBusy is returned when MaxGenerations generations are already running.
var ErrGenerationBusy = errors.New("generation skipped: the maximum number of generations is already running")

// DefaultGenerationTimeout is used when Options.GenerationTimeout is 0.
const DefaultGenerationTimeout = 2 * time.Minute

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
		Cache:             cacheDB,
		CacheDuration:     opts.CacheDuration,
		Config:            cfg,
		Rules:             rules,
		EventLogger:       eventLogger,
		LLMConfig:         modelCfg,
		Logger:            logger,
		Model:             model,
		GenerationTimeout: opts.GenerationTimeout,
		slots:             newSlots(opts.MaxGenerations),
	}, nil
}

// DefaultMaxGenerations is the cap used when Options.MaxGenerations is 0. Upgrading must not turn a disconnect
// (which used to cancel the work) into unbounded detached generations, so the default is a small cap, not none.
const DefaultMaxGenerations = 4

func newSlots(n int) chan struct{} {
	if n < 0 {
		return nil // no cap
	}
	if n == 0 {
		n = DefaultMaxGenerations
	}
	return make(chan struct{}, n)
}

// GenerateHTTPResponse creates an HTTP response using the LLM.
//
// The generation runs detached from the client's request: if the client hangs up (scanners often give up after a
// few seconds, and a cold model can take longer), it still completes and is cached for the next request for the
// same port and URL. Concurrent requests for the same key share one generation instead of each starting their own.
func (s *Service) GenerateHTTPResponse(r *http.Request, port string) ([]byte, error) {
	messages, err := llm.CreateMessageContent(r, s.Config, s.LLMConfig.Provider)
	if err != nil {
		s.Logger.WithPrefix("GALAH").Errorf("error creating llm message: %s", err)
		return nil, err
	}
	resp, err, shared := s.generate(r, port, messages)
	if shared {
		s.Logger.WithPrefix("GALAH").Infof("joined the generation already running for %q", r.URL.String())
	}
	if r.Context().Err() != nil && err == nil {
		s.Logger.WithPrefix("GALAH").Infof("client left before the response was ready; cached it for the next request to %q", r.URL.String())
	}
	return resp, err
}

// RefreshInBackground regenerates the response for r's cache key without making the caller wait (used to serve a
// stale cached response and refresh it). The request's body and headers are read now, before the handler returns.
func (s *Service) RefreshInBackground(r *http.Request, port string) {
	messages, err := llm.CreateMessageContent(r, s.Config, s.LLMConfig.Provider)
	if err != nil {
		s.Logger.WithPrefix("GALAH").Errorf("error creating llm message for refresh: %s", err)
		return
	}
	go func() {
		if _, err, _ := s.generate(r, port, messages); err == nil {
			s.Logger.WithPrefix("GALAH").Infof("refreshed the cached response for %q", r.URL.String())
		}
	}()
}

// generate runs one LLM generation per cache key at a time, detached from the request, and caches the result.
func (s *Service) generate(r *http.Request, port string, messages []llms.MessageContent) ([]byte, error, bool) {
	key := cache.GetCacheKey(r, port)
	v, err, shared := s.flight.Do(key, func() (any, error) {
		if s.slots != nil {
			select {
			case s.slots <- struct{}{}:
				defer func() { <-s.slots }()
			default:
				s.Logger.WithPrefix("GALAH").Warnf("%s (%d) for %q", ErrGenerationBusy, cap(s.slots), r.URL.String())
				return nil, ErrGenerationBusy
			}
		}
		timeout := s.GenerationTimeout
		if timeout <= 0 {
			timeout = DefaultGenerationTimeout
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), timeout)
		defer cancel()

		respStr, err := llm.GenerateLLMResponse(ctx, s.Model, s.LLMConfig.Temperature, messages)
		if err != nil {
			s.Logger.WithPrefix("GALAH").Errorf("error generating response: %s", err)
			s.EventLogger.LogError(r, respStr, port, err)
			return nil, err
		}
		resp := []byte(respStr)
		s.Logger.WithPrefix("GALAH").Infof("generated HTTP response: %s", strings.ReplaceAll(respStr, "\n", " "))
		if s.CacheDuration != 0 {
			if err := cache.StoreResponse(s.Cache, key, resp); err != nil {
				s.Logger.WithPrefix("GALAH").Errorf("error storing response in cache: %s", err)
			}
		}
		return resp, nil
	})
	if err != nil {
		return nil, err, shared
	}
	return v.([]byte), nil, shared
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
