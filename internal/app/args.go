package app

var args struct {
	ConfigFile       string  `arg:"-c,--config-file" help:"Path to config file" default:"config/config.yaml"`
	DatabaseFile     string  `arg:"-d,--database-file" help:"Path to database file for response caching" default:"cache.db"`
	EventLogFile     string  `arg:"-o,--event-log-file" help:"Path to event log file" default:"event_log.json"`
	LogLevel         string  `arg:"-l,--log-level" help:"Log level (debug, info, error, fatal)" default:"info"`
	LLMAPIKey        string  `arg:"-k,--api-key,env:LLM_API_KEY" help:"LLM API Key"`
	LLMCloudLocation string  `arg:"--cloud-location,env:LLM_CLOUD_LOCATION" help:"LLM cloud location region (required for GCP Vertex)"`
	LLMCloudProject  string  `arg:"--cloud-project,env:LLM_CLOUD_PROJECT" help:"LLM cloud project ID (required for GCP Vertex)"`
	LLMModel         string  `arg:"-m,--model,env:LLM_MODEL,required" help:"LLM model (e.g. gpt-3.5-turbo-1106, gemini-1.5-pro-preview-0409)"`
	LLMProvider      string  `arg:"-p,--provider,env:LLM_PROVIDER,required" help:"LLM provider (openai, gcp-vertex)"`
	LLMTemperature   float64 `arg:"-t,--temperature,env:LLM_TEMPERATURE" help:"LLM sampling temperature (0-2). Higher values make the output more random" default:"1"`
}
