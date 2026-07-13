package config

import (
	"fmt"
	"os"
)

// Load reads and parses a configuration file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	config, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}
	return config, nil
}
