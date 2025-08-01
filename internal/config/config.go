package config

import (
	"fmt"
	"os"

	"github.com/pedrospdc/gespann/internal/adapters"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel string            `yaml:"log_level"`
	Adapters []adapters.Config `yaml:"adapters"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.LogLevel == "" {
		config.LogLevel = "info"
	}

	return &config, nil
}

func Default() *Config {
	return &Config{
		LogLevel: "info",
		Adapters: []adapters.Config{
			{
				Type: "prometheus",
				Settings: map[string]string{
					"port": "8080",
				},
			},
		},
	}
}
