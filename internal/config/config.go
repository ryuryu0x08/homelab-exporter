package config

import "time"

// Platform identifies the host operating system supported by a configuration.
type Platform string

const (
	PlatformWindows Platform = "windows"
)

// Dependency controls whether a failed source makes the unified scrape fail.
type Dependency string

const (
	DependencyOptional Dependency = "optional"
	DependencyRequired Dependency = "required"
)

// Source describes one local Prometheus exposition endpoint.
type Source struct {
	Name       string
	URL        string
	Dependency Dependency
}

// Config is the validated runtime configuration.
type Config struct {
	Platform      Platform
	Listen        string
	ScrapeTimeout time.Duration
	MaxBodyBytes  int64
	Sources       []Source
}
