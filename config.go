package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PromptTemplate string               `yaml:"prompt_template"`
	CacheDuration  int                  `yaml:"cache_duration"`
	Ports          []PortConfig         `yaml:"ports"`
	TLS            map[string]TLSConfig `yaml:"tls"`
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

func LoadConfig(file string) (Config, error) {
	var config Config

	data, err := os.ReadFile(file)
	if err != nil {
		return Config{}, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
