package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	LLM            LLMConfig            `yaml:"llm"`
	PromptTemplate string               `yaml:"prompt_template"`
	CacheDuration  int                  `yaml:"cache_duration"`
	Ports          []PortConfig         `yaml:"ports"`
	TLS            map[string]TLSConfig `yaml:"tls"`
}

type LLMConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	APIKey   string `yaml:"api_key"`
}

type TLSConfig struct {
	Certificate string `yaml:"certificate"`
	Key         string `yaml:"key"`
}

type PortConfig struct {
	Port       uint16 `yaml:"port"`
	Protocol   string `yaml:"protocol"`
	TLSProfile string `yaml:"tls_profile,omitempty"`
}

func LoadConfig(file string) (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
