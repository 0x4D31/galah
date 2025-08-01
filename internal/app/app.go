package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0x4d31/galah/galah"
	"github.com/0x4d31/galah/internal/cache"
	"github.com/0x4d31/galah/internal/config"
	el "github.com/0x4d31/galah/internal/logger"
	"github.com/0x4d31/galah/internal/server"
	"github.com/0x4d31/galah/pkg/enrich"
	"github.com/0x4d31/galah/pkg/llm"
	"github.com/0x4d31/galah/pkg/suricata"
	"github.com/alexflint/go-arg"
	cblog "github.com/charmbracelet/log"
	"github.com/tmc/langchaingo/llms"
)

// App contains the core components and dependencies of the application.
type App struct {
	Cache       *sql.DB
	Config      *config.Config
	Rules       []config.Rule
	EventLogger *el.Logger
	Hostname    string
	LLMConfig   llm.Config
	Logger      *cblog.Logger
	Model       llms.Model
	Service     *galah.Service
	Suricata    *suricata.RuleSet
	Servers     map[uint16]*http.Server
}

var logger *cblog.Logger

const (
	version    = "1.1.1"
	CacheSize  = 1_000_000
	lookupTTL  = 1 * time.Hour
	sessionTTL = 2 * time.Minute
)

// Run starts the app with the provided configuration.
func (a *App) Run() error {
	printBanner()

	// Set up global logger configuration
	cblog.SetPrefix("APP")
	cblog.SetTimeFormat("2006/01/02 15:04:05")
	arg.MustParse(&args)

	if err := logLevel(args.LogLevel); err != nil {
		cblog.Fatalf("error setting log level: %s", err)
	}

	err := a.init()
	if err != nil {
		cblog.Fatalf("error initializing app: %s", err)
	}

	a.Service, err = galah.NewServiceFromConfig(context.Background(), a.Config, a.Rules, galah.Options{
		LLMProvider:      a.LLMConfig.Provider,
		LLMModel:         a.LLMConfig.Model,
		LLMServerURL:     a.LLMConfig.ServerURL,
		LLMTemperature:   a.LLMConfig.Temperature,
		LLMAPIKey:        a.LLMConfig.APIKey,
		LLMCloudProject:  a.LLMConfig.CloudProject,
		LLMCloudLocation: a.LLMConfig.CloudLocation,
		EventLogFile:     args.EventLogFile,
		CacheDBFile:      args.CacheDBFile,
		CacheDuration:    args.CacheDuration,
		LogLevel:         args.LogLevel,
	})
	if err != nil {
		cblog.Fatalf("error creating service: %s", err)
	}

	srv := server.Server{
		Cache:         a.Cache,
		CacheDuration: args.CacheDuration,
		Interface:     args.Interface,
		Config:        a.Config,
		Rules:         a.Rules,
		EventLogger:   a.EventLogger,
		LLMConfig:     a.LLMConfig,
		Logger:        cblog.Default(),
		Model:         a.Model,
		Service:       a.Service,
		Suricata:      a.Suricata,
	}

	srv.ListenForShutdownSignals()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		if err := a.Service.Close(); err != nil {
			cblog.Errorf("error closing service: %s", err)
		}
	}()
	if err := srv.StartServers(); err != nil {
		cblog.Fatalf("application failed to start: %s", err)
	}

	return nil
}

func (a *App) init() error {
	ctx := context.Background()

	cfg, err := config.LoadConfig(args.ConfigFile)
	if err != nil {
		return fmt.Errorf("error loading config: %s", err)
	}

	var rules []config.Rule
	if args.RulesConfigFile != "" {
		rulesConfig, err := config.LoadRules(args.RulesConfigFile)
		if err != nil {
			return fmt.Errorf("error loading rules config: %s", err)
		}
		rules = rulesConfig.Rules
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
		CacheSize: CacheSize,
		CacheTTL:  lookupTTL,
	})

	sessionizer := el.NewSessionizer(el.Config{
		CacheSize: CacheSize,
		CacheTTL:  sessionTTL,
	})

	eventLogger, err := el.New(args.EventLogFile, modelConfig, enrichCache, sessionizer, cblog.Default())
	if err != nil {
		return err
	}

	a.Cache = cache
	a.Config = cfg
	a.Rules = rules
	a.EventLogger = eventLogger
	a.LLMConfig = modelConfig
	a.Logger = cblog.Default()
	a.Model = model
	a.Servers = make(map[uint16]*http.Server)

	// Optionally load Suricata HTTP rules
	if args.SuricataEnabled {
		if args.SuricataRulesDir == "" {
			return fmt.Errorf("suricata enabled but no --suricata-rules-dir provided")
		}
		rs := suricata.NewRuleSet()
		if err := rs.LoadRules(args.SuricataRulesDir); err != nil {
			return fmt.Errorf("error loading Suricata rules from %s: %w", args.SuricataRulesDir, err)
		}
		cblog.Infof("loaded %d Suricata rules from %s", len(rs.Rules), args.SuricataRulesDir)
		a.Suricata = rs
	}

	return nil
}

func logLevel(level string) error {
	lvl, err := cblog.ParseLevel(level)
	if err != nil {
		return err
	}
	cblog.SetLevel(lvl)
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
}
