package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name             string   `yaml:"name"`
	Enabled          bool     `yaml:"enabled"`
	HTTPRequestRegex string   `yaml:"http_request_regex"`
	Response         Response `yaml:"response"`
}

type Response struct {
	Type     string `yaml:"type"`
	Template string `yaml:"template"`
}

type RulesConfig struct {
	Rules []Rule `yaml:"rules"`
}

func LoadRules(file string) (*RulesConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var rules RulesConfig
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, err
	}

	return &rules, nil
}
