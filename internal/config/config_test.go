package config_test

import (
	"testing"

	"github.com/0x4d31/galah/internal/config"
)

const configPath = "../../config/config.yaml"

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		wantErr    bool
	}{
		{
			name:       "validConfiguration",
			configPath: configPath,
			wantErr:    false,
		},
		{
			name:       "nonExistentConfiguration",
			configPath: "path_to_non_existent_config.yaml",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.LoadConfig(tt.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
