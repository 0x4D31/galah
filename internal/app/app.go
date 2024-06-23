package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/0x4d31/galah/internal/cache"
	"github.com/0x4d31/galah/internal/config"
	el "github.com/0x4d31/galah/internal/logger"
	"github.com/0x4d31/galah/internal/server"
	"github.com/0x4d31/galah/pkg/enrich"
	"github.com/0x4d31/galah/pkg/llm"
	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

// App contains the core components and dependencies of the application.
type App struct {
	Cache       *sql.DB
	Config      *config.Config
	Rules       []config.Rule
	EnrichCache *enrich.Enricher
	EventLogger *el.Logger
	Hostname    string
	LLMConfig   llm.Config
	Logger      *logrus.Logger
	Model       llms.Model
	Servers     map[uint16]*http.Server
}

var logger *logrus.Logger

const (
	version         = "1.0"
	lookupCacheSize = 1_000_000
	lookupTTL       = 1 * time.Hour
	// sessionTTL = 2 * time.Minute
)

// Run starts the app with the provided configuration.
func (a *App) Run() error {
	printBanner()

	logger = logrus.New()
	arg.MustParse(&args)

	if err := logLevel(args.LogLevel); err != nil {
		logger.Fatalf("error setting log level: %s", err)
	}

	err := a.init()
	if err != nil {
		logger.Fatalf("error initializing app: %s", err)
	}

	srv := server.Server{
		Cache:         a.Cache,
		CacheDuration: args.CacheDuration,
		Interface:     args.Interface,
		Config:        a.Config,
		Rules:         a.Rules,
		EventLogger:   a.EventLogger,
		LLMConfig:     a.LLMConfig,
		Logger:        a.Logger,
		Model:         a.Model,
	}

	srv.ListenForShutdownSignals()
	if err := srv.StartServers(); err != nil {
		logger.Fatalf("application failed to start: %s", err)
	}

	return nil
}

func (a *App) init() error {
	ctx := context.Background()

	cfg, err := config.LoadConfig(args.ConfigFile)
	if err != nil {
		return fmt.Errorf("error loading config: %s", err)
	}

	rulesConfig, err := config.LoadRules(args.RulesConfigFile)
	if err != nil {
		return fmt.Errorf("error loading rules config: %s", err)
	}

	modelConfig := llm.Config{
		Provider:      args.LLMProvider,
		Model:         args.LLMModel,
		ServerURL:     args.LLMServerURL,
		Temperature:   args.LLMTemperature,
		APIKey:        args.LLMAPIKey,
		CloudProject:  args.LLMCloudProject,
		CloudLocation: args.LLMCloudLocation,
	}
	model, err := llm.New(ctx, modelConfig)
	if err != nil {
		return fmt.Errorf("error initializing the LLM client: %s", err)
	}

	cache, err := cache.InitializeCache(args.CacheDBFile)
	if err != nil {
		return fmt.Errorf("error initializing the cache database: %s", err)
	}

	enrichCache := enrich.New(enrich.Config{
		CacheSize: lookupCacheSize,
		CacheTTL:  lookupTTL,
	})

	eventLogger, err := el.New(args.EventLogFile, modelConfig, enrichCache, logger)
	if err != nil {
		return err
	}

	a.Cache = cache
	a.Config = cfg
	a.Rules = rulesConfig.Rules
	a.EnrichCache = enrichCache
	a.EventLogger = eventLogger
	a.LLMConfig = modelConfig
	a.Logger = logger
	a.Model = model
	a.Servers = make(map[uint16]*http.Server)

	return nil
}

func logLevel(level string) error {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logger.SetLevel(l)
	return nil
}

func printBanner() {
	banner := `
 ██████   █████  ██       █████  ██   ██ 
██       ██   ██ ██      ██   ██ ██   ██ 
██   ███ ███████ ██      ███████ ███████ 
██    ██ ██   ██ ██      ██   ██ ██   ██ 
 ██████  ██   ██ ███████ ██   ██ ██   ██ 
  llm-based web honeypot // version %s
  	author: Adel "0x4D31" Karimi

`
	fmt.Printf(banner, version)
	return
}
