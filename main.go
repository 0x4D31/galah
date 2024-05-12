package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/0x4d31/galah/enrich"
	"github.com/alexflint/go-arg"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tmc/langchaingo/llms"
)

type App struct {
	Client      llms.Model
	Config      *Config
	DB          *sql.DB
	EnrichCache *enrich.Default
	Servers     map[uint16]*http.Server
	OutputFile  string
	Hostname    string
	LLMConfig   LLMConfig
}

var args struct {
	LLMAPIKey        string `arg:"-k,--api-key,env:LLM_API_KEY,required" help:"LLM API Key"`
	LLMProvider      string `arg:"-p,--provider,env:LLM_PROVIDER" help:"LLM provider" default:"openai"`
	LLMModel         string `arg:"-m,--model,env:LLM_MODEL" help:"LLM model" default:"gpt-3.5-turbo-1106"`
	LLMCloudProject  string `arg:"--cloud-project,env:LLM_CLOUD_PROJECT" help:"LLM cloud project (e.g. for GCP Vertex)"`
	LLMCloudLocation string `arg:"--cloud-location,env:LLM_CLOUD_LOCATION" help:"LLM cloud location (e.g. for GCP Vertex)"`
	Interface        string `arg:"-i,--interface" help:"Interface to serve on"`
	ConfigFile       string `arg:"-c,--config" help:"Path to config file" default:"config.yaml"`
	DBPath           string `arg:"-d,--database" help:"Path to database file for cache" default:"cache.db"`
	OutputFile       string `arg:"-o,--output" help:"Path to output log file" default:"log.json"`
	LogLevel         string `arg:"-l,--log-level" help:"Log level (debug, info, error, fatal)" default:"info"`
}

const (
	version   = "1.1"
	cacheSize = 1_000_000
	lookupTTL = 1 * time.Hour
	// sessionTTL = 2 * time.Minute
)

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

func main() {
	printBanner()
	arg.MustParse(&args)
	setLogLevel(args.LogLevel)
	ctx := context.Background()

	app, err := initializeApp()
	if err != nil {
		logger.Fatalln(err)
	}
	defer app.DB.Close()

	client, err := app.initializeLLMClient(ctx)
	if err != nil {
		logger.Fatalf("error initializing the LLM client: %s", err)
	}
	app.Client = client

	app.listenForShutdownSignals()
	if err := app.startServers(); err != nil {
		logger.Fatalf("application failed to start: %s", err)
	}
}

func initializeApp() (*App, error) {
	// Set the interface to the first non-loopback interface if not already specified.
	if args.Interface == "" {
		interfaceName, err := getDefaultInterface()
		if err != nil {
			return nil, fmt.Errorf("error getting default interface: %s", err)
		}
		args.Interface = interfaceName
	}

	config, err := LoadConfig(args.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %s", err)
	}

	db, err := initializeCache(args.DBPath)
	if err != nil {
		return nil, fmt.Errorf("error initializing the cache database: %s", err)
	}

	hostname, err := getHostname()
	if err != nil {
		return nil, fmt.Errorf("error getting the hostname: %s", err)
	}

	enrichCache := enrich.New(&enrich.Config{
		CacheSize: cacheSize,
		CacheTTL:  lookupTTL,
	})

	return &App{
		Config:      config,
		DB:          db,
		EnrichCache: enrichCache,
		OutputFile:  args.OutputFile,
		Hostname:    hostname,
		LLMConfig: LLMConfig{
			APIKey:        args.LLMAPIKey,
			Provider:      args.LLMProvider,
			Model:         args.LLMModel,
			CloudProject:  args.LLMCloudProject,
			CloudLocation: args.LLMCloudLocation,
		},
	}, nil
}
