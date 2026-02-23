package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RootConfig struct {
	Alias string `yaml:"alias"`
	Path  string `yaml:"path"`
}

type Settings struct {
	FinancialYearStart int `yaml:"financial_year_start"`
	HashLength         int `yaml:"hash_length"`
}

type Config struct {
	Roots    []RootConfig `yaml:"roots"`
	Settings Settings     `yaml:"settings"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func (c *Config) GetRootPath(alias string) (string, error) {
	for _, root := range c.Roots {
		if root.Alias == alias {
			return root.Path, nil
		}
	}
	return "", fmt.Errorf("root alias '%s' not found in config", alias)
}
