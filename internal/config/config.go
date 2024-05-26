package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration file settings for the application.
type Config struct {
	SystemPrompt string               `yaml:"system_prompt"`
	UserPrompt   string               `yaml:"user_prompt"`
	Ports        []PortConfig         `yaml:"ports"`
	Profiles     map[string]TLSConfig `yaml:"profiles"`
}

// TLSConfig contains TLS-related settings.
type TLSConfig struct {
	Certificate string `yaml:"certificate"`
	Key         string `yaml:"key"`
}

// PortConfig specifies honeypot port settings.
type PortConfig struct {
	Port       uint16 `yaml:"port"`
	Protocol   string `yaml:"protocol"`
	TLSProfile string `yaml:"tls_profile,omitempty"`
}

// LoadConfig reads and parses the configuration file.
func LoadConfig(file string) (*Config, error) {
	var config *Config

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
