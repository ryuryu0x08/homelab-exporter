package app

import (
	"context"
	"time"

	"github.com/ryuryu0x08/homelab-exporter/internal/aggregate"
	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

// Service binds runtime configuration to the aggregation core.
type Service struct {
	aggregator *aggregate.Aggregator
	sources    []config.Source
	timeout    time.Duration
}

// NewService constructs the configured aggregation service.
func NewService(aggregator *aggregate.Aggregator, sources []config.Source, timeout time.Duration) *Service {
	return &Service{aggregator: aggregator, sources: sources, timeout: timeout}
}

// Gather produces one bounded unified scrape.
func (s *Service) Gather(ctx context.Context) ([]byte, aggregate.GatherStatus) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.aggregator.Gather(ctx, s.sources)
}
