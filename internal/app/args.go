package app

var args struct {
	LLMProvider      string  `arg:"-p,--provider,env:LLM_PROVIDER,required" help:"LLM provider (openai, googleai, gcp-vertex, anthropic, cohere, ollama)"`
	LLMModel         string  `arg:"-m,--model,env:LLM_MODEL,required" help:"LLM model (e.g. gpt-3.5-turbo-1106, gemini-1.5-pro-preview-0409)"`
	LLMServerURL     string  `arg:"-u,--server-url,env:LLM_SERVER_URL" help:"LLM Server URL (required for Ollama)"`
	LLMTemperature   float64 `arg:"-t,--temperature,env:LLM_TEMPERATURE" help:"LLM sampling temperature (0-2). Higher values make the output more random" default:"1"`
	LLMAPIKey        string  `arg:"-k,--api-key,env:LLM_API_KEY" help:"LLM API Key"`
	LLMCloudLocation string  `arg:"--cloud-location,env:LLM_CLOUD_LOCATION" help:"LLM cloud location region (required for GCP's Vertex AI)"`
	LLMCloudProject  string  `arg:"--cloud-project,env:LLM_CLOUD_PROJECT" help:"LLM cloud project ID (required for GCP's Vertex AI)"`
	Interface        string  `arg:"-i,--interface" help:"interface to serve on"`
	ConfigFile       string  `arg:"-c,--config-file" help:"Path to config file" default:"config/config.yaml"`
	RulesConfigFile  string  `arg:"-r,--rules-config-file" help:"Path to rules config file" default:"config/rules.yaml"`
	EventLogFile     string  `arg:"-o,--event-log-file" help:"Path to event log file" default:"event_log.json"`
	CacheDBFile      string  `arg:"-f,--cache-db-file" help:"Path to database file for response caching" default:"cache.db"`
	CacheDuration    int     `arg:"-d,--cache-duration" help:"Cache duration for generated responses (in hours). Use 0 to disable caching, and -1 for unlimited caching (no expiration)." default:"24"`
	LogLevel         string  `arg:"-l,--log-level" help:"Log level (debug, info, error, fatal)" default:"info"`
}
