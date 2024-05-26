package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SystemPrompt string               `yaml:"system_prompt"`
	UserPrompt   string               `yaml:"user_prompt"`
	Ports        []PortConfig         `yaml:"ports"`
	Profiles     map[string]TLSConfig `yaml:"profiles"`
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
