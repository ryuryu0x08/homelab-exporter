package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type rawConfig struct {
	Platform      Platform    `toml:"platform"`
	Listen        string      `toml:"listen"`
	ScrapeTimeout string      `toml:"scrape_timeout"`
	MaxBodyBytes  int64       `toml:"max_body_bytes"`
	Sources       []rawSource `toml:"source"`
}

type rawSource struct {
	Name       string     `toml:"name"`
	Kind       SourceKind `toml:"kind"`
	URL        string     `toml:"url"`
	Dependency Dependency `toml:"dependency"`
}

// Parse decodes and validates TOML configuration.
func Parse(data []byte) (*Config, error) {
	var raw rawConfig
	err := toml.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("parse config TOML: %w", err)
	}

	timeout, err := time.ParseDuration(raw.ScrapeTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse scrape_timeout: %w", err)
	}

	config := &Config{
		Platform:      raw.Platform,
		Listen:        strings.TrimSpace(raw.Listen),
		ScrapeTimeout: timeout,
		MaxBodyBytes:  raw.MaxBodyBytes,
		Sources:       make([]Source, 0, len(raw.Sources)),
	}
	for _, source := range raw.Sources {
		if source.Kind == "" {
			source.Kind = SourceKindHTTP
		}
		config.Sources = append(config.Sources, Source{
			Name:       strings.TrimSpace(source.Name),
			Kind:       source.Kind,
			URL:        strings.TrimSpace(source.URL),
			Dependency: source.Dependency,
		})
	}

	err = validate(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func validate(config *Config) error {
	var errs []error
	if config.Platform != PlatformWindows {
		errs = append(errs, fmt.Errorf("unsupported platform %q", config.Platform))
	}
	if config.Listen == "" {
		errs = append(errs, errors.New("listen is empty"))
	}
	if config.ScrapeTimeout <= 0 {
		errs = append(errs, errors.New("scrape_timeout must be positive"))
	}
	if config.MaxBodyBytes <= 0 {
		errs = append(errs, errors.New("max_body_bytes must be positive"))
	}
	if len(config.Sources) == 0 {
		errs = append(errs, errors.New("no sources configured"))
	}

	seen := make(map[string]struct{}, len(config.Sources))
	for index, source := range config.Sources {
		err := validateSource(source, seen)
		if err != nil {
			errs = append(errs, fmt.Errorf("source[%d]: %w", index, err))
			continue
		}
		seen[source.Name] = struct{}{}
	}
	return errors.Join(errs...)
}

func validateSource(source Source, seen map[string]struct{}) error {
	if source.Name == "" {
		return errors.New("name is empty")
	}
	if _, exists := seen[source.Name]; exists {
		return fmt.Errorf("duplicate source %q", source.Name)
	}
	if source.Dependency != DependencyOptional && source.Dependency != DependencyRequired {
		return fmt.Errorf("dependency %q is invalid", source.Dependency)
	}
	if source.Kind == SourceKindNVIDIASMI {
		if source.URL != "" {
			return errors.New("nvidia_smi source URL must be empty")
		}
		return nil
	}
	if source.Kind != SourceKindHTTP {
		return fmt.Errorf("source kind %q is invalid", source.Kind)
	}
	parsedURL, err := url.ParseRequestURI(source.URL)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme %q is unsupported", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return errors.New("URL host is empty")
	}
	return nil
}
